package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

// SSETransport implements Transport using HTTP POST with SSE (Server-Sent Events) response.
type SSETransport struct {
	url     string
	headers map[string]string
	client  *http.Client

	mu      sync.Mutex
	running bool
}

// NewSSETransport creates a new SSE transport from config.
func NewSSETransport(cfg config.MCPServerConfig) *SSETransport {
	return &SSETransport{
		url:     cfg.URL,
		headers: cfg.Headers,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Start validates the URL and marks the transport as running.
// SSE transport is stateless, so Start is a no-op beyond validation.
func (t *SSETransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.url == "" {
		return fmt.Errorf("url required for SSE transport")
	}
	t.running = true
	return nil
}

// Stop marks the transport as stopped.
func (t *SSETransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	return nil
}

// Call sends a JSON-RPC request via HTTP POST and reads SSE response.
func (t *SSETransport) Call(ctx context.Context, req jsonRPCRequest, result interface{}) error {
	t.mu.Lock()
	running := t.running
	t.mu.Unlock()

	if !running {
		return fmt.Errorf("transport not running")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse SSE stream - look for "data:" lines
	scanner := bufio.NewScanner(resp.Body)
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		}
		if line == "" && len(dataLines) > 0 {
			// Empty line signals end of event
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SSE stream: %w", err)
	}

	if len(dataLines) == 0 {
		return fmt.Errorf("no SSE data received")
	}

	// Join data lines and parse as JSON
	combined := strings.Join(dataLines, "\n")
	return json.Unmarshal([]byte(combined), result)
}

// WriteNotification sends a notification via HTTP POST (no response expected).
func (t *SSETransport) WriteNotification(msg map[string]any) error {
	t.mu.Lock()
	running := t.running
	t.mu.Unlock()

	if !running {
		return fmt.Errorf("transport not running")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// IsRunning returns true if the transport is active.
func (t *SSETransport) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

var _ Transport = (*SSETransport)(nil)
