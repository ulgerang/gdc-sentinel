package note

import (
	"path/filepath"
	"testing"
)

func TestAddAndListByNode(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	n1, err := mgr.Add("node-a", "first note")
	if err != nil {
		t.Fatalf("Add node-a: %v", err)
	}
	n2, err := mgr.Add("node-b", "second note")
	if err != nil {
		t.Fatalf("Add node-b: %v", err)
	}

	if n1.ID == n2.ID {
		t.Fatalf("expected unique ids, got %q == %q", n1.ID, n2.ID)
	}

	a, err := mgr.ListByNode("node-a")
	if err != nil {
		t.Fatalf("ListByNode: %v", err)
	}
	if len(a) != 1 || a[0].Text != "first note" {
		t.Fatalf("ListByNode(node-a) = %+v, want exactly the first note", a)
	}

	all, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List() len = %d, want 2", len(all))
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	n, err := mgr.Add("node-x", "to delete")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := mgr.Delete(n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	all, _ := mgr.List()
	if len(all) != 0 {
		t.Fatalf("after delete, List() = %+v, want empty", all)
	}
}

func TestListOnEmptyDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	mgr := NewManager(dir)
	all, err := mgr.List()
	if err != nil {
		t.Fatalf("List on missing dir: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("List on missing dir returned %+v, want empty", all)
	}
}
