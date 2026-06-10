package inbox

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdc-tools/gdc-sentinel/internal/gdc"
	"github.com/gdc-tools/gdc-sentinel/internal/idgen"
	"gopkg.in/yaml.v3"
)

type DriftItem struct {
	ID          string            `yaml:"id"`
	NodeID      string            `yaml:"node_id"`
	FilePath    string            `yaml:"file_path"`
	ChangeType  string            `yaml:"change_type"`
	Drift       *gdc.DiffResponse `yaml:"drift,omitempty"`
	QueryResult *gdc.QueryResponse `yaml:"query_result,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at"`
	Status      string            `yaml:"status"`
}

type Manager struct {
	sentinelDir string
	inboxDir    string
}

func NewManager(sentinelDir string) *Manager {
	return &Manager{
		sentinelDir: sentinelDir,
		inboxDir:    filepath.Join(sentinelDir, "inbox"),
	}
}

func (m *Manager) Create(item DriftItem) error {
	if err := os.MkdirAll(m.inboxDir, 0755); err != nil {
		return fmt.Errorf("create inbox dir: %w", err)
	}

	if item.ID == "" {
		id, err := idgen.New()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		item.ID = id
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.Status == "" {
		item.Status = "open"
	}

	data, err := yaml.Marshal(&item)
	if err != nil {
		return fmt.Errorf("marshal drift item: %w", err)
	}

	path := filepath.Join(m.inboxDir, item.ID+".yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write drift item: %w", err)
	}

	return nil
}

func (m *Manager) List() ([]DriftItem, error) {
	entries, err := os.ReadDir(m.inboxDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read inbox dir: %w", err)
	}

	var items []DriftItem
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		item, err := m.readItem(filepath.Join(m.inboxDir, entry.Name()))
		if err != nil {
			continue
		}
		items = append(items, *item)
	}

	return items, nil
}

func (m *Manager) Get(id string) (*DriftItem, error) {
	path := filepath.Join(m.inboxDir, id+".yaml")
	return m.readItem(path)
}

func (m *Manager) ListByNode(nodeID string) ([]DriftItem, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}

	var filtered []DriftItem
	for _, item := range all {
		if item.NodeID == nodeID {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (m *Manager) UpdateStatus(id, status string) error {
	item, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("get drift item %s: %w", id, err)
	}

	item.Status = status

	data, err := yaml.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal drift item: %w", err)
	}

	path := filepath.Join(m.inboxDir, id+".yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write drift item: %w", err)
	}

	return nil
}

func (m *Manager) readItem(path string) (*DriftItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read drift item %s: %w", path, err)
	}

	var item DriftItem
	if err := yaml.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("parse drift item %s: %w", path, err)
	}
	return &item, nil
}
