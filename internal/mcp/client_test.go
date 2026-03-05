package mcp_test

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/mcp"
)

func ptrBool(b bool) *bool { return &b }

func TestParseMCPConfig(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   map[string]config.MCPServerConfig
		hasErr bool
	}{
		{
			name:  "mcpServers format",
			input: `{"mcpServers": {"server1": {"command": "echo", "args": ["--test"]}}}`,
			want: map[string]config.MCPServerConfig{
				"server1": {
					Command:   "echo",
					Args:      []string{"--test"},
					Enabled:   ptrBool(true),
					Transport: "stdio",
				},
			},
		},
		{
			name:  "key-value format",
			input: `{"server2": {"command": "echo", "args": ["--test2"]}}`,
			want: map[string]config.MCPServerConfig{
				"server2": {
					Command:   "echo",
					Args:      []string{"--test2"},
					Enabled:   ptrBool(true),
					Transport: "stdio",
				},
			},
		},
		{
			name:  "single format",
			input: `{"key": "server3", "command": "echo", "args": ["--test3"]}`,
			want: map[string]config.MCPServerConfig{
				"server3": {
					Command:   "echo",
					Args:      []string{"--test3"},
					Enabled:   ptrBool(true),
					Transport: "stdio",
				},
			},
		},
		{
			name:  "http transport",
			input: `{"server4": {"url": "http://example.com/mcp", "transport": "streamable_http"}}`,
			want: map[string]config.MCPServerConfig{
				"server4": {
					URL:       "http://example.com/mcp",
					Transport: "streamable_http",
					Enabled:   ptrBool(true),
				},
			},
		},
		{
			name:  "sse transport",
			input: `{"server5": {"url": "http://example.com/mcp/sse", "transport": "sse"}}`,
			want: map[string]config.MCPServerConfig{
				"server5": {
					URL:       "http://example.com/mcp/sse",
					Transport: "sse",
					Enabled:   ptrBool(true),
				},
			},
		},
		{
			name:  "empty input",
			input: `{}`,
			want:  map[string]config.MCPServerConfig{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mcp.ParseMCPConfig([]byte(tt.input))
			if tt.hasErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMCPConfig() err = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Errorf("len(got)=%d len(want)=%d", len(got), len(tt.want))
			}
			for k, w := range tt.want {
				g, ok := got[k]
				if !ok {
					t.Errorf("missing key %q in result", k)
					continue
				}
				transportEq := g.Transport == w.Transport || (g.Transport == "" && w.Transport == "stdio")
				if !transportEq || g.Command != w.Command || g.URL != w.URL {
					t.Errorf("key %q: got transport=%q command=%q url=%q; want transport=%q command=%q url=%q",
						k, g.Transport, g.Command, g.URL, w.Transport, w.Command, w.URL)
				}
			}
		})
	}
}

func TestNewMCPClient_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.MCPServerConfig
		wantErr string
	}{
		{
			name:    "missing command for stdio",
			cfg:     config.MCPServerConfig{Transport: "stdio"},
			wantErr: "command required for stdio transport",
		},
		{
			name:    "missing url for streamable_http",
			cfg:     config.MCPServerConfig{Transport: "streamable_http"},
			wantErr: "url required for streamable_http transport",
		},
		{
			name:    "missing url for sse",
			cfg:     config.MCPServerConfig{Transport: "sse"},
			wantErr: "url required for sse transport",
		},
		{
			name:    "unsupported transport",
			cfg:     config.MCPServerConfig{Transport: "invalid", URL: "http://x"},
			wantErr: "unsupported transport: invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mcp.NewMCPClient("test", tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("err = %q; want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMCPManager_LoadConfig_GetTools(t *testing.T) {
	ctx := context.Background()
	mgr := mcp.NewManager()
	cfg := map[string]config.MCPServerConfig{
		"stdio1": {Command: "echo", Args: []string{"hello"}, Transport: "stdio"},
	}
	if err := mgr.LoadConfig(cfg); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer mgr.Stop()
	tools := mgr.GetTools()
	// echo is not an MCP server, so list tools may fail and we get no tools; just ensure no panic
	_ = tools
}

func TestParseMCPConfig_InvalidJSON(t *testing.T) {
	_, err := mcp.ParseMCPConfig([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseMCPConfig_SingleFormatWithDescription(t *testing.T) {
	input := `{"key": "s1", "name": "My Server", "description": "Test", "command": "echo", "args": []}`
	got, err := mcp.ParseMCPConfig([]byte(input))
	if err != nil {
		t.Fatalf("ParseMCPConfig: %v", err)
	}
	c, ok := got["s1"]
	if !ok {
		t.Fatal("expected key s1")
	}
	if c.Name != "My Server" || c.Description != "Test" {
		t.Errorf("name=%q description=%q", c.Name, c.Description)
	}
}
