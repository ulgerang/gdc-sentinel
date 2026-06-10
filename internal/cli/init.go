package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ulgerang/gdc-sentinel/internal/config"
	"github.com/ulgerang/gdc-sentinel/internal/ignore"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initName string
var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gdc-sentinel configuration",
	Long:  "Initialize gdc-sentinel in the current project directory, creating default config and directory structure.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", "", "project name (defaults to directory basename)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing configuration")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	sentinelDir := filepath.Join(cwd, ".gdc-sentinel")

	if info, _ := os.Stat(sentinelDir); info != nil {
		if !initForce {
			return fmt.Errorf(".gdc-sentinel already exists, use --force to reinitialize")
		}
		if err := os.RemoveAll(sentinelDir); err != nil {
			return fmt.Errorf("wipe existing .gdc-sentinel: %w", err)
		}
		printWarning("Wiped existing .gdc-sentinel (notes, inbox, runs, packets, reports)")
	}

	dirs := []string{
		sentinelDir,
		filepath.Join(sentinelDir, "state"),
		filepath.Join(sentinelDir, "inbox"),
		filepath.Join(sentinelDir, "reports"),
		filepath.Join(sentinelDir, "packets"),
		filepath.Join(sentinelDir, "notes"),
		filepath.Join(sentinelDir, "runs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	cfg := config.DefaultConfig()

	name := initName
	if name == "" {
		name = filepath.Base(cwd)
	}
	cfg.Project.Name = name

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	configPath := filepath.Join(sentinelDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	ignorePath := filepath.Join(cwd, ignore.DefaultFilename)
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(ignorePath, []byte(ignore.DefaultContent()), 0644); err != nil {
			return fmt.Errorf("write ignore file: %w", err)
		}
		printInfo("Created %s", ignore.DefaultFilename)
	}

	printSuccess("Initialized gdc-sentinel for project '%s'", name)
	printInfo("Created directories:")
	for _, dir := range dirs {
		rel, _ := filepath.Rel(cwd, dir)
		printInfo("  %s/", rel)
	}
	printInfo("Config written to .gdc-sentinel/config.yaml")

	return nil
}
