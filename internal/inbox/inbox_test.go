package inbox

import (
	"path/filepath"
	"testing"

	"github.com/ulgerang/gdc-sentinel/internal/gdc"
)

func TestCreateAndListByNode(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	a := DriftItem{NodeID: "node-a", FilePath: "a.go", ChangeType: "modified"}
	b := DriftItem{NodeID: "node-b", FilePath: "b.go", ChangeType: "added"}
	for _, it := range []DriftItem{a, b} {
		if err := mgr.Create(it); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	got, err := mgr.ListByNode("node-a")
	if err != nil {
		t.Fatalf("ListByNode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListByNode(node-a) len = %d, want 1", len(got))
	}
	if got[0].FilePath != "a.go" {
		t.Errorf("got FilePath = %q, want a.go", got[0].FilePath)
	}
	if got[0].Status != "open" {
		t.Errorf("default Status = %q, want open", got[0].Status)
	}
}

func TestUpdateStatus(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	item := DriftItem{NodeID: "n", FilePath: "f"}
	if err := mgr.Create(item); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := mgr.ListByNode("n")
	if err != nil {
		t.Fatalf("ListByNode: %v", err)
	}
	if err := mgr.UpdateStatus(got[0].ID, "resolved"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	after, _ := mgr.ListByNode("n")
	if after[0].Status != "resolved" {
		t.Errorf("Status = %q, want resolved", after[0].Status)
	}
}

func TestListOnEmptyDir(t *testing.T) {
	mgr := NewManager(filepath.Join(t.TempDir(), "missing"))
	all, err := mgr.List()
	if err != nil {
		t.Fatalf("List on missing dir: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("List on missing dir returned %+v, want empty", all)
	}
}

func TestRoundTripWithDriftPayload(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	item := DriftItem{
		NodeID:     "n",
		FilePath:    "f",
		ChangeType:  "modified",
		Drift:       &gdc.DiffResponse{Node: "n", HasDrift: true, Drift: &gdc.DriftReport{MissingMethods: []string{"Foo"}}},
		QueryResult: &gdc.QueryResponse{ID: "n", CanonicalID: "pkg.N"},
	}
	if err := mgr.Create(item); err != nil {
		t.Fatalf("Create: %v", err)
	}
	loaded, _ := mgr.ListByNode("n")
	if loaded[0].Drift == nil || !loaded[0].Drift.HasDrift {
		t.Fatalf("drift payload lost in round-trip: %+v", loaded[0].Drift)
	}
	if loaded[0].QueryResult == nil || loaded[0].QueryResult.CanonicalID != "pkg.N" {
		t.Fatalf("query payload lost: %+v", loaded[0].QueryResult)
	}
}
