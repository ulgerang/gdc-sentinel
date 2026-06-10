package gdc

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFakeBinary drops a shell script that records its argv. A file name
// ending in "fake-gdc-err" produces a script that exits 1 and writes to
// stderr; any other name produces a script that returns minimal valid JSON
// containing the joined args (used to assert command construction and to
// keep JSON-parsing code paths happy).
func writeFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	var script string
	if len(name) >= 12 && name[len(name)-12:] == "fake-gdc-err" {
		script = "#!/bin/sh\necho boom >&2\nexit 1\n"
	} else {
		script = "#!/bin/sh\nprintf '{\"id\":\"'\nfor a in \"$@\"; do :; done\nprintf '%s' \"$1\"\nprintf '\"}'\n"
	}
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}
