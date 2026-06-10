package gdc

import (
	"testing"
)

func TestExpandVars(t *testing.T) {
	cases := []struct {
		in   string
		vars map[string]string
		want string
	}{
		{"query {symbol} --format json", map[string]string{"symbol": "Foo"}, "query Foo --format json"},
		{"no vars here", nil, "no vars here"},
		{"{a}-{b}-{a}", map[string]string{"a": "x", "b": "y"}, "x-y-x"},
		{"{missing} stays", map[string]string{"other": "v"}, "{missing} stays"},
	}
	for _, tc := range cases {
		got := expandVars(tc.in, tc.vars)
		if got != tc.want {
			t.Errorf("expandVars(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNewClient_NilCommands_UsesDefaults(t *testing.T) {
	c := NewClient("gdc", "/tmp", nil)
	if c.commands == nil {
		t.Fatalf("default commands should be populated when nil is passed")
	}
	if _, ok := c.commands["query"]; !ok {
		t.Errorf("default commands missing 'query'")
	}
}

func TestNewClient_CustomCommands(t *testing.T) {
	cmds := map[string][]string{
		"query": {"q", "{symbol}"},
	}
	c := NewClient("gdc", "/tmp", cmds)
	if c.commands["query"][0] != "q" {
		t.Errorf("custom command not used, got %v", c.commands["query"])
	}
}

func TestClient_Run_BuildsCommandFromConfig(t *testing.T) {
	tmp := t.TempDir()
	fake := writeFakeBinary(t, tmp, "fake-gdc")

	cmds := map[string][]string{
		"query": {"query", "{symbol}", "--format", "json"},
	}
	c := NewClient(fake, tmp, cmds)
	resp, err := c.Query("MySymbol")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if resp == nil || resp.ID != "query" {
		t.Errorf("expected resp.ID == \"query\" (first argv), got %+v", resp)
	}
}

func TestClient_Run_UnknownSubcmdFallsBackToSubcmdAsArg(t *testing.T) {
	tmp := t.TempDir()
	fake := writeFakeBinary(t, tmp, "fake-gdc")
	c := NewClient(fake, tmp, nil)
	c.commands = map[string][]string{} // force unknown
	out, err := c.run("subcmd", map[string]string{"node": "N"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	wantPrefix := `{"id":"subcmd"}`
	if string(out) != wantPrefix {
		t.Errorf("unknown subcmd should be passed as-is, got %q (want prefix %q)", string(out), wantPrefix)
	}
}

func TestClient_Run_CapturesStderr(t *testing.T) {
	tmp := t.TempDir()
	fake := writeFakeBinary(t, tmp, "fake-gdc-err")
	cmds := map[string][]string{"query": {"query", "{symbol}", "--format", "json"}}
	c := NewClient(fake, tmp, cmds)
	_, err := c.Query("X")
	if err == nil {
		t.Fatalf("expected error from fake failing binary")
	}
}
