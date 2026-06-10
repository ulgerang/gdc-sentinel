package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/ulgerang/gdc-sentinel/internal/config"
	"github.com/ulgerang/gdc-sentinel/internal/daemon"
	"github.com/ulgerang/gdc-sentinel/internal/inbox"
)

var (
	daemonName      string
	daemonWorkspace string
	daemonFollow    bool
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage sentinel daemon processes (PM2-style)",
	Long: `Manage sentinel daemon processes.

Subcommands:
  start    Start watching a workspace
  stop     Stop a running watcher
  restart  Restart a watcher
  list     List all registered workspaces
  status   Show detailed status of a workspace`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon and register a workspace for watching",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspacePath := daemonWorkspace
		if workspacePath == "" || workspacePath == "." {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			workspacePath = cwd
		}

		name := daemonName
		if name == "" {
			name = filepath.Base(workspacePath)
		}

		cfg, err := resolveSentinelDir(workspacePath)
		if err != nil {
			return err
		}
		socketPath := daemon.ReadSocketPath(cfg)

		if !daemon.IsDaemonRunning(socketPath) {
			printInfo("Starting daemon...")
			if err := spawnDaemon(cfg); err != nil {
				return fmt.Errorf("start daemon: %w", err)
			}
			if err := waitForDaemon(socketPath, 10*time.Second); err != nil {
				return fmt.Errorf("daemon did not start: %w", err)
			}
			printSuccess("Daemon started")
		}

		payload, _ := json.Marshal(daemon.StartPayload{
			Name: name,
			Path: workspacePath,
		})
		resp, err := daemon.SendIPC(socketPath, daemon.Message{
			Type:    daemon.MsgStart,
			Payload: payload,
		})
		if err != nil {
			return err
		}

		var result daemon.ResultPayload
		json.Unmarshal(resp.Payload, &result)
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}

		printSuccess("Workspace %q is now being watched", name)
		return nil
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop watching a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := daemonName
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("workspace name required (pass as argument or --name)")
		}

		socketPath, err := findSocketPath()
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(daemon.StopPayload{Name: name})
		resp, err := daemon.SendIPC(socketPath, daemon.Message{
			Type:    daemon.MsgStop,
			Payload: payload,
		})
		if err != nil {
			return err
		}

		var result daemon.ResultPayload
		json.Unmarshal(resp.Payload, &result)
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}

		printSuccess("Workspace %q stopped", name)
		return nil
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart [name]",
	Short: "Restart a workspace watcher",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := daemonName
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("workspace name required")
		}

		socketPath, err := findSocketPath()
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(daemon.StopPayload{Name: name})
		resp, err := daemon.SendIPC(socketPath, daemon.Message{
			Type:    daemon.MsgRestart,
			Payload: payload,
		})
		if err != nil {
			return err
		}

		var result daemon.ResultPayload
		json.Unmarshal(resp.Payload, &result)
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}

		printSuccess("Workspace %q restarted", name)
		return nil
	},
}

var daemonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		socketPath, err := findSocketPath()
		if err != nil {
			return err
		}

		resp, err := daemon.SendIPC(socketPath, daemon.Message{Type: daemon.MsgList})
		if err != nil {
			return err
		}

		var result daemon.ResultPayload
		json.Unmarshal(resp.Payload, &result)
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}

		data, _ := json.Marshal(result.Data)
		var workspaces []*daemon.Workspace
		json.Unmarshal(data, &workspaces)

		if len(workspaces) == 0 {
			printInfo("No workspaces registered")
			return nil
		}

		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tSTATUS\tPID\tUPTIME\tSCANS\tRESTARTS")
		for _, w := range workspaces {
			uptime := "-"
			if w.Status == daemon.StatusRunning && !w.StartedAt.IsZero() {
				uptime = time.Since(w.StartedAt).Truncate(time.Second).String()
			}
			pid := "-"
			if w.Pid > 0 {
				pid = strconv.Itoa(w.Pid)
			}
			statusStr := string(w.Status)
			if w.Status == daemon.StatusRunning {
				statusStr = color.GreenString("online")
			} else if w.Status == daemon.StatusCrashed {
				statusStr = color.RedString("crashed")
			} else {
				statusStr = color.YellowString("stopped")
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\n",
				w.Name, statusStr, pid, uptime, w.ScanCount, w.Restarts)
		}
		tw.Flush()
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show detailed status of a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := daemonName
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("workspace name required")
		}

		socketPath, err := findSocketPath()
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(daemon.StatusPayload{Name: name})
		resp, err := daemon.SendIPC(socketPath, daemon.Message{
			Type:    daemon.MsgStatus,
			Payload: payload,
		})
		if err != nil {
			return err
		}

		var result daemon.ResultPayload
		json.Unmarshal(resp.Payload, &result)
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}

		data, _ := json.Marshal(result.Data)
		var w daemon.Workspace
		json.Unmarshal(data, &w)

		fmt.Printf("Name:      %s\n", w.Name)
		fmt.Printf("Path:      %s\n", w.Path)
		fmt.Printf("Status:    %s\n", w.Status)
		if w.Pid > 0 {
			fmt.Printf("PID:       %d\n", w.Pid)
		}
		if !w.StartedAt.IsZero() {
			fmt.Printf("Started:   %s (%s ago)\n", w.StartedAt.Format(time.RFC3339), time.Since(w.StartedAt).Truncate(time.Second))
		}
		fmt.Printf("Scans:     %d\n", w.ScanCount)
		fmt.Printf("Restarts:  %d\n", w.Restarts)
		if w.LastError != "" {
			color.Red("Last Error: %s\n", w.LastError)
		}
		return nil
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Show logs for a workspace watcher",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := daemonName
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("workspace name required")
		}

		socketPath, err := findSocketPath()
		if err != nil {
			return err
		}

		resp, err := daemon.SendIPC(socketPath, daemon.Message{
			Type:    daemon.MsgStatus,
			Payload: mustMarshal(daemon.StatusPayload{Name: name}),
		})
		if err != nil {
			return err
		}

		var result daemon.ResultPayload
		json.Unmarshal(resp.Payload, &result)
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}

		data, _ := json.Marshal(result.Data)
		var w daemon.Workspace
		json.Unmarshal(data, &w)

		logDir := filepath.Join(filepath.Dir(socketPath), "daemons", name)
		logFile := filepath.Join(logDir, "stderr.log")

		if daemonFollow {
			return tailFollow(logFile)
		}

		data, err = os.ReadFile(logFile)
		if err != nil {
			if os.IsNotExist(err) {
				printInfo("No logs yet")
				return nil
			}
			return err
		}
		os.Stdout.Write(data)
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonListCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)

	daemonStartCmd.Flags().StringVar(&daemonName, "name", "", "workspace name (default: directory name)")
	daemonStartCmd.Flags().StringVar(&daemonWorkspace, "workspace", ".", "workspace path to watch")
	daemonStopCmd.Flags().StringVar(&daemonName, "name", "", "workspace name")
	daemonRestartCmd.Flags().StringVar(&daemonName, "name", "", "workspace name")
	daemonStatusCmd.Flags().StringVar(&daemonName, "name", "", "workspace name")
	daemonLogsCmd.Flags().StringVar(&daemonName, "name", "", "workspace name")
	daemonLogsCmd.Flags().BoolVar(&daemonFollow, "follow", false, "follow log output (tail -f)")
}

var watchCmd = &cobra.Command{
	Use:    "_watch",
	Short:  "Internal: run as a watcher worker process",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		watchName, _ := cmd.Flags().GetString("name")
		watchWorkspace, _ := cmd.Flags().GetString("workspace")

		cfg, err := config.Load(watchWorkspace)
		if err != nil {
			return fmt.Errorf("load config for %s: %w", watchWorkspace, err)
		}
		projectRoot := cfg.ResolvePath(cfg.Project.Path)
		inboxMgr := inbox.NewManager(cfg.SentinelDir())

		scanFn := func(workspace, filePath string) error {
			if !shouldScan(filePath, cfg.Scan) {
				return nil
			}
			printInfo("[scan] %s changed", filePath)
			result, err := scanFile(cfg, projectRoot, inboxMgr, filePath, "M")
			if err != nil {
				return err
			}
			if result.Created {
				printSuccess("Drift item created for %s (node: %s)", filePath, result.NodeID)
			}
			return nil
		}

		w, err := daemon.NewWatcher(watchName, watchWorkspace, scanFn)
		if err != nil {
			return err
		}

		fmt.Printf("[watcher:%s] watching %s\n", watchName, watchWorkspace)
		w.Run()
		return nil
	},
}

func init() {
	watchCmd.Flags().String("name", "", "workspace name")
	watchCmd.Flags().String("workspace", "", "workspace path")
}

func resolveSentinelDir(workspacePath string) (string, error) {
	cfg, err := config.Load(workspacePath)
	if err != nil {
		abs, err := filepath.Abs(workspacePath)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, ".gdc-sentinel"), nil
	}
	return cfg.SentinelDir(), nil
}

func findSocketPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	socketPath := daemon.ReadSocketPath(filepath.Join(cwd, ".gdc-sentinel"))
	if !daemon.IsDaemonRunning(socketPath) {
		return "", fmt.Errorf("daemon is not running (tried %s)", socketPath)
	}
	return socketPath, nil
}

func spawnDaemon(sentinelDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	logDir := filepath.Join(sentinelDir, "daemons", "_launcher")
	os.MkdirAll(logDir, 0755)

	stdout, _ := os.OpenFile(filepath.Join(logDir, "stdout.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	stderr, _ := os.OpenFile(filepath.Join(logDir, "stderr.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	cmd := exec.Command(exe, "_daemon", "--sentinel-dir", sentinelDir)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd.Start()
}

func waitForDaemon(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if daemon.IsDaemonRunning(socketPath) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for daemon")
}

func tailFollow(path string) error {
	cmd := exec.Command("tail", "-f", "-n", "50", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
