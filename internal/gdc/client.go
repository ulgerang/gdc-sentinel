package gdc

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Command    string
	WorkingDir string

	commands map[string][]string
}

func NewClient(gdcCommand string, workingDir string, commands map[string][]string) *Client {
	if commands == nil {
		commands = defaultGDCCommands()
	}
	return &Client{
		Command:    gdcCommand,
		WorkingDir: workingDir,
		commands:   commands,
	}
}

func (c *Client) Query(symbol string) (*QueryResponse, error) {
	out, err := c.run("query", map[string]string{"symbol": symbol})
	if err != nil {
		return nil, fmt.Errorf("gdc query %s: %w", symbol, err)
	}
	var resp QueryResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse query response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Deps(nodeID string, depth int) (*DepsResponse, error) {
	out, err := c.run("deps", map[string]string{"node": nodeID, "depth": strconv.Itoa(depth)})
	if err != nil {
		return nil, fmt.Errorf("gdc deps %s: %w", nodeID, err)
	}
	var resp DepsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse deps response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Refs(nodeID string, depth int) (*RefsResponse, error) {
	out, err := c.run("refs", map[string]string{"node": nodeID, "depth": strconv.Itoa(depth)})
	if err != nil {
		return nil, fmt.Errorf("gdc refs %s: %w", nodeID, err)
	}
	var resp RefsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Context(nodeID string, withImpl, withTests, withCallers bool) (*ContextResponse, error) {
	vars := map[string]string{"node": nodeID}
	if withImpl {
		vars["with_impl"] = "--with-impl"
	}
	if withTests {
		vars["with_tests"] = "--with-tests"
	}
	if withCallers {
		vars["with_callers"] = "--with-callers"
	}

	out, err := c.run("context", vars)
	if err != nil {
		return nil, fmt.Errorf("gdc context %s: %w", nodeID, err)
	}
	var resp ContextResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse context response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Diff(nodeID string) (*DiffResponse, error) {
	out, err := c.run("diff", map[string]string{"node": nodeID})
	if err != nil {
		return nil, fmt.Errorf("gdc diff %s: %w", nodeID, err)
	}
	var resp DiffResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse diff response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Check() (*CheckResponse, error) {
	out, err := c.run("check", nil)
	if err != nil {
		return nil, fmt.Errorf("gdc check: %w", err)
	}
	var resp CheckResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse check response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Graph() (*GraphResponse, error) {
	out, err := c.run("graph", nil)
	if err != nil {
		return nil, fmt.Errorf("gdc graph: %w", err)
	}
	var resp GraphResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse graph response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Extract(nodeID string) (string, error) {
	out, err := c.run("extract", map[string]string{"node": nodeID})
	if err != nil {
		return "", fmt.Errorf("gdc extract %s: %w", nodeID, err)
	}
	return string(out), nil
}

func (c *Client) run(subcmd string, vars map[string]string) ([]byte, error) {
	template, ok := c.commands[subcmd]
	if !ok {
		template = []string{subcmd}
	}
	args := make([]string, 0, len(template))
	for _, piece := range template {
		args = append(args, expandVars(piece, vars))
	}
	allArgs := args

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Command, allArgs...)
	cmd.Dir = c.WorkingDir

	stdout, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return nil, fmt.Errorf("command %s %v: %w\nstderr: %s", c.Command, allArgs, err, stderr)
	}

	return stdout, nil
}

func expandVars(s string, vars map[string]string) string {
	if !strings.Contains(s, "{") {
		return s
	}
	out := s
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

func defaultGDCCommands() map[string][]string {
	return map[string][]string{
		"query":   {"query", "{symbol}", "--format", "json"},
		"deps":    {"deps", "{node}", "--depth", "{depth}"},
		"refs":    {"refs", "{node}", "--depth", "{depth}"},
		"context": {"context", "{node}", "{with_impl}", "{with_tests}", "{with_callers}"},
		"diff":    {"diff", "{node}", "--format", "json"},
		"check":   {"check", "--format", "json"},
		"graph":   {"graph", "--format", "json"},
		"extract": {"extract", "{node}"},
	}
}
