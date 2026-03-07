// Package mcp provides MCP (Model Context Protocol) client for connecting to external tool servers.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
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
			Key         string            `json:"key"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Transport   string            `json:"transport"`
			URL         string            `json:"url"`
			Headers     map[string]string `json:"headers"`
			Command     string            `json:"command"`
			Args        []string          `json:"args"`
			Env         map[string]string `json:"env"`
			Cwd         string            `json:"cwd"`
			Enabled     *bool             `json:"enabled"`
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
				Name:        single.Name,
				Description: single.Description,
				Transport:   single.Transport,
				URL:         single.URL,
				Headers:     single.Headers,
				Command:     single.Command,
				Args:        single.Args,
				Env:         single.Env,
				Cwd:         single.Cwd,
				Enabled:     single.Enabled,
			},
		}, nil
	}
	out := make(map[string]config.MCPServerConfig)
	for k, v := range raw {
		var cfg config.MCPServerConfig
		if err := json.Unmarshal(v, &cfg); err != nil {
			continue
		}
		if cfg.Command != "" || cfg.URL != "" {
			out[k] = cfg
		}
	}
	return out, nil
}

// ValidateConfig validates an MCP server configuration.
func ValidateConfig(name string, cfg config.MCPServerConfig) error {
	if name == "" {
		return fmt.Errorf("client name cannot be empty")
	}

	// Validate transport type
	validTransports := map[string]bool{"": true, "stdio": true, "streamable_http": true, "sse": true}
	if !validTransports[cfg.Transport] {
		return fmt.Errorf("invalid transport type: %s (supported: stdio, streamable_http, sse)", cfg.Transport)
	}

	// Validate based on transport type
	switch cfg.Transport {
	case "", "stdio":
		if cfg.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
		// Check if command exists in PATH (best effort)
		if !strings.Contains(cfg.Command, "/") && !strings.Contains(cfg.Command, "\\") {
			if _, err := lookPath(cfg.Command); err != nil {
				return fmt.Errorf("command %q not found in PATH: %w", cfg.Command, err)
			}
		}
	case "streamable_http", "sse":
		if cfg.URL == "" {
			return fmt.Errorf("url is required for %s transport", cfg.Transport)
		}
		// Validate URL format
		parsedURL, err := url.Parse(cfg.URL)
		if err != nil {
			return fmt.Errorf("invalid url %q: %w", cfg.URL, err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("url scheme must be http or https, got: %s", parsedURL.Scheme)
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("url must have a host")
		}
	}

	return nil
}

// lookPath searches for an executable in PATH (cross-platform helper).
func lookPath(file string) (string, error) {
	return execLookPath(file)
}

// execLookPath is a variable for testing purposes.
var execLookPath = func(file string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func init() {
	// Initialize execLookPath with the actual implementation
	execLookPath = func(file string) (string, error) {
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			return "", fmt.Errorf("PATH environment variable not set")
		}

		paths := strings.Split(pathEnv, string(os.PathListSeparator))
		for _, dir := range paths {
			fullPath := strings.Join([]string{dir, file}, string(os.PathSeparator))
			if info, err := os.Stat(fullPath); err == nil {
				if !info.IsDir() {
					return fullPath, nil
				}
			}
		}
		return "", fmt.Errorf("executable file not found in $PATH")
	}
}

// MCPClient represents a connection to a single MCP server.
type MCPClient struct {
	Name        string
	Description string
	Transport   Transport
	Enabled     bool

	// Reconnection support
	reconnectCfg    *ReconnectConfig
	reconnectCount  int
	stopReconnect   chan struct{}
	stopHealthCheck chan struct{}

	// RebuildInfo stores original config for client reconstruction
	rebuildInfo config.MCPServerConfig

	// Error recovery enhancements
	circuitBreaker *CircuitBreaker
	lastError      error
	errorCount     int
}

// ReconnectConfig holds reconnection settings.
type ReconnectConfig struct {
	Enabled      bool          // Enable auto-reconnect
	MaxRetries   int           // Max retry attempts (0 = infinite)
	InitialDelay time.Duration // Initial reconnect delay
	MaxDelay     time.Duration // Maximum reconnect delay

	// Error recovery enhancements
	HealthCheckInterval time.Duration         // Interval for health checks
	HealthCheckTimeout  time.Duration         // Timeout for health check requests
	CircuitBreaker      *CircuitBreakerConfig // Circuit breaker configuration
}

// CircuitBreakerConfig holds circuit breaker settings.
type CircuitBreakerConfig struct {
	Enabled          bool          // Enable circuit breaker
	FailureThreshold int           // Number of failures before opening circuit
	SuccessThreshold int           // Number of successes before closing circuit
	Timeout          time.Duration // Time to wait before attempting to close circuit
}

// DefaultCircuitBreakerConfig returns default circuit breaker settings.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// DefaultReconnectConfig returns default reconnection settings.
func DefaultReconnectConfig() *ReconnectConfig {
	return &ReconnectConfig{
		Enabled:             true,
		MaxRetries:          5,
		InitialDelay:        1 * time.Second,
		MaxDelay:            30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		CircuitBreaker:      DefaultCircuitBreakerConfig(),
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config          *CircuitBreakerConfig
	failures        int
	successes       int
	state           string // "closed", "open", "half-open"
	lastStateChange time.Time
	mu              sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &CircuitBreaker{
		config:          config,
		state:           "closed",
		lastStateChange: time.Now(),
	}
}

// CanExecute checks if the circuit breaker allows execution.
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.config.Enabled {
		return true
	}

	switch cb.state {
	case "closed":
		return true
	case "open":
		if time.Since(cb.lastStateChange) > cb.config.Timeout {
			cb.state = "half-open"
			cb.successes = 0
			cb.lastStateChange = time.Now()
			logger.L().Info("Circuit breaker half-open",
				zap.String("state", cb.state),
			)
			return true
		}
		return false
	case "half-open":
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful execution.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.config.Enabled {
		return
	}

	cb.failures = 0
	cb.successes++

	if cb.state == "half-open" && cb.successes >= cb.config.SuccessThreshold {
		cb.state = "closed"
		cb.successes = 0
		cb.lastStateChange = time.Now()
		logger.L().Info("Circuit breaker closed",
			zap.String("state", cb.state),
		)
	}
}

// RecordFailure records a failed execution.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.config.Enabled {
		return
	}

	cb.successes = 0
	cb.failures++

	if cb.state == "half-open" {
		cb.state = "open"
		cb.lastStateChange = time.Now()
		logger.L().Warn("Circuit breaker opened (from half-open)",
			zap.String("state", cb.state),
			zap.Int("failures", cb.failures),
		)
	} else if cb.state == "closed" && cb.failures >= cb.config.FailureThreshold {
		cb.state = "open"
		cb.lastStateChange = time.Now()
		logger.L().Warn("Circuit breaker opened",
			zap.String("state", cb.state),
			zap.Int("failures", cb.failures),
		)
	}
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// NewMCPClient creates a new MCP client from config.
func NewMCPClient(name string, cfg config.MCPServerConfig) (*MCPClient, error) {
	// Validate configuration
	if err := ValidateConfig(name, cfg); err != nil {
		return nil, err
	}

	enabled := cfg.Enabled == nil || *cfg.Enabled

	var t Transport
	switch cfg.Transport {
	case "", "stdio":
		t = NewStdioTransport(cfg)
	case "streamable_http":
		t = NewHTTPTransport(cfg)
	case "sse":
		t = NewSSETransport(cfg)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", cfg.Transport)
	}

	return &MCPClient{
		Name:            name,
		Description:     cfg.Description,
		Transport:       t,
		Enabled:         enabled,
		reconnectCfg:    DefaultReconnectConfig(),
		reconnectCount:  0,
		stopReconnect:   make(chan struct{}),
		stopHealthCheck: make(chan struct{}),
		rebuildInfo:     cfg,
		circuitBreaker:  NewCircuitBreaker(DefaultReconnectConfig().CircuitBreaker),
		errorCount:      0,
	}, nil
}

// StartWithReconnect starts the client with auto-reconnection support.
func (c *MCPClient) StartWithReconnect(ctx context.Context) error {
	if err := c.Transport.Start(ctx); err != nil {
		c.recordError(err)
		return err
	}

	c.recordSuccess()

	if c.reconnectCfg != nil && c.reconnectCfg.Enabled {
		go c.reconnectLoop(ctx)
		if c.reconnectCfg.HealthCheckInterval > 0 {
			go c.healthCheckLoop(ctx)
		}
	}

	return nil
}

// healthCheckLoop performs periodic health checks.
func (c *MCPClient) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(c.reconnectCfg.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopHealthCheck:
			return
		case <-ticker.C:
			if !c.checkHealth(ctx) {
				c.tryReconnect(ctx)
			}
		}
	}
}

// checkHealth performs a health check on the client.
func (c *MCPClient) checkHealth(ctx context.Context) bool {
	if !c.Transport.IsRunning() {
		return false
	}

	// Check circuit breaker
	if !c.circuitBreaker.CanExecute() {
		logger.L().Debug("Circuit breaker is open, skipping health check",
			zap.String("client", c.Name),
			zap.String("state", c.circuitBreaker.GetState()),
		)
		return false
	}

	// Try to list tools as a health check
	checkCtx, cancel := context.WithTimeout(ctx, c.reconnectCfg.HealthCheckTimeout)
	defer cancel()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      9999,
		Method:  "ping",
		Params:  map[string]string{},
	}
	var resp struct {
		Error *jsonRPCError `json:"error,omitempty"`
	}

	if err := c.Transport.Call(checkCtx, req, &resp); err != nil {
		c.recordError(err)
		logger.L().Debug("Health check failed",
			zap.String("client", c.Name),
			zap.Error(err),
		)
		return false
	}

	c.recordSuccess()
	return true
}

// recordError records an error for circuit breaker.
func (c *MCPClient) recordError(err error) {
	c.lastError = err
	c.errorCount++
	if c.circuitBreaker != nil {
		c.circuitBreaker.RecordFailure()
	}
}

// recordSuccess records a success for circuit breaker.
func (c *MCPClient) recordSuccess() {
	c.lastError = nil
	c.errorCount = 0
	if c.circuitBreaker != nil {
		c.circuitBreaker.RecordSuccess()
	}
}

// reconnectLoop monitors connection and reconnects on failure.
func (c *MCPClient) reconnectLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopReconnect:
			return
		case <-ticker.C:
			if !c.Transport.IsRunning() {
				c.tryReconnect(ctx)
			}
		}
	}
}

// tryReconnect attempts to reconnect with exponential backoff.
func (c *MCPClient) tryReconnect(ctx context.Context) {
	// Check circuit breaker
	if !c.circuitBreaker.CanExecute() {
		logger.L().Debug("Circuit breaker is open, skipping reconnect",
			zap.String("client", c.Name),
			zap.String("state", c.circuitBreaker.GetState()),
		)
		return
	}

	if c.reconnectCfg.MaxRetries > 0 && c.reconnectCount >= c.reconnectCfg.MaxRetries {
		logger.L().Warn("MCP client max retries reached",
			zap.String("client", c.Name),
			zap.Int("attempts", c.reconnectCount),
		)
		return
	}

	c.reconnectCount++
	delay := c.calculateBackoff()

	logger.L().Info("Attempting MCP client reconnect",
		zap.String("client", c.Name),
		zap.Int("attempt", c.reconnectCount),
		zap.Duration("delay", delay),
		zap.String("circuit_state", c.circuitBreaker.GetState()),
	)

	time.Sleep(delay)

	if err := c.Transport.Start(ctx); err != nil {
		c.recordError(err)
		logger.L().Warn("MCP client reconnect failed",
			zap.String("client", c.Name),
			zap.Error(err),
			zap.Int("error_count", c.errorCount),
		)
	} else {
		c.recordSuccess()
		c.reconnectCount = 0 // Reset on success
		logger.L().Info("MCP client reconnected successfully",
			zap.String("client", c.Name),
		)
	}
}

// calculateBackoff calculates exponential backoff delay.
func (c *MCPClient) calculateBackoff() time.Duration {
	delay := c.reconnectCfg.InitialDelay * time.Duration(1<<uint(c.reconnectCount-1))
	if delay > c.reconnectCfg.MaxDelay {
		delay = c.reconnectCfg.MaxDelay
	}
	return delay
}

// StopWithReconnect stops the client and reconnection loop.
func (c *MCPClient) StopWithReconnect() error {
	close(c.stopReconnect)
	close(c.stopHealthCheck)
	return c.Transport.Stop()
}

// GetRebuildInfo returns the original config for client reconstruction.
func (c *MCPClient) GetRebuildInfo() config.MCPServerConfig {
	return c.rebuildInfo
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
		client, err := NewMCPClient(name, c)
		if err != nil {
			logger.L().Warn("MCP client config error", zap.String("name", name), zap.Error(err))
			continue
		}
		m.clients[name] = client
	}
	return nil
}

// AddClient dynamically adds and starts an MCP client. Caller must hold context for startup.
func (m *MCPManager) AddClient(ctx context.Context, name string, cfg config.MCPServerConfig) error {
	client, err := NewMCPClient(name, cfg)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if _, exists := m.clients[name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("client %q already exists", name)
	}
	m.clients[name] = client
	m.mu.Unlock()

	if !client.Enabled {
		return nil
	}

	if err := client.Transport.Start(ctx); err != nil {
		m.mu.Lock()
		delete(m.clients, name)
		m.mu.Unlock()
		return err
	}

	tools, err := m.listClientTools(ctx, client)
	if err != nil {
		_ = client.Transport.Stop()
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
	_ = c.Transport.Stop()
	m.rebuildToolsLocked()
	return nil
}

func (m *MCPManager) rebuildToolsLocked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	var allTools []agent.Tool
	for _, c := range m.clients {
		if !c.Enabled || !c.Transport.IsRunning() {
			continue
		}
		tools, err := m.listClientTools(context.Background(), c)
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
		current[k] = c.GetRebuildInfo()
	}
	m.mu.Unlock()

	for name := range current {
		if _, ok := newConfigs[name]; !ok {
			_ = m.RemoveClient(ctx, name)
		}
	}
	for name, cfg := range newConfigs {
		// Skip invalid configs
		if cfg.Command == "" && cfg.URL == "" {
			continue
		}
		cur, ok := current[name]
		if !ok {
			_ = m.AddClient(ctx, name, cfg)
			continue
		}
		if configChanged(cur, cfg) {
			_ = m.RemoveClient(ctx, name)
			_ = m.AddClient(ctx, name, cfg)
		}
	}
	return nil
}

func configChanged(a, b config.MCPServerConfig) bool {
	if a.Transport != b.Transport {
		return true
	}
	if a.Command != b.Command || a.URL != b.URL || a.Cwd != b.Cwd {
		return true
	}
	if !mapsEqual(a.Env, b.Env) || !mapsEqual(a.Headers, b.Headers) {
		return true
	}
	if !slicesEqual(a.Args, b.Args) {
		return true
	}
	return false
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
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
		if err := c.Transport.Start(ctx); err != nil {
			logger.L().Warn("MCP client start failed", zap.String("name", name), zap.Error(err))
			continue
		}
		tools, err := m.listClientTools(ctx, c)
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
		if err := c.Transport.Stop(); err != nil {
			logger.L().Warn("MCP client stop failed", zap.String("name", name), zap.Error(err))
		}
	}
	m.tools = nil
	return nil
}

func (m *MCPManager) listClientTools(ctx context.Context, c *MCPClient) ([]agent.Tool, error) {
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
		Result struct{}      `json:"result"`
		Error  *jsonRPCError `json:"error,omitempty"`
	}
	if err := c.Transport.Call(ctx, initReq, &initResp); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}
	if initResp.Error != nil {
		return nil, fmt.Errorf("initialize error: %s", initResp.Error.Message)
	}

	// Send initialized notification (no response expected)
	c.Transport.WriteNotification(map[string]any{
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
	if err := c.Transport.Call(ctx, req, &resp); err != nil {
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
	if err := c.Transport.Call(ctx, req, &resp); err != nil {
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

// jsonRPCRequest represents a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// jsonRPCError represents a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// jsonRPCListResponse represents a tools/list response.
type jsonRPCListResponse struct {
	Result jsonRPCListResult `json:"result"`
	Error  *jsonRPCError     `json:"error,omitempty"`
}

// jsonRPCListResult represents the result of tools/list.
type jsonRPCListResult struct {
	Tools []mcpToolDef `json:"tools"`
}

// mcpToolDef represents an MCP tool definition.
type mcpToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// jsonRPCCallResponse represents a tools/call response.
type jsonRPCCallResponse struct {
	Result jsonRPCCallResult `json:"result"`
	Error  *jsonRPCError     `json:"error,omitempty"`
}

// jsonRPCCallResult represents the result of tools/call.
type jsonRPCCallResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// mcpToolAdapter adapts an MCP tool to agent.Tool interface.
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
