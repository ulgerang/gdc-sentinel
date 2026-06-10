package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ulgerang/gdc-sentinel/internal/config"
	"github.com/ulgerang/gdc-sentinel/internal/gdc"
	"github.com/ulgerang/gdc-sentinel/internal/inbox"
	"github.com/spf13/cobra"
)

var scanSince string

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for drift between specs and implementation",
	Long:  "Run GDC commands to detect drift between node specifications and implementation code.",
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVar(&scanSince, "since", "", "git ref to diff from (defaults to config default_since)")
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	projectRoot := cfg.ResolvePath(cfg.Project.Path)

	since := scanSince
	if since == "" {
		since = cfg.Project.DefaultSince
	}
	if since == "" {
		since = "HEAD~1"
	}

	gitCmd := exec.Command("git", "diff", "-M", "--name-status", since)
	gitCmd.Dir = projectRoot
	out, err := gitCmd.Output()
	if err != nil {
		return fmt.Errorf("git diff --name-status %s: %w", since, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		printInfo("No changes found since %s", since)
		return nil
	}

	inboxMgr := inbox.NewManager(cfg.SentinelDir())

	filesScanned := 0
	nodesFound := 0
	itemsCreated := 0
	filesQueryFailed := 0
	filesDiffFailed := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		parsed, ok := parseGitStatusLine(line)
		if !ok {
			continue
		}

		if !shouldScan(parsed.filePath, cfg.Scan) {
			continue
		}

		filesScanned++

		result, err := scanFile(cfg, projectRoot, inboxMgr, parsed.filePath, parsed.changeType)
		if err != nil {
			printWarning("Scan failed for %s: %v", parsed.filePath, err)
			continue
		}
		if result.QueryFailed {
			filesQueryFailed++
			continue
		}
		if result.DiffFailed {
			filesDiffFailed++
		}
		if result.Created {
			itemsCreated++
		}
		if result.NodeFound {
			nodesFound++
		}
	}

	printSuccess("Scan complete: %d files scanned, %d nodes found, %d drift items created",
		filesScanned, nodesFound, itemsCreated)
	if filesQueryFailed > 0 || filesDiffFailed > 0 {
		printWarning("Failures: %d query, %d diff (see warnings above)", filesQueryFailed, filesDiffFailed)
	}

	return nil
}

type scanFileResult struct {
	NodeID       string
	NodeFound    bool
	Created      bool
	QueryFailed  bool
	DiffFailed   bool
}

func scanFile(cfg *config.Config, projectRoot string, inboxMgr *inbox.Manager, filePath, changeType string) (*scanFileResult, error) {
	result := &scanFileResult{}

	client := gdc.NewClient(cfg.Project.GDCCommand, projectRoot, cfg.GDCCommands())
	queryResult, err := client.Query(filePath)
	if err != nil {
		result.QueryFailed = true
		return result, fmt.Errorf("query %s: %w", filePath, err)
	}

	nodeID := queryResult.CanonicalID
	if nodeID == "" {
		nodeID = queryResult.ID
	}
	if nodeID == "" {
		return result, nil
	}

	result.NodeFound = true
	result.NodeID = nodeID

	var driftResp *gdc.DiffResponse
	diffResult, diffErr := client.Diff(nodeID)
	if diffErr == nil {
		driftResp = diffResult
	} else {
		result.DiffFailed = true
		printWarning("Could not diff %s: %v", nodeID, diffErr)
	}

	item := inbox.DriftItem{
		NodeID:      nodeID,
		FilePath:    filePath,
		ChangeType:  mapGitStatus(changeType),
		Drift:       driftResp,
		QueryResult: queryResult,
	}

	if err := inboxMgr.Create(item); err != nil {
		return result, fmt.Errorf("create drift item for %s: %w", filePath, err)
	}

	result.Created = true
	return result, nil
}

func shouldScan(filePath string, scanCfg config.ScanConfig) bool {
	for _, pattern := range scanCfg.Exclude {
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return false
		}
	}

	if len(scanCfg.Include) > 0 {
		matched := false
		for _, pattern := range scanCfg.Include {
			if m, _ := filepath.Match(pattern, filePath); m {
				matched = true
				break
			}
		}
		return matched
	}

	return true
}

func mapGitStatus(status string) string {
	switch {
	case strings.HasPrefix(status, "A"):
		return "added"
	case strings.HasPrefix(status, "D"):
		return "deleted"
	case strings.HasPrefix(status, "R"):
		return "renamed"
	default:
		return "modified"
	}
}

type gitStatusEntry struct {
	changeType string
	filePath   string
}

func parseGitStatusLine(line string) (gitStatusEntry, bool) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) < 2 {
		return gitStatusEntry{}, false
	}
	status := parts[0]
	filePath := parts[1]

	// Renames (Rxxx) emit three tab-separated fields; the third field
	// is the new path. The similarity suffix is dropped.
	if strings.HasPrefix(status, "R") && len(parts) >= 3 {
		filePath = parts[2]
	}

	return gitStatusEntry{
		changeType: mapGitStatus(status),
		filePath:   filePath,
	}, true
}
