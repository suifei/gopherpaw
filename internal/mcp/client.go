// Package mcp provides MCP (Model Context Protocol) client for connecting to external tool servers.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// ParseMCPConfig parses MCP config from JSON bytes. Supports three formats:
// 1. Standard: {"mcpServers": {"name": {"command": "...", ...}}}
// 2. Key-value: {"name": {"command": "...", ...}}
// 3. Single: {"key": "name", "command": "...", ...}
func ParseMCPConfig(jsonBytes []byte) (map[string]config.MCPServerConfig, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	if raw == nil {
		return nil, nil
	}
	if servers, ok := raw["mcpServers"]; ok {
		var m map[string]config.MCPServerConfig
		if err := json.Unmarshal(servers, &m); err != nil {
			return nil, fmt.Errorf("parse mcpServers: %w", err)
		}
		return m, nil
	}
	if _, hasKey := raw["key"]; hasKey {
		var single struct {
			Key     string `json:"key"`
			Command string `json:"command"`
			Args    []string `json:"args"`
			Env     map[string]string `json:"env"`
			Enabled *bool   `json:"enabled"`
		}
		if err := json.Unmarshal(jsonBytes, &single); err != nil {
			return nil, fmt.Errorf("parse single: %w", err)
		}
		name := single.Key
		if name == "" {
			name = "default"
		}
		return map[string]config.MCPServerConfig{
			name: {
				Command: single.Command,
				Args:    single.Args,
				Env:     single.Env,
				Enabled: single.Enabled,
			},
		}, nil
	}
	out := make(map[string]config.MCPServerConfig)
	for k, v := range raw {
		var cfg config.MCPServerConfig
		if err := json.Unmarshal(v, &cfg); err != nil {
			continue
		}
		if cfg.Command != "" {
			out[k] = cfg
		}
	}
	return out, nil
}

// MCPClient represents a connection to a single MCP server.
type MCPClient struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	Enabled bool

	cmd     *exec.Cmd
	stdin   *bufio.Writer
	stdout  *bufio.Scanner
	mu      sync.Mutex
	running bool
}

// MCPManager manages multiple MCP clients and provides tools.
type MCPManager struct {
	mu      sync.RWMutex
	clients map[string]*MCPClient
	tools   []agent.Tool
}

// NewManager creates an MCP manager.
func NewManager() *MCPManager {
	return &MCPManager{
		clients: make(map[string]*MCPClient),
		tools:   nil,
	}
}

// LoadConfig loads MCP server configurations.
func (m *MCPManager) LoadConfig(cfg map[string]config.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients = make(map[string]*MCPClient)
	for name, c := range cfg {
		if c.Command == "" {
			continue
		}
		enabled := c.Enabled == nil || *c.Enabled
		m.clients[name] = &MCPClient{
			Name:    name,
			Command: c.Command,
			Args:    c.Args,
			Env:     c.Env,
			Enabled: enabled,
		}
	}
	return nil
}

// AddClient dynamically adds and starts an MCP client. Caller must hold context for startup.
func (m *MCPManager) AddClient(ctx context.Context, name string, cfg config.MCPServerConfig) error {
	if cfg.Command == "" {
		return fmt.Errorf("command required")
	}
	enabled := cfg.Enabled == nil || *cfg.Enabled
	c := &MCPClient{
		Name:    name,
		Command: cfg.Command,
		Args:    cfg.Args,
		Env:     cfg.Env,
		Enabled: enabled,
	}
	m.mu.Lock()
	if _, exists := m.clients[name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("client %q already exists", name)
	}
	m.clients[name] = c
	m.mu.Unlock()
	if !enabled {
		return nil
	}
	if err := c.start(ctx); err != nil {
		m.mu.Lock()
		delete(m.clients, name)
		m.mu.Unlock()
		return err
	}
	tools, err := c.listTools(ctx)
	if err != nil {
		_ = c.stop()
		m.mu.Lock()
		delete(m.clients, name)
		m.mu.Unlock()
		return err
	}
	m.mu.Lock()
	if m.tools == nil {
		m.tools = tools
	} else {
		m.tools = append(m.tools, tools...)
	}
	m.mu.Unlock()
	return nil
}

// RemoveClient stops and removes an MCP client. Tools from this client are removed.
func (m *MCPManager) RemoveClient(ctx context.Context, name string) error {
	m.mu.Lock()
	c, ok := m.clients[name]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.clients, name)
	m.mu.Unlock()
	_ = c.stop()
	m.rebuildToolsLocked()
	return nil
}

func (m *MCPManager) rebuildToolsLocked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	var allTools []agent.Tool
	for _, c := range m.clients {
		if !c.Enabled || !c.running {
			continue
		}
		tools, err := c.listTools(context.Background())
		if err != nil {
			continue
		}
		allTools = append(allTools, tools...)
	}
	m.tools = allTools
}

// Reload compares newConfigs with current clients: add new, remove missing, update changed.
func (m *MCPManager) Reload(ctx context.Context, newConfigs map[string]config.MCPServerConfig) error {
	m.mu.Lock()
	current := make(map[string]config.MCPServerConfig)
	for k, c := range m.clients {
		current[k] = config.MCPServerConfig{
			Command: c.Command,
			Args:    c.Args,
			Env:     c.Env,
			Enabled: ptr(c.Enabled),
		}
	}
	m.mu.Unlock()
	for name := range current {
		if _, ok := newConfigs[name]; !ok {
			_ = m.RemoveClient(ctx, name)
		}
	}
	for name, cfg := range newConfigs {
		if cfg.Command == "" {
			continue
		}
		cur, ok := current[name]
		if !ok {
			_ = m.AddClient(ctx, name, cfg)
			continue
		}
		if cur.Command != cfg.Command || !slicesEqual(cur.Args, cfg.Args) {
			_ = m.RemoveClient(ctx, name)
			_ = m.AddClient(ctx, name, cfg)
		}
	}
	return nil
}

