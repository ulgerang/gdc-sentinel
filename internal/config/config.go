package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version     int                        `yaml:"version"`
	Project     ProjectConfig              `yaml:"project"`
	GDC         GDCConfig                  `yaml:"gdc"`
	Scan        ScanConfig                 `yaml:"scan"`
	DriftPolicy DriftPolicy                `yaml:"drift_policy"`
	Packets     PacketsConfig              `yaml:"packets"`
	Agents      map[string]AgentConfig     `yaml:"agents"`
	Templates   TemplatesConfig            `yaml:"templates"`
}

type ProjectConfig struct {
	Name         string `yaml:"name"`
	Path         string `yaml:"path"`
	GDCCommand   string `yaml:"gdc_command"`
	DefaultSince string `yaml:"default_since"`
}

type GDCConfig struct {
	PreferContextCommand bool                `yaml:"prefer_context_command"`
	Commands             map[string][]string `yaml:"commands"`
}

type ScanConfig struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type DriftPolicy struct {
	AutoWriteSpecs       bool   `yaml:"auto_write_specs"`
	RequireHumanApproval bool   `yaml:"require_human_approval"`
	ConfidenceThreshold  string `yaml:"confidence_threshold"`
}

type PacketsConfig struct {
	DefaultAgent string   `yaml:"default_agent"`
	Include      []string `yaml:"include"`
}

type AgentConfig struct {
	Type           string   `yaml:"type"`
	Command        string   `yaml:"command"`
	Args           []string `yaml:"args"`
	InputMode      string   `yaml:"input_mode"`
	WorkingDir     string   `yaml:"working_dir"`
	TimeoutSeconds int      `yaml:"timeout_seconds"`
}

type TemplatesConfig struct {
	DefaultPacket     string            `yaml:"default_packet"`
	AgentInstructions map[string]string `yaml:"agent_instructions"`
}

func Load(projectRoot string) (*Config, error) {
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		projectRoot = cwd
	}

	configPath := filepath.Join(projectRoot, ".gdc-sentinel", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Project: ProjectConfig{
			Name:         "",
			Path:         ".",
			GDCCommand:   "gdc",
			DefaultSince: "24h",
		},
		GDC: GDCConfig{
			PreferContextCommand: false,
			Commands: map[string][]string{
				"query":   {"query", "{symbol}", "--format", "json"},
				"deps":    {"deps", "{node}", "--depth", "{depth}"},
				"refs":    {"refs", "{node}", "--depth", "{depth}"},
				"context": {"context", "{node}", "--with-impl", "--with-tests", "--with-callers"},
				"diff":    {"diff", "{node}", "--format", "json"},
				"check":   {"check", "--format", "json"},
				"graph":   {"graph", "--format", "json"},
				"extract": {"extract", "{node}"},
			},
		},
		Scan: ScanConfig{
			Include: []string{"**/*.go", "**/*.ts", "**/*.tsx"},
			Exclude: []string{"vendor/**", "node_modules/**", ".git/**"},
		},
		DriftPolicy: DriftPolicy{
			AutoWriteSpecs:       false,
			RequireHumanApproval: true,
			ConfidenceThreshold:  "high",
		},
		Packets: PacketsConfig{
			DefaultAgent: "default",
			Include:      []string{"context", "drift", "notes", "changed_files"},
		},
		Agents: map[string]AgentConfig{
			"default": {
				Type:           "cli",
				Command:        "echo",
				Args:           []string{"{{packet_path}}"},
				InputMode:      "file",
				WorkingDir:     ".",
				TimeoutSeconds: 300,
			},
		},
		Templates: TemplatesConfig{
			DefaultPacket:     "default",
			AgentInstructions: map[string]string{},
		},
	}
}

func (c *Config) SentinelDir() string {
	root := c.Project.Path
	if root == "" || root == "." {
		root = mustGetwd()
	}
	return filepath.Join(root, ".gdc-sentinel")
}

func (c *Config) ResolvePath(path string) string {
	root := c.Project.Path
	if root == "" || root == "." {
		root = mustGetwd()
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("resolve working directory: %v", err)
	}
	return cwd
}

func (c *Config) GDCCommands() map[string][]string {
	if len(c.GDC.Commands) == 0 {
		return nil
	}
	out := make(map[string][]string, len(c.GDC.Commands))
	for k, v := range c.GDC.Commands {
		out[k] = v
	}
	return out
}
