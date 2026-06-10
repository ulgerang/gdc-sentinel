package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulgerang/gdc-sentinel/internal/config"
)

func TestReplaceTemplateVars(t *testing.T) {
	got := replaceTemplateVars("cmd {{packet_path}} {{project_root}} {{packet_text}} end",
		"/tmp/p.md", "BODY", "/proj")
	want := "cmd /tmp/p.md /proj BODY end"
	if got != want {
		t.Errorf("replaceTemplateVars = %q, want %q", got, want)
	}
}

func TestReplaceTemplateVars_LeavesUnknown(t *testing.T) {
	got := replaceTemplateVars("{{unknown}} stays", "/p", "B", "/r")
	if got != "{{unknown}} stays" {
		t.Errorf("unknown vars should be left alone, got %q", got)
	}
}

func TestValidateCommand_OK(t *testing.T) {
	dir := t.TempDir()
	cfg := config.AgentConfig{
		Command:    "echo",
		Args:       []string{"{{packet_path}}"},
		WorkingDir: ".",
	}
	if err := ValidateCommand(cfg, dir); err != nil {
		t.Errorf("ValidateCommand OK case: %v", err)
	}
}

func TestValidateCommand_RejectsShellMetachars(t *testing.T) {
	cases := []string{
		"echo foo; rm -rf /",
		"echo foo && bad",
		"echo $(whoami)",
		"echo `id`",
		"echo a|b",
		"echo $HOME",
		"echo a<file",
		"echo a>file",
		"echo a\nb",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			cfg := config.AgentConfig{Command: cmd}
			if err := ValidateCommand(cfg, t.TempDir()); err == nil {
				t.Errorf("expected error for %q, got nil", cmd)
			}
		})
	}
}

func TestValidateCommand_AllowsTemplateBraces(t *testing.T) {
	cfg := config.AgentConfig{
		Command: "echo",
		Args:    []string{"{{packet_path}}", "{{project_root}}", "{{packet_text}}"},
	}
	if err := ValidateCommand(cfg, t.TempDir()); err != nil {
		t.Errorf("template braces in args should be allowed, got: %v", err)
	}
}

func TestValidateCommand_RejectsMetacharsInArgs(t *testing.T) {
	cfg := config.AgentConfig{Command: "echo", Args: []string{"ok", "; bad"}}
	if err := ValidateCommand(cfg, t.TempDir()); err == nil {
		t.Errorf("expected error for metachar in args")
	}
}

func TestValidateCommand_RejectsMissingBinary(t *testing.T) {
	cfg := config.AgentConfig{Command: "definitely-not-a-real-binary-xyz123"}
	if err := ValidateCommand(cfg, t.TempDir()); err == nil {
		t.Errorf("expected error for missing binary")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestValidateCommand_RejectsEmptyCommand(t *testing.T) {
	cfg := config.AgentConfig{Command: ""}
	if err := ValidateCommand(cfg, t.TempDir()); err == nil {
		t.Errorf("expected error for empty command")
	}
}

func TestValidateCommand_RejectsWorkingDirEscape(t *testing.T) {
	dir := t.TempDir()
	cfg := config.AgentConfig{Command: "echo", WorkingDir: "../escape"}
	if err := ValidateCommand(cfg, dir); err == nil {
		t.Errorf("expected error for working_dir escape, got nil")
	}
}

func TestValidateCommand_AcceptsWorkingDirInsideRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := config.AgentConfig{Command: "echo", WorkingDir: "sub"}
	if err := ValidateCommand(cfg, dir); err != nil {
		t.Errorf("expected ok for subdir, got %v", err)
	}
}

func TestExecutor_ExecuteEcho(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)

	cfg := config.AgentConfig{
		Type:           "cli",
		Command:        "echo",
		Args:           []string{"hello-{{packet_path}}"},
		InputMode:      "file",
		WorkingDir:     ".",
		TimeoutSeconds: 10,
	}
	log, err := exe.Execute(cfg, "/tmp/packet.md", "BODY", t.TempDir(), "node-1")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if log.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 (stderr=%q)", log.ExitCode, log.Stderr)
	}
	if !strings.Contains(log.Stdout, "hello-/tmp/packet.md") {
		t.Errorf("Stdout = %q, want template substitution", log.Stdout)
	}
	if log.ID == "" {
		t.Errorf("log.ID is empty")
	}
}

func TestExecutor_LogsAreWritten(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{Type: "cli", Command: "echo", Args: []string{"hi"}, TimeoutSeconds: 5}
	log, err := exe.Execute(cfg, "/tmp/p.md", "BODY", t.TempDir(), "n")
	if err != nil {
		t.Fatal(err)
	}
	got, err := exe.Get(log.ID)
	if err != nil {
		t.Fatalf("Get(%s): %v", log.ID, err)
	}
	if got.Command != "echo" {
		t.Errorf("loaded command = %q, want echo", got.Command)
	}
}

