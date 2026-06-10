package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ulgerang/gdc-sentinel/internal/config"
	"github.com/ulgerang/gdc-sentinel/internal/gdc"
	"github.com/ulgerang/gdc-sentinel/internal/inbox"
	"github.com/ulgerang/gdc-sentinel/internal/note"
	"github.com/ulgerang/gdc-sentinel/internal/packet"
	"github.com/spf13/cobra"
)

var packetNode string
var packetAgent string

var packetCmd = &cobra.Command{
	Use:   "packet",
	Short: "Generate context packets for agents",
	Long:  "Create context packets from GDC data, drift items, and notes for external coding agents.",
	RunE:  runPacket,
}

func init() {
	packetCmd.Flags().StringVar(&packetNode, "node", "", "node ID (required)")
	packetCmd.Flags().StringVar(&packetAgent, "agent", "", "agent name (defaults to config default_agent)")
}

func runPacket(cmd *cobra.Command, args []string) error {
	if packetNode == "" {
		return fmt.Errorf("--node is required")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	projectRoot := cfg.ResolvePath(cfg.Project.Path)
	client := gdc.NewClient(cfg.Project.GDCCommand, projectRoot, cfg.GDCCommands())

	ctxResult, err := client.Context(packetNode, true, true, true)
	if err != nil {
		return fmt.Errorf("get context for %s: %w", packetNode, err)
	}

	ctxJSON, err := json.Marshal(ctxResult)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	rawCtx := json.RawMessage(ctxJSON)

	inboxMgr := inbox.NewManager(cfg.SentinelDir())
	driftItems, err := inboxMgr.ListByNode(packetNode)
	if err != nil {
		printWarning("Could not load drift items: %v", err)
	}

	noteMgr := note.NewManager(cfg.SentinelDir())
	notes, err := noteMgr.ListByNode(packetNode)
	if err != nil {
		printWarning("Could not load notes: %v", err)
	}

	agentName := packetAgent
	if agentName == "" {
		agentName = cfg.Packets.DefaultAgent
	}

	var agentTemplate string
	if instrPath, ok := cfg.Templates.AgentInstructions[agentName]; ok {
		templatePath := cfg.ResolvePath(instrPath)
		data, readErr := os.ReadFile(templatePath)
		if readErr == nil {
			agentTemplate = string(data)
		} else if verbose {
			printWarning("Could not read agent template %s: %v", templatePath, readErr)
		}
	}

	gen := packet.NewGenerator(cfg.SentinelDir())

	pkt := packet.Packet{
		NodeID:        packetNode,
		AgentName:     agentName,
		GDCContext:    &rawCtx,
		DriftItems:    driftItems,
		Notes:         notes,
		AgentTemplate: agentTemplate,
	}

	pktPath, err := gen.Generate(pkt)
	if err != nil {
		return fmt.Errorf("generate packet: %w", err)
	}

	relPath, _ := filepath.Rel(projectRoot, pktPath)
	if relPath == "" {
		relPath = pktPath
	}
	printSuccess("Packet generated: %s", relPath)

	return nil
}
