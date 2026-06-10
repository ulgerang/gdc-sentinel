package packet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdc-tools/gdc-sentinel/internal/inbox"
	"github.com/gdc-tools/gdc-sentinel/internal/note"
)

type Packet struct {
	NodeID        string
	AgentName     string
	GDCContext    *json.RawMessage
	DriftItems    []inbox.DriftItem
	Notes         []note.Note
	AgentTemplate string
	GeneratedAt   time.Time
}

type Generator struct {
	sentinelDir string
	packetsDir  string
}

func NewGenerator(sentinelDir string) *Generator {
	return &Generator{
		sentinelDir: sentinelDir,
		packetsDir:  filepath.Join(sentinelDir, "packets"),
	}
}

func (g *Generator) Generate(p Packet) (string, error) {
	if err := os.MkdirAll(g.packetsDir, 0755); err != nil {
		return "", fmt.Errorf("create packets dir: %w", err)
	}

	if p.GeneratedAt.IsZero() {
		p.GeneratedAt = time.Now()
	}

	content, err := g.RenderMarkdown(p)
	if err != nil {
		return "", fmt.Errorf("render packet: %w", err)
	}

	ts := p.GeneratedAt.Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s.md", sanitizeFilename(p.NodeID), sanitizeFilename(p.AgentName), ts)
	path := filepath.Join(g.packetsDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write packet: %w", err)
	}

	return path, nil
}

func (g *Generator) RenderMarkdown(p Packet) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Context Packet: %s\n\n", p.NodeID))
	b.WriteString(fmt.Sprintf("- **Agent**: %s\n", p.AgentName))
	b.WriteString(fmt.Sprintf("- **Generated**: %s\n\n", p.GeneratedAt.Format(time.RFC3339)))

	b.WriteString("## GDC Context\n\n")
	if p.GDCContext != nil {
		ctxJSON, err := json.MarshalIndent(p.GDCContext, "", "  ")
		if err != nil {
			b.WriteString("*(error rendering context)*\n")
		} else {
			b.WriteString("```json\n")
			b.Write(ctxJSON)
			b.WriteString("\n```\n")
		}
	} else {
		b.WriteString("*No GDC context available*\n")
	}
	b.WriteString("\n")

	b.WriteString("## Drift Items\n\n")
	if len(p.DriftItems) > 0 {
		for _, item := range p.DriftItems {
			b.WriteString(fmt.Sprintf("- **%s** (%s): %s — `%s`\n", item.ID, item.ChangeType, item.NodeID, item.FilePath))
		}
	} else {
		b.WriteString("*No drift items*\n")
	}
	b.WriteString("\n")

	b.WriteString("## Notes\n\n")
	if len(p.Notes) > 0 {
		for _, n := range p.Notes {
			b.WriteString(fmt.Sprintf("- **%s** (node: %s): %s\n", n.ID, n.NodeID, n.Text))
		}
	} else {
		b.WriteString("*No notes*\n")
	}
	b.WriteString("\n")

	b.WriteString("## Agent Instructions\n\n")
	if p.AgentTemplate != "" {
		b.WriteString(p.AgentTemplate)
		b.WriteString("\n")
	} else {
		b.WriteString("*No agent-specific instructions*\n")
	}

	return b.String(), nil
}

func sanitizeFilename(s string) string {
	if s == "" {
		return "unnamed"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == ' ' || r == '\t':
			b.WriteByte('_')
		case r == '.' && b.Len() == 0:
			b.WriteByte('_')
		case r < 0x20 || r == 0x7f:
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "." || out == ".." {
		return "_" + out
	}
	return out
}
