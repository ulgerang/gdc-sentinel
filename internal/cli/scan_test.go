package cli

import (
	"testing"

	"github.com/gdc-tools/gdc-sentinel/internal/config"
)

func TestShouldScan_ExcludeMatches(t *testing.T) {
	cfg := config.ScanConfig{Exclude: []string{"vendor/*", "node_modules/*"}}
	if shouldScan("vendor/foo.go", cfg) {
		t.Errorf("expected vendor/foo.go to be excluded")
	}
	if shouldScan("node_modules/x.js", cfg) {
		t.Errorf("expected node_modules/x.js to be excluded")
	}
	if !shouldScan("src/main.go", cfg) {
		t.Errorf("expected src/main.go to be scanned")
	}
}

func TestShouldScan_IncludeOnly(t *testing.T) {
	cfg := config.ScanConfig{Include: []string{"*.go"}}
	if !shouldScan("main.go", cfg) {
		t.Errorf("main.go should match *.go")
	}
	if !shouldScan("foo.go", cfg) {
		t.Errorf("foo.go should match *.go")
	}
	if shouldScan("a/b.go", cfg) {
		t.Errorf("a/b.go should NOT match *.go (filepath.Match does not cross /)")
	}
	if shouldScan("README.md", cfg) {
		t.Errorf("README.md should be excluded by Include filter")
	}
}

func TestShouldScan_DefaultIsEverything(t *testing.T) {
	cfg := config.ScanConfig{}
	if !shouldScan("anything.txt", cfg) {
		t.Errorf("with no include/exclude, everything should be scanned")
	}
}

func TestMapGitStatus(t *testing.T) {
	cases := map[string]string{
		"M":       "modified",
		"MM":      "modified",
		"A":       "added",
		"AM":      "added",
		"D":       "deleted",
		"R100":    "renamed",
		"R050":    "renamed",
		"X":       "modified",
		"":        "modified",
	}
	for in, want := range cases {
		if got := mapGitStatus(in); got != want {
			t.Errorf("mapGitStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseGitStatusLine(t *testing.T) {
	cases := []struct {
		line        string
		wantType    string
		wantPath    string
		wantPresent bool
	}{
		{"M\tsrc/main.go", "modified", "src/main.go", true},
		{"A\tnew.go", "added", "new.go", true},
		{"D\told.go", "deleted", "old.go", true},
		{"R100\told/name.go\tnew/name.go", "renamed", "new/name.go", true},
		{"R078\told.go\tnew.go", "renamed", "new.go", true},
		{"R100\tgo.mod", "renamed", "go.mod", true}, // 2-field rename (no new path) — fall back
		{"", "", "", false},
		{"only-one-field", "", "", false},
		{"MM\tsrc/main.go", "modified", "src/main.go", true}, // staged + unstaged
	}
	for _, tc := range cases {
		got, ok := parseGitStatusLine(tc.line)
		if ok != tc.wantPresent {
			t.Errorf("parseGitStatusLine(%q) ok = %v, want %v", tc.line, ok, tc.wantPresent)
			continue
		}
		if !ok {
			continue
		}
		if got.changeType != tc.wantType {
			t.Errorf("parseGitStatusLine(%q) type = %q, want %q", tc.line, got.changeType, tc.wantType)
		}
		if got.filePath != tc.wantPath {
			t.Errorf("parseGitStatusLine(%q) path = %q, want %q", tc.line, got.filePath, tc.wantPath)
		}
	}
}
