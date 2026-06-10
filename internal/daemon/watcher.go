package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	name      string
	workspace string
	fswatcher *fsnotify.Watcher
	debounce  time.Duration
	mu        sync.Mutex
	pending   map[string]time.Time
	stopCh    chan struct{}
	done      chan struct{}
	scanFn    ScanFunc
	scanCount int64
	stateDir  string
}

type ScanFunc func(workspace, filePath string) error

func NewWatcher(name, workspace string, scanFn ScanFunc) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	return &Watcher{
		name:      name,
		workspace: workspace,
		fswatcher: fsw,
		debounce:  500 * time.Millisecond,
		pending:   make(map[string]time.Time),
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
		scanFn:    scanFn,
		stateDir:  filepath.Join(workspace, ".gdc-sentinel", "daemons", name),
	}, nil
}

func (w *Watcher) Run() {
	defer close(w.done)
	defer w.fswatcher.Close()

	sentinelDir := filepath.Join(w.workspace, ".gdc-sentinel")
	if err := w.addWatchTargets(sentinelDir); err != nil {
		log.Printf("[watcher:%s] initial watch setup failed: %v", w.name, err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case sig := <-sigCh:
			log.Printf("[watcher:%s] received signal %v, shutting down", w.name, sig)
			return
		case event, ok := <-w.fswatcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}
			if w.shouldIgnore(event.Name) {
				continue
			}
			if event.Op&fsnotify.Create != 0 && w.isDir(event.Name) {
				w.fswatcher.Add(event.Name)
			}
			w.mu.Lock()
			w.pending[event.Name] = time.Now()
			w.mu.Unlock()
		case <-ticker.C:
			w.flushPending()
		case err, ok := <-w.fswatcher.Errors:
			if !ok {
				return
			}
			log.Printf("[watcher:%s] fsnotify error: %v", w.name, err)
		}
	}
}

func (w *Watcher) Stop() {
	close(w.stopCh)
	<-w.done
}

func (w *Watcher) addWatchTargets(sentinelDir string) error {
	dirs := []string{}

	filepath.Walk(w.workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if base == ".git" || base == "node_modules" || base == "vendor" || base == ".gdc-sentinel" {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})

	for _, d := range dirs {
		if err := w.fswatcher.Add(d); err != nil {
			log.Printf("[watcher:%s] cannot watch %s: %v", w.name, d, err)
		}
	}
	return nil
}

func (w *Watcher) shouldIgnore(name string) bool {
	ext := filepath.Ext(name)
	if ext == ".test" || ext == ".out" {
		return true
	}
	base := filepath.Base(name)
	if base[0] == '.' && base != ".go" {
		return true
	}
	return false
}

func (w *Watcher) isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func (w *Watcher) ScanCount() int64 {
	return atomic.LoadInt64(&w.scanCount)
}

type watcherState struct {
	Name      string `json:"name"`
	ScanCount int64  `json:"scan_count"`
}

func (w *Watcher) saveState() {
	os.MkdirAll(w.stateDir, 0755)
	state := watcherState{
		Name:      w.name,
		ScanCount: w.scanCount,
	}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(w.stateDir, "state.json"), data, 0644)
}

func (w *Watcher) flushPending() {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	var toScan []string
	for path, t := range w.pending {
		if now.Sub(t) >= w.debounce {
			toScan = append(toScan, path)
			delete(w.pending, path)
		}
	}

	for _, path := range toScan {
		log.Printf("[watcher:%s] change detected: %s", w.name, path)
		if err := w.scanFn(w.workspace, path); err != nil {
			log.Printf("[watcher:%s] scan error for %s: %v", w.name, path, err)
		} else {
			atomic.AddInt64(&w.scanCount, 1)
			w.saveState()
		}
	}
}
