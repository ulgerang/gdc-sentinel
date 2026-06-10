package run

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ulgerang/gdc-sentinel/internal/config"
	"github.com/ulgerang/gdc-sentinel/internal/idgen"
	"gopkg.in/yaml.v3"
)

type RunLog struct {
	ID         string    `yaml:"id"`
	AgentName  string    `yaml:"agent_name"`
	NodeID     string    `yaml:"node_id"`
	PacketPath string    `yaml:"packet_path"`
	Command    string    `yaml:"command"`
	Stdout     string    `yaml:"stdout"`
	Stderr     string    `yaml:"stderr"`
	ExitCode   int       `yaml:"exit_code"`
	StartTime  time.Time `yaml:"start_time"`
	EndTime    time.Time `yaml:"end_time"`
	Duration   string    `yaml:"duration"`
}

type Executor struct {
	sentinelDir string
	runsDir     string
}

func NewExecutor(sentinelDir string) *Executor {
	return &Executor{
		sentinelDir: sentinelDir,
		runsDir:     filepath.Join(sentinelDir, "runs"),
	}
}

func (e *Executor) Execute(agentCfg config.AgentConfig, packetPath, packetText, projectRoot, nodeID string) (*RunLog, error) {
	if err := os.MkdirAll(e.runsDir, 0755); err != nil {
		return nil, fmt.Errorf("create runs dir: %w", err)
	}

	if err := ValidateCommand(agentCfg, projectRoot); err != nil {
		return nil, fmt.Errorf("validate agent config: %w", err)
	}

	resolvedCmd := replaceTemplateVars(agentCfg.Command, packetPath, packetText, projectRoot)
	resolvedArgs := make([]string, len(agentCfg.Args))
	for i, arg := range agentCfg.Args {
		resolvedArgs[i] = replaceTemplateVars(arg, packetPath, packetText, projectRoot)
	}

	timeout := time.Duration(agentCfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := time.Now()
	cmd := exec.CommandContext(ctx, resolvedCmd, resolvedArgs...)

	if agentCfg.WorkingDir != "" {
		wd := replaceTemplateVars(agentCfg.WorkingDir, packetPath, packetText, projectRoot)
		cmd.Dir = wd
	} else {
		cmd.Dir = projectRoot
	}

	mode := agentCfg.InputMode
	if mode == "" {
		mode = "file"
	}
	if err := wireInput(cmd, mode, packetText, packetPath); err != nil {
		return nil, fmt.Errorf("input_mode %q: %w", agentCfg.InputMode, err)
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	id, err := idgen.New()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	log := &RunLog{
		ID:         id,
		AgentName:  agentCfg.Type,
		NodeID:     nodeID,
		PacketPath: packetPath,
		Command:    resolvedCmd,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   duration.String(),
	}

	data, err := yaml.Marshal(log)
	if err != nil {
		return nil, fmt.Errorf("marshal run log: %w", err)
	}

	logPath := filepath.Join(e.runsDir, log.ID+".yaml")
	if err := os.WriteFile(logPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write run log: %w", err)
	}

	return log, nil
}

func (e *Executor) List() ([]RunLog, error) {
	entries, err := os.ReadDir(e.runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}

	var logs []RunLog
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		log, err := e.readLog(filepath.Join(e.runsDir, entry.Name()))
		if err != nil {
			continue
		}
		logs = append(logs, *log)
	}

	return logs, nil
}

func (e *Executor) Get(id string) (*RunLog, error) {
	path := filepath.Join(e.runsDir, id+".yaml")
	return e.readLog(path)
}

func (e *Executor) readLog(path string) (*RunLog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read run log %s: %w", path, err)
	}

	var log RunLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("parse run log %s: %w", path, err)
	}
	return &log, nil
}

func replaceTemplateVars(s, packetPath, packetText, projectRoot string) string {
	s = strings.ReplaceAll(s, "{{packet_path}}", packetPath)
	s = strings.ReplaceAll(s, "{{packet_text}}", packetText)
	s = strings.ReplaceAll(s, "{{project_root}}", projectRoot)
	return s
}

func wireInput(cmd *exec.Cmd, mode, packetText, packetPath string) error {
	switch mode {
	case "stdin":
		cmd.Stdin = strings.NewReader(packetText)
	case "file":
		// {{packet_path}} in Args is the contract for this mode.
		// Nothing to wire; the resolved args already contain the path.
	case "argument":
		cmd.Args = append(cmd.Args, packetText)
	default:
		return fmt.Errorf("unsupported mode (want file, stdin, or argument)")
	}
	return nil
}

func ValidateCommand(agentCfg config.AgentConfig, projectRoot string) error {
	if agentCfg.Command == "" {
		return fmt.Errorf("agent command is empty")
	}
	if err := assertNoShellMetachars("command", agentCfg.Command); err != nil {
		return err
	}
	if !filepath.IsAbs(agentCfg.Command) {
		if _, err := exec.LookPath(agentCfg.Command); err != nil {
			return fmt.Errorf("command %q not found on PATH", agentCfg.Command)
		}
	}
	for i, arg := range agentCfg.Args {
		if err := assertNoShellMetachars(fmt.Sprintf("args[%d]", i), arg); err != nil {
			return err
		}
	}
	if agentCfg.WorkingDir != "" {
		wd := agentCfg.WorkingDir
		if !filepath.IsAbs(wd) {
			wd = filepath.Join(projectRoot, wd)
		}
		cleaned := filepath.Clean(wd)
		absRoot, _ := filepath.Abs(projectRoot)
		rel, err := filepath.Rel(absRoot, cleaned)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			return fmt.Errorf("working_dir %q escapes project root", agentCfg.WorkingDir)
		}
	}
	return nil
}

func assertNoShellMetachars(field, value string) error {
	if strings.ContainsAny(value, ";&|`$()<>[]*?!~#\\\"'\n\r") {
		return fmt.Errorf("%s contains shell metacharacters: %q", field, value)
	}
	return nil
}
