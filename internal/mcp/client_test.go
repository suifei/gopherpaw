package mcp

import (
	"testing"
)

func TestParseMCPConfig_Standard(t *testing.T) {
	json := `{"mcpServers":{"foo":{"command":"npx","args":["-y","@modelcontextprotocol/server-filesystem"],"enabled":true}}}`
	cfg, err := ParseMCPConfig([]byte(json))
	if err != nil {
		t.Fatalf("ParseMCPConfig: %v", err)
	}
	if len(cfg) != 1 {
		t.Fatalf("expected 1 server, got %d", len(cfg))
	}
	f, ok := cfg["foo"]
	if !ok {
		t.Fatal("expected foo")
	}
	if f.Command != "npx" {
		t.Errorf("command: got %q", f.Command)
	}
	if len(f.Args) != 2 {
		t.Errorf("args: got %d", len(f.Args))
	}
}

func TestParseMCPConfig_KeyValue(t *testing.T) {
	json := `{"bar":{"command":"echo","args":["hello"]}}`
	cfg, err := ParseMCPConfig([]byte(json))
	if err != nil {
		t.Fatalf("ParseMCPConfig: %v", err)
	}
	if len(cfg) != 1 {
		t.Fatalf("expected 1 server, got %d", len(cfg))
	}
	b, ok := cfg["bar"]
	if !ok {
		t.Fatal("expected bar")
	}
	if b.Command != "echo" {
		t.Errorf("command: got %q", b.Command)
	}
}

func TestParseMCPConfig_Single(t *testing.T) {
	json := `{"key":"single","command":"cat","args":[]}`
	cfg, err := ParseMCPConfig([]byte(json))
	if err != nil {
		t.Fatalf("ParseMCPConfig: %v", err)
	}
	if len(cfg) != 1 {
		t.Fatalf("expected 1 server, got %d", len(cfg))
	}
	s, ok := cfg["single"]
	if !ok {
		t.Fatal("expected single")
	}
	if s.Command != "cat" {
		t.Errorf("command: got %q", s.Command)
	}
}

func TestParseMCPConfig_Empty(t *testing.T) {
	cfg, err := ParseMCPConfig([]byte(`{}`))
	if err != nil {
		t.Fatalf("ParseMCPConfig: %v", err)
	}
	if cfg != nil && len(cfg) != 0 {
		t.Errorf("expected empty, got %d", len(cfg))
	}
}

func TestParseMCPConfig_Invalid(t *testing.T) {
	_, err := ParseMCPConfig([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error")
	}
}
