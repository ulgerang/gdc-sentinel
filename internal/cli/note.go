package cli

import (
	"fmt"
	"strings"

	"github.com/ulgerang/gdc-sentinel/internal/config"
	"github.com/ulgerang/gdc-sentinel/internal/note"
	"github.com/spf13/cobra"
)

var noteNode string
var noteText string
var noteID string

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage local notes for nodes",
	Long:  "Add, list, and delete local notes associated with GDC nodes.",
}

var noteAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a note to a node",
	RunE:  runNoteAdd,
}

var noteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	RunE:  runNoteList,
}

var noteDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a note",
	RunE:  runNoteDelete,
}

func init() {
	noteAddCmd.Flags().StringVar(&noteNode, "node", "", "node ID (required)")
	noteAddCmd.Flags().StringVar(&noteText, "text", "", "note text (required)")

	noteListCmd.Flags().StringVar(&noteNode, "node", "", "filter by node ID")

	noteDeleteCmd.Flags().StringVar(&noteID, "id", "", "note ID (required)")

	noteCmd.AddCommand(noteAddCmd, noteListCmd, noteDeleteCmd)
}

func runNoteAdd(cmd *cobra.Command, args []string) error {
	if noteNode == "" || noteText == "" {
		return fmt.Errorf("--node and --text are required")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	mgr := note.NewManager(cfg.SentinelDir())
	n, err := mgr.Add(noteNode, noteText)
	if err != nil {
		return fmt.Errorf("add note: %w", err)
	}

	printSuccess("Note added: %s", n.ID)
	return nil
}

func runNoteList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	mgr := note.NewManager(cfg.SentinelDir())

	var notes []note.Note
	if noteNode != "" {
		notes, err = mgr.ListByNode(noteNode)
	} else {
		notes, err = mgr.List()
	}
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}

	if len(notes) == 0 {
		printInfo("No notes found")
		return nil
	}

	fmt.Printf("%-20s %-20s %-60s %s\n", "ID", "Node", "Text", "Created")
	fmt.Println(strings.Repeat("-", 120))

	for _, n := range notes {
		text := n.Text
		if len(text) > 60 {
			text = text[:57] + "..."
		}
		fmt.Printf("%-20s %-20s %-60s %s\n", n.ID, n.NodeID, text, n.CreatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runNoteDelete(cmd *cobra.Command, args []string) error {
	if noteID == "" {
		return fmt.Errorf("--id is required")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	mgr := note.NewManager(cfg.SentinelDir())
	if err := mgr.Delete(noteID); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	printSuccess("Note deleted: %s", noteID)
	return nil
}
