package ignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

const DefaultFilename = ".gdc-sentinelignore"

type Matcher struct {
	impl     *gitignore.GitIgnore
	patterns []string
	root     string
}

func Load(projectRoot string) (*Matcher, error) {
	path := filepath.Join(projectRoot, DefaultFilename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Matcher{root: projectRoot}, nil
	}
	return LoadFile(path)
}

func LoadFile(path string) (*Matcher, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ignore file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	impl := gitignore.CompileIgnoreLines(lines...)

	return &Matcher{
		impl:     impl,
		patterns: lines,
		root:     filepath.Dir(path),
	}, nil
}

func (m *Matcher) ShouldIgnore(path string) bool {
	if m.impl == nil {
		return false
	}
	rel, err := filepath.Rel(m.root, path)
	if err != nil {
		return false
	}
	return m.impl.MatchesPath(rel)
}

func (m *Matcher) Patterns() []string {
	return m.patterns
}

func DefaultContent() string {
	return `# GDC Sentinel Ignore File
# Syntax: same as .gitignore (glob patterns, # comments, ! negation)

# === Non-code files ===
*.log
*.tmp
*.swp
*.swo
*.test
*.out
*.bak
*.cache

# === Build artifacts ===
dist/
build/
target/
bin/
out/
*.o
*.a
*.so
*.exe
*.dll

# === Package managers ===
vendor/
node_modules/

# === IDE / OS ===
.idea/
.vscode/
*.iml
.DS_Store
Thumbs.db

# === GDC Sentinel internals ===
.gdc-sentinel/

# === Git ===
.git/

# === Lockfiles and generated ===
package-lock.json
yarn.lock
go.sum
*.pb.go
*.gen.go
*_test.go

# === Only watch source code (negation patterns) ===
# Uncomment to ONLY track specific file types:
# !*.go
# !*.ts
# !*.tsx
# !*.py
`
}
