package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gdc-tools/gdc-sentinel/internal/config"
	"github.com/gdc-tools/gdc-sentinel/internal/gdc"
	"github.com/gdc-tools/gdc-sentinel/internal/inbox"
	"github.com/gdc-tools/gdc-sentinel/internal/note"
	"github.com/gdc-tools/gdc-sentinel/internal/packet"
	"github.com/gdc-tools/gdc-sentinel/internal/run"
	"github.com/spf13/cobra"
)

var runAgent string
var runNode string
var runPacketPath string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute an agent with a context packet",
	Long:  "Run an external coding agent using a generated context packet, capturing output and results.",
	RunE:  runRun,
}

func init() {
	runCmd.Flags().StringVar(&runAgent, "agent", "", "agent name (required)")
	runCmd.Flags().StringVar(&runNode, "node", "", "node ID (required)")
	runCmd.Flags().StringVar(&runPacketPath, "packet", "", "path to existing packet (optional)")
}

func runRun(cmd *cobra.Command, args []string) error {
	if runAgent == "" || runNode == "" {
		return fmt.Errorf("--agent and --node are required")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	agentCfg, ok := cfg.Agents[runAgent]
	if !ok {
		return fmt.Errorf("agent %q not found in config", runAgent)
	}

	projectRoot := cfg.ResolvePath(cfg.Project.Path)

	packetPath := runPacketPath
	var packetText string

	if packetPath == "" {
		client := gdc.NewClient(cfg.Project.GDCCommand, projectRoot, cfg.GDCCommands())

		ctxResult, err := client.Context(runNode, true, true, true)
		if err != nil {
			return fmt.Errorf("get context: %w", err)
		}

		ctxJSON, err := json.Marshal(ctxResult)
		if err != nil {
			return fmt.Errorf("marshal context: %w", err)
		}
		rawCtx := json.RawMessage(ctxJSON)

		inboxMgr := inbox.NewManager(cfg.SentinelDir())
		driftItems, err := inboxMgr.ListByNode(runNode)
		if err != nil {
			printWarning("Could not load drift items: %v", err)
		}

		noteMgr := note.NewManager(cfg.SentinelDir())
		notes, err := noteMgr.ListByNode(runNode)
		if err != nil {
			printWarning("Could not load notes: %v", err)
		}

		var agentTemplate string
		if instrPath, ok := cfg.Templates.AgentInstructions[runAgent]; ok {
			templatePath := cfg.ResolvePath(instrPath)
			data, readErr := os.ReadFile(templatePath)
			if readErr == nil {
				agentTemplate = string(data)
			}
		}

		gen := packet.NewGenerator(cfg.SentinelDir())
		pkt := packet.Packet{
			NodeID:        runNode,
			AgentName:     runAgent,
			GDCContext:    &rawCtx,
			DriftItems:    driftItems,
			Notes:         notes,
			AgentTemplate: agentTemplate,
		}

		pktPath, err := gen.Generate(pkt)
		if err != nil {
			return fmt.Errorf("generate packet: %w", err)
		}
		packetPath = pktPath
	}

	data, err := os.ReadFile(packetPath)
	if err != nil {
		return fmt.Errorf("read packet %s: %w", packetPath, err)
	}
	packetText = string(data)

	executor := run.NewExecutor(cfg.SentinelDir())

	log, err := executor.Execute(agentCfg, packetPath, packetText, projectRoot, runNode)
	if err != nil {
		return fmt.Errorf("execute agent: %w", err)
	}

	printSuccess("Run complete")
	printInfo("  Command: %s", log.Command)
	printInfo("  Exit code: %d", log.ExitCode)
	printInfo("  Duration: %s", log.Duration)

	if verbose && log.Stdout != "" {
		fmt.Println()
		printInfo("Output:")
		fmt.Println(log.Stdout)
	}

	if log.ExitCode != 0 && log.Stderr != "" {
		printWarning("Stderr: %s", log.Stderr)
	}

	return nil
}
