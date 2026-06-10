package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ReadsConfig(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, ".gdc-sentinel")
	if err := os.MkdirAll(sentinel, 0755); err != nil {
		t.Fatal(err)
	}
	cfgYAML := `version: 1
project:
  name: demo
  path: .
  gdc_command: gdc
  default_since: 24h
agents:
  default:
    type: cli
    command: echo
    args: ["{{packet_path}}"]
    working_dir: .
    timeout_seconds: 60
`
	if err := os.WriteFile(filepath.Join(sentinel, "config.yaml"), []byte(cfgYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Project.Name != "demo" {
		t.Errorf("Project.Name = %q, want demo", cfg.Project.Name)
	}
	if cfg.Agents["default"].Command != "echo" {
		t.Errorf("default agent command = %q, want echo", cfg.Agents["default"].Command)
	}
	if cfg.Agents["default"].TimeoutSeconds != 60 {
		t.Errorf("timeout = %d, want 60", cfg.Agents["default"].TimeoutSeconds)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Errorf("expected error for missing config")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, ".gdc-sentinel")
	if err := os.MkdirAll(sentinel, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sentinel, "config.yaml"), []byte("not: [valid"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Errorf("expected parse error")
	}
}

func TestDefaultConfig_Shape(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if cfg.Project.GDCCommand == "" {
		t.Errorf("DefaultConfig missing GDCCommand")
	}
	if _, ok := cfg.Agents["default"]; !ok {
		t.Errorf("DefaultConfig missing default agent")
	}
	if cfg.Agents["default"].TimeoutSeconds <= 0 {
		t.Errorf("default agent timeout not set")
	}
}

func TestResolvePath_AbsoluteReturned(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Project.Path = t.TempDir()
	got := cfg.ResolvePath("/etc/passwd")
	if got != "/etc/passwd" {
		t.Errorf("absolute path should be returned as-is, got %q", got)
	}
}

func TestResolvePath_RelativeJoined(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfig()
	cfg.Project.Path = root
	got := cfg.ResolvePath("sub/file.yaml")
	want := filepath.Join(root, "sub/file.yaml")
	if got != want {
		t.Errorf("ResolvePath = %q, want %q", got, want)
	}
}

func TestSentinelDir(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfig()
	cfg.Project.Path = root
	got := cfg.SentinelDir()
	want := filepath.Join(root, ".gdc-sentinel")
	if got != want {
		t.Errorf("SentinelDir = %q, want %q", got, want)
	}
}

func TestGDCCommands_ReturnsCopy(t *testing.T) {
	cfg := DefaultConfig()
	got := cfg.GDCCommands()
	if got == nil {
		t.Fatalf("GDCCommands returned nil")
	}
	if len(got) != len(cfg.GDC.Commands) {
		t.Errorf("length mismatch")
	}
	got["query"] = []string{"hacked"}
	if cfg.GDC.Commands["query"][0] == "hacked" {
		t.Errorf("GDCCommands should return a defensive copy")
	}
}