func TestExecutor_RejectsInvalidCommand(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{Command: "echo; bad", Args: []string{"x"}}
	_, err := exe.Execute(cfg, "/p", "B", t.TempDir(), "n")
	if err == nil {
		t.Errorf("expected validation error")
	}
}

func TestExecutor_TimeoutProducesNonZero(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{
		Command:        "sleep",
		Args:           []string{"30"},
		TimeoutSeconds: 1,
	}
	log, err := exe.Execute(cfg, "/p", "B", t.TempDir(), "n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if log.ExitCode == 0 {
		t.Errorf("expected non-zero exit on timeout, got 0")
	}
}

func TestWireInput_Stdin(t *testing.T) {
	cmd := exec.Command("cat")
	if err := wireInput(cmd, "stdin", "hello-body", "/p"); err != nil {
		t.Fatalf("wireInput: %v", err)
	}
	if cmd.Stdin == nil {
		t.Errorf("stdin should be set for stdin mode")
	}
}

func TestWireInput_Argument(t *testing.T) {
	cmd := exec.Command("echo", "a", "b")
	if err := wireInput(cmd, "argument", "BODY", "/p"); err != nil {
		t.Fatalf("wireInput: %v", err)
	}
	if cmd.Stdin != nil {
		t.Errorf("argument mode should not touch stdin")
	}
	if got := cmd.Args; len(got) != 4 || got[3] != "BODY" {
		t.Errorf("argument mode should append packet text to cmd.Args, got %v", got)
	}
}

func TestWireInput_FileNoOp(t *testing.T) {
	cmd := exec.Command("echo", "{{packet_path}}")
	if err := wireInput(cmd, "file", "BODY", "/p"); err != nil {
		t.Fatalf("wireInput: %v", err)
	}
	if cmd.Stdin != nil {
		t.Errorf("file mode should not set stdin")
	}
	if len(cmd.Args) != 2 {
		t.Errorf("file mode should not touch cmd.Args, got %v", cmd.Args)
	}
}

func TestWireInput_UnknownErrors(t *testing.T) {
	cmd := exec.Command("echo")
	if err := wireInput(cmd, "magic", "BODY", "/p"); err == nil {
		t.Errorf("expected error for unknown input mode")
	}
}

func TestExecutor_StdinModePipesPacketText(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{
		Command:        "cat",
		Args:           []string{},
		InputMode:      "stdin",
		TimeoutSeconds: 5,
	}
	log, err := exe.Execute(cfg, "/p.md", "MARKER-XYZ", t.TempDir(), "n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if log.ExitCode != 0 {
		t.Errorf("ExitCode = %d, stderr=%q", log.ExitCode, log.Stderr)
	}
	if !strings.Contains(log.Stdout, "MARKER-XYZ") {
		t.Errorf("stdin mode should pipe packet text to command, got stdout=%q", log.Stdout)
	}
}

func TestExecutor_ArgumentModeAppendsBody(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{
		Command:        "echo",
		Args:           []string{"prefix"},
		InputMode:      "argument",
		TimeoutSeconds: 5,
	}
	log, err := exe.Execute(cfg, "/p.md", "BODY-CONTENT", t.TempDir(), "n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(log.Stdout, "prefix") || !strings.Contains(log.Stdout, "BODY-CONTENT") {
		t.Errorf("argument mode should pass body after configured args, got stdout=%q", log.Stdout)
	}
}

func TestExecutor_EmptyInputModeDefaultsToFile(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{
		Command:        "echo",
		Args:           []string{"{{packet_path}}"},
		InputMode:      "",
		TimeoutSeconds: 5,
	}
	log, err := exe.Execute(cfg, "/p.md", "B", t.TempDir(), "n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if log.ExitCode != 0 {
		t.Errorf("ExitCode = %d, stderr=%q", log.ExitCode, log.Stderr)
	}
	if !strings.Contains(log.Stdout, "/p.md") {
		t.Errorf("empty InputMode should default to file mode, got stdout=%q", log.Stdout)
	}
}

func TestExecutor_UnknownInputModeErrors(t *testing.T) {
	dir := t.TempDir()
	exe := NewExecutor(dir)
	cfg := config.AgentConfig{
		Command:        "echo",
		Args:           []string{"x"},
		InputMode:      "magic",
		TimeoutSeconds: 5,
	}
	_, err := exe.Execute(cfg, "/p.md", "B", t.TempDir(), "n")
	if err == nil {
		t.Errorf("expected error for unknown input mode")
	} else if !strings.Contains(err.Error(), "input_mode") {
		t.Errorf("error should mention input_mode, got: %v", err)
	}
}
