package packet

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gdc-tools/gdc-sentinel/internal/gdc"
	"github.com/gdc-tools/gdc-sentinel/internal/inbox"
	"github.com/gdc-tools/gdc-sentinel/internal/note"
)

func TestRenderMarkdown_Golden(t *testing.T) {
	rawCtx := json.RawMessage(`{"node":{"id":"pkg.Foo"}}`)

	p := Packet{
		NodeID:        "pkg.Foo",
		AgentName:     "default",
		GDCContext:    &rawCtx,
		DriftItems:    []inbox.DriftItem{{ID: "d1", NodeID: "pkg.Foo", FilePath: "a.go", ChangeType: "modified"}},
		Notes:         []note.Note{{ID: "n1", NodeID: "pkg.Foo", Text: "watch the edge case"}},
		AgentTemplate: "do the thing",
		GeneratedAt:   time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}

	gen := NewGenerator(t.TempDir())
	got, err := gen.RenderMarkdown(p)
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}

	wants := []string{
		"# Context Packet: pkg.Foo",
		"**Agent**: default",
		"## GDC Context",
		"\"id\": \"pkg.Foo\"",
		"## Drift Items",
		"**d1**",
		"`a.go`",
		"## Notes",
		"watch the edge case",
		"## Agent Instructions",
		"do the thing",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("rendered markdown missing %q\n---got---\n%s", want, got)
		}
	}
}

func TestRenderMarkdown_Empty(t *testing.T) {
	p := Packet{NodeID: "X", AgentName: "a", GeneratedAt: time.Unix(0, 0).UTC()}
	gen := NewGenerator(t.TempDir())
	got, err := gen.RenderMarkdown(p)
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	for _, placeholder := range []string{"*No GDC context available*", "*No drift items*", "*No notes*", "*No agent-specific instructions*"} {
		if !strings.Contains(got, placeholder) {
			t.Errorf("expected placeholder %q in empty packet render", placeholder)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"foo", "foo"},
		{"foo/bar", "foo_bar"},
		{"a:b c", "a_b_c"},
		{"", "unnamed"},
		{".", "_"},
		{"..", "_."},
		{"..hidden", "_.hidden"},
		{"ctrl\x00char", "ctrl_char"},
		{"./etc/passwd", "__etc_passwd"},
		{"normal_name.go", "normal_name.go"},
	}
	for _, tc := range cases {
		got := sanitizeFilename(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGenerate_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	gen := NewGenerator(dir)

	raw := json.RawMessage(`{}`)
	pkt := Packet{
		NodeID:      "node-1",
		AgentName:   "default",
		GDCContext:  &raw,
		GeneratedAt: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}

	path, err := gen.Generate(pkt)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Errorf("packet path %q missing .md suffix", path)
	}
	if !strings.Contains(path, "node-1") {
		t.Errorf("packet path %q missing node id", path)
	}
}

func TestPacket_StructTypes(t *testing.T) {
	raw := json.RawMessage(`{"k":"v"}`)
	p := Packet{
		GDCContext: &raw,
		DriftItems: []inbox.DriftItem{{Drift: &gdc.DiffResponse{HasDrift: true}}},
		Notes:      []note.Note{{ID: "n"}},
	}
	if p.GDCContext == nil || string(*p.GDCContext) != `{"k":"v"}` {
		t.Errorf("GDCContext lost: %v", p.GDCContext)
	}
	if !p.DriftItems[0].Drift.HasDrift {
		t.Errorf("DriftItems payload lost")
	}
	if p.Notes[0].ID != "n" {
		t.Errorf("Notes payload lost")
	}
}
