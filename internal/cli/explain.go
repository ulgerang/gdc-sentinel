package cli

import (
	"fmt"

	"github.com/gdc-tools/gdc-sentinel/internal/config"
	"github.com/gdc-tools/gdc-sentinel/internal/gdc"
	"github.com/gdc-tools/gdc-sentinel/internal/inbox"
	"github.com/gdc-tools/gdc-sentinel/internal/note"
	"github.com/spf13/cobra"
)

var explainFile string
var explainNode string

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain drift or node context",
	Long:  "Provide a human-readable explanation of drift items or node context.",
	RunE:  runExplain,
}

func init() {
	explainCmd.Flags().StringVar(&explainFile, "file", "", "file path to explain")
	explainCmd.Flags().StringVar(&explainNode, "node", "", "node ID to explain")
}

func runExplain(cmd *cobra.Command, args []string) error {
	if explainFile == "" && explainNode == "" {
		return fmt.Errorf("at least one of --file or --node is required")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	projectRoot := cfg.ResolvePath(cfg.Project.Path)
	client := gdc.NewClient(cfg.Project.GDCCommand, projectRoot, cfg.GDCCommands())

	target := explainNode
	var queryResult *gdc.QueryResponse
	if target == "" && explainFile != "" {
		res, err := client.Query(explainFile)
		if err != nil {
			return fmt.Errorf("query file %s: %w", explainFile, err)
		}
		queryResult = res
		target = res.CanonicalID
		if target == "" {
			target = res.ID
		}
	}

	if queryResult == nil {
		res, err := client.Query(target)
		if err != nil {
			return fmt.Errorf("query node %s: %w", target, err)
		}
		queryResult = res
	}

	depsResult, err := client.Deps(queryResult.CanonicalID, 2)
	if err != nil {
		printWarning("Could not fetch dependencies: %v", err)
	}

	refsResult, err := client.Refs(queryResult.CanonicalID, 2)
	if err != nil {
		printWarning("Could not fetch references: %v", err)
	}

	inboxMgr := inbox.NewManager(cfg.SentinelDir())
	driftItems, err := inboxMgr.ListByNode(queryResult.CanonicalID)
	if err != nil {
		printWarning("Could not load drift items: %v", err)
	}

	noteMgr := note.NewManager(cfg.SentinelDir())
	notes, notesErr := noteMgr.ListByNode(queryResult.CanonicalID)
	if notesErr != nil {
		printWarning("Could not load notes: %v", notesErr)
	}

	fmt.Println()
	printInfo("Node: %s (%s)", queryResult.CanonicalID, queryResult.Type)
	fmt.Printf("  Layer: %s\n", queryResult.Layer)
	fmt.Printf("  Status: %s\n", queryResult.Status)
	fmt.Printf("  Responsibility: %s\n", queryResult.Responsibility)
	fmt.Println()

	if depsResult != nil {
		fmt.Printf("Dependencies (%d):\n", len(depsResult.Dependencies))
		for _, dep := range depsResult.Dependencies {
			fmt.Printf("  → %s (%s)\n", dep.Target, dep.Type)
		}
		fmt.Println()
	}

	if refsResult != nil {
		fmt.Printf("Referenced By (%d):\n", len(refsResult.References))
		for _, ref := range refsResult.References {
			fmt.Printf("  ← %s (%s)\n", ref.Node, ref.Type)
		}
		fmt.Println()
	}

	fmt.Printf("Drift Items (%d):\n", len(driftItems))
	for _, item := range driftItems {
		fmt.Printf("  - %s: %s at %s\n", item.ID, item.ChangeType, item.FilePath)
	}
	fmt.Println()

	fmt.Printf("Notes (%d):\n", len(notes))
	for _, n := range notes {
		fmt.Printf("  - %s\n", n.Text)
	}
	fmt.Println()

	return nil
}
