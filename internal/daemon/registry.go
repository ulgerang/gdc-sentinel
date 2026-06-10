package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WorkspaceStatus string

const (
	StatusRunning  WorkspaceStatus = "running"
	StatusStopped  WorkspaceStatus = "stopped"
	StatusCrashed  WorkspaceStatus = "crashed"
	StatusStarting WorkspaceStatus = "starting"
)

type Workspace struct {
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	Pid       int            `json:"pid,omitempty"`
	Status    WorkspaceStatus `json:"status"`
	StartedAt time.Time      `json:"started_at,omitempty"`
	Restarts  int            `json:"restarts"`
	ScanCount int64          `json:"scan_count"`
	LastError string         `json:"last_error,omitempty"`

	mu sync.RWMutex
}

type Registry struct {
	Dir   string
	mu     sync.RWMutex
	items  map[string]*Workspace
}

func LoadRegistry(dir string) (*Registry, error) {
	r := &Registry{
		Dir:   dir,
		items: make(map[string]*Workspace),
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create registry dir: %w", err)
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Register(name, workspacePath string) (*Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.items[name]; ok {
		return existing, fmt.Errorf("workspace %q already exists (status: %s)", name, existing.Status)
	}

	w := &Workspace{
		Name:   name,
		Path:   workspacePath,
		Status: StatusStopped,
	}
	r.items[name] = w
	return w, r.save(name, w)
}

func (r *Registry) Get(name string) (*Workspace, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.items[name]
	return w, ok
}

func (r *Registry) List() []*Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Workspace, 0, len(r.items))
	for _, w := range r.items {
		out = append(out, w)
	}
	return out
}

func (r *Registry) Update(name string, fn func(w *Workspace)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.items[name]
	if !ok {
		return fmt.Errorf("workspace %q not found", name)
	}
	fn(w)
	return r.save(name, w)
}

func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[name]; !ok {
		return fmt.Errorf("workspace %q not found", name)
	}
	delete(r.items, name)
	path := r.filePath(name)
	return os.Remove(path)
}

func (r *Registry) load() error {
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read registry dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(r.Dir, e.Name()))
		if err != nil {
			continue
		}
		var w Workspace
		if err := json.Unmarshal(data, &w); err != nil {
			continue
		}
		w.Status = StatusStopped
		w.Pid = 0
		r.items[w.Name] = &w
	}
	return nil
}

func (r *Registry) save(name string, w *Workspace) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspace: %w", err)
	}
	return os.WriteFile(r.filePath(name), data, 0644)
}

func (r *Registry) filePath(name string) string {
	return filepath.Join(r.Dir, name+".json")
}
