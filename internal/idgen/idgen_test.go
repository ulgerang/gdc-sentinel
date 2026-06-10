package idgen

import (
	"regexp"
	"testing"
)

func TestNew_Format(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{16}$`, id); !matched {
		t.Errorf("id %q does not match 16-char lowercase hex", id)
	}
}

func TestNew_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := New()
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}
