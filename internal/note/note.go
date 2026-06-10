package note

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ulgerang/gdc-sentinel/internal/idgen"
	"gopkg.in/yaml.v3"
)

type Note struct {
	ID        string    `yaml:"id"`
	NodeID    string    `yaml:"node_id"`
	Text      string    `yaml:"text"`
	CreatedAt time.Time `yaml:"created_at"`
}

type Manager struct {
	sentinelDir string
	notesDir    string
}

func NewManager(sentinelDir string) *Manager {
	return &Manager{
		sentinelDir: sentinelDir,
		notesDir:    filepath.Join(sentinelDir, "notes"),
	}
}

func (m *Manager) Add(nodeID, text string) (*Note, error) {
	if err := os.MkdirAll(m.notesDir, 0755); err != nil {
		return nil, fmt.Errorf("create notes dir: %w", err)
	}

	id, err := idgen.New()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	n := &Note{
		ID:        id,
		NodeID:    nodeID,
		Text:      text,
		CreatedAt: time.Now(),
	}

	data, err := yaml.Marshal(n)
	if err != nil {
		return nil, fmt.Errorf("marshal note: %w", err)
	}

	path := filepath.Join(m.notesDir, n.ID+".yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("write note: %w", err)
	}

	return n, nil
}

func (m *Manager) ListByNode(nodeID string) ([]Note, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}

	var filtered []Note
	for _, n := range all {
		if n.NodeID == nodeID {
			filtered = append(filtered, n)
		}
	}
	return filtered, nil
}

func (m *Manager) List() ([]Note, error) {
	entries, err := os.ReadDir(m.notesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read notes dir: %w", err)
	}

	var notes []Note
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		n, err := m.readNote(filepath.Join(m.notesDir, entry.Name()))
		if err != nil {
			continue
		}
		notes = append(notes, *n)
	}

	return notes, nil
}

func (m *Manager) Delete(id string) error {
	path := filepath.Join(m.notesDir, id+".yaml")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete note %s: %w", id, err)
	}
	return nil
}

func (m *Manager) readNote(path string) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read note %s: %w", path, err)
	}

	var n Note
	if err := yaml.Unmarshal(data, &n); err != nil {
		return nil, fmt.Errorf("parse note %s: %w", path, err)
	}
	return &n, nil
}
