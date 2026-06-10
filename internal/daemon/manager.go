package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	registry *Registry
	sentinel string
	mu       sync.Mutex
	workers  map[string]*workerInfo
}

type workerInfo struct {
	cmd      *exec.Cmd
	done     chan struct{}
	cancel   chan struct{}
	crashed  bool
}

const (
	maxRestartDelay = 30 * time.Second
	baseRestartDelay = time.Second
	maxRestarts     = 5
)

func NewManager(registry *Registry) (*Manager, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find executable: %w", err)
	}
	return &Manager{
		registry: registry,
		sentinel: exe,
		workers:  make(map[string]*workerInfo),
	}, nil
}

func (m *Manager) StartWorkspace(name, workspacePath string) (*Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.workers[name]; ok {
		return nil, fmt.Errorf("workspace %q is already running", name)
	}

	w, err := m.registry.Register(name, workspacePath)
	if err != nil {
		return nil, err
	}

	if err := m.startWorker(name, w); err != nil {
		return nil, err
	}

	return w, nil
}

func (m *Manager) StopWorkspace(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.workers[name]
	if !ok {
		return fmt.Errorf("workspace %q is not running", name)
	}

	close(info.cancel)
	if info.cmd.Process != nil {
		info.cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-info.done:
	case <-time.After(10 * time.Second):
		if info.cmd.Process != nil {
			info.cmd.Process.Kill()
		}
	}

	m.registry.Update(name, func(w *Workspace) {
		w.Status = StatusStopped
		w.Pid = 0
	})

	delete(m.workers, name)
	return nil
}

func (m *Manager) RestartWorkspace(name string) error {
	w, ok := m.registry.Get(name)
	if !ok {
		return fmt.Errorf("workspace %q not found", name)
	}

	if info, running := m.workers[name]; running {
		close(info.cancel)
		if info.cmd.Process != nil {
			info.cmd.Process.Signal(syscall.SIGTERM)
		}
		<-info.done
		delete(m.workers, name)
	}

	m.registry.Update(name, func(w *Workspace) {
		w.Restarts++
	})

	return m.startWorker(name, w)
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, info := range m.workers {
		close(info.cancel)
		if info.cmd.Process != nil {
			info.cmd.Process.Signal(syscall.SIGTERM)
		}
		<-info.done
		m.registry.Update(name, func(w *Workspace) {
			w.Status = StatusStopped
			w.Pid = 0
		})
		delete(m.workers, name)
	}
}

func (m *Manager) ListWithStatus() []*Workspace {
	list := m.registry.List()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, w := range list {
		if _, ok := m.workers[w.Name]; ok {
			w.mu.Lock()
			w.Status = StatusRunning
			w.mu.Unlock()
		}
		w.ScanCount = m.loadScanCount(w.Name)
	}
	return list
}

func (m *Manager) GetWithStatus(name string) (*Workspace, bool) {
	w, ok := m.registry.Get(name)
	if !ok {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, running := m.workers[w.Name]; running {
		w.mu.Lock()
		w.Status = StatusRunning
		w.mu.Unlock()
	}
	w.ScanCount = m.loadScanCount(name)
	return w, true
}

func (m *Manager) loadScanCount(name string) int64 {
	statePath := filepath.Join(filepath.Dir(m.registry.Dir), "daemons", name, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return 0
	}
	var state struct {
		ScanCount int64 `json:"scan_count"`
	}
	if json.Unmarshal(data, &state) != nil {
		return 0
	}
	return state.ScanCount
}

func (m *Manager) startWorker(name string, w *Workspace) error {
	logDir := filepath.Join(filepath.Dir(m.registry.Dir), "daemons", name)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	stdout, err := os.OpenFile(filepath.Join(logDir, "stdout.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open stdout log: %w", err)
	}
	stderr, err := os.OpenFile(filepath.Join(logDir, "stderr.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		stdout.Close()
		return fmt.Errorf("open stderr log: %w", err)
	}

	cmd := exec.Command(m.sentinel, "_watch", "--workspace", w.Path, "--name", name)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("start worker: %w", err)
	}

	cancel := make(chan struct{})
	done := make(chan struct{})

	info := &workerInfo{
		cmd:    cmd,
		done:   done,
		cancel: cancel,
	}
	m.workers[name] = info

	m.registry.Update(name, func(w *Workspace) {
		w.Status = StatusRunning
		w.Pid = cmd.Process.Pid
		w.StartedAt = time.Now()
	})

	go func() {
		defer close(done)
		defer stdout.Close()
		defer stderr.Close()

		err := cmd.Wait()
		if err != nil && !isCanceled(cancel) {
			info.crashed = true
			m.registry.Update(name, func(w *Workspace) {
				w.Status = StatusCrashed
				w.Pid = 0
				w.LastError = err.Error()
			})
			go m.restartWithBackoff(name, w, cancel)
		}
	}()

	return nil
}

func (m *Manager) restartWithBackoff(name string, w *Workspace, cancel chan struct{}) {
	for attempt := 0; attempt < maxRestarts; attempt++ {
		select {
		case <-time.After(backoffDuration(attempt)):
		case <-cancel:
			return
		}

		m.mu.Lock()
		_, exists := m.workers[name]
		m.mu.Unlock()
		if !exists {
			return
		}

		m.registry.Update(name, func(w *Workspace) {
			w.Restarts++
			w.Status = StatusStarting
		})

		if err := m.startWorker(name, w); err != nil {
			continue
		}
		return
	}

	m.registry.Update(name, func(w *Workspace) {
		w.Status = StatusCrashed
		w.LastError = fmt.Sprintf("exceeded max restarts (%d)", maxRestarts)
	})
	m.mu.Lock()
	delete(m.workers, name)
	m.mu.Unlock()
}

func backoffDuration(attempt int) time.Duration {
	d := baseRestartDelay * time.Duration(1<<uint(attempt))
	if d > maxRestartDelay {
		return maxRestartDelay
	}
	return d
}

func isCanceled(cancel chan struct{}) bool {
	select {
	case <-cancel:
		return true
	default:
		return false
	}
}