func ptr(b bool) *bool { return &b }

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GetTools returns all tools from enabled MCP clients.
func (m *MCPManager) GetTools() []agent.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.tools == nil {
		return nil
	}
	out := make([]agent.Tool, len(m.tools))
	copy(out, m.tools)
	return out
}

// Start starts all enabled MCP clients and fetches their tools.
func (m *MCPManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var allTools []agent.Tool
	for name, c := range m.clients {
		if !c.Enabled {
			continue
		}
		if err := c.start(ctx); err != nil {
			logger.L().Warn("MCP client start failed", zap.String("name", name), zap.Error(err))
			continue
		}
		tools, err := c.listTools(ctx)
		if err != nil {
			logger.L().Warn("MCP list tools failed", zap.String("name", name), zap.Error(err))
			continue
		}
		for _, t := range tools {
			allTools = append(allTools, t)
		}
	}
	m.tools = allTools
	return nil
}

// Stop stops all MCP clients.
func (m *MCPManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, c := range m.clients {
		if err := c.stop(); err != nil {
			logger.L().Warn("MCP client stop failed", zap.String("name", name), zap.Error(err))
		}
	}
	m.tools = nil
	return nil
}

func (c *MCPClient) start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return nil
	}
	c.cmd = exec.CommandContext(ctx, c.Command, c.Args...)
	c.cmd.Env = envToSlice(c.Env)
	c.cmd.Stderr = nil
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	c.stdin = bufio.NewWriter(stdin)
	c.stdout = bufio.NewScanner(stdout)
	c.running = true
	return nil
}

func (c *MCPClient) stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return nil
	}
	c.running = false
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	c.cmd = nil
	return nil
}

func (c *MCPClient) listTools(ctx context.Context) ([]agent.Tool, error) {
	// Send initialize (required by many MCP servers)
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "gopherpaw", "version": "0.1"},
		},
	}
	var initResp struct {
		Result struct{}     `json:"result"`
		Error  *jsonRPCError `json:"error,omitempty"`
	}
	if err := c.call(ctx, initReq, &initResp); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}
	if initResp.Error != nil {
		return nil, fmt.Errorf("initialize error: %s", initResp.Error.Message)
	}
	// Send initialized notification (no response expected)
	c.writeNotification(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  map[string]string{},
	}
	var resp jsonRPCListResponse
	if err := c.call(ctx, req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}
	var tools []agent.Tool
	for _, t := range resp.Result.Tools {
		tools = append(tools, &mcpToolAdapter{
			client: c,
			name:   t.Name,
			desc:   t.Description,
			schema: t.InputSchema,
		})
	}
	return tools, nil
}

func (c *MCPClient) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      name,
			"arguments": args,
		},
	}
	var resp jsonRPCCallResponse
	if err := c.call(ctx, req, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("tools/call error: %s", resp.Error.Message)
	}
	if len(resp.Result.Content) > 0 {
		return resp.Result.Content[0].Text, nil
	}
	return "", nil
}

func (c *MCPClient) call(ctx context.Context, req jsonRPCRequest, result interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return fmt.Errorf("client not running")
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := c.stdin.Write(data); err != nil {
		return err
	}
	if err := c.stdin.WriteByte('\n'); err != nil {
		return err
	}
	if err := c.stdin.Flush(); err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	done := make(chan struct{})
	var scanErr error
	go func() {
		if c.stdout.Scan() {
			scanErr = json.Unmarshal(c.stdout.Bytes(), result)
		} else {
			scanErr = c.stdout.Err()
			if scanErr == nil {
				scanErr = fmt.Errorf("unexpected EOF")
			}
		}
		close(done)
	}()
	select {
	case <-done:
		return scanErr
	case <-time.After(time.Until(deadline)):
		return fmt.Errorf("timeout waiting for MCP response")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *MCPClient) writeNotification(msg map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.stdin.Write(data)
	c.stdin.WriteByte('\n')
	return c.stdin.Flush()
}

func envToSlice(env map[string]string) []string {
	base := []string{}
	for k, v := range env {
		base = append(base, k+"="+v)
	}
	return base
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCListResponse struct {
	Result  jsonRPCListResult `json:"result"`
	Error   *jsonRPCError     `json:"error,omitempty"`
}

type jsonRPCListResult struct {
	Tools []mcpToolDef `json:"tools"`
}

type mcpToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type jsonRPCCallResponse struct {
	Result  jsonRPCCallResult `json:"result"`
	Error   *jsonRPCError     `json:"error,omitempty"`
}

type jsonRPCCallResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type mcpToolAdapter struct {
	client *MCPClient
	name   string
	desc   string
	schema interface{}
}

func (m *mcpToolAdapter) Name() string {
	return m.name
}

func (m *mcpToolAdapter) Description() string {
	return m.desc
}

func (m *mcpToolAdapter) Parameters() any {
	return m.schema
}

func (m *mcpToolAdapter) Execute(ctx context.Context, arguments string) (string, error) {
	var args map[string]any
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("invalid arguments JSON: %w", err)
		}
	}
	if args == nil {
		args = make(map[string]any)
	}
	return m.client.callTool(ctx, m.name, args)
}

var _ agent.Tool = (*mcpToolAdapter)(nil)
