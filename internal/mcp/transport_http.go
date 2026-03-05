package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

// HTTPTransport implements Transport using HTTP POST requests.
type HTTPTransport struct {
	url     string
	headers map[string]string
	client  *http.Client

	mu      sync.Mutex
	running bool
}

// NewHTTPTransport creates a new HTTP transport from config.
func NewHTTPTransport(cfg config.MCPServerConfig) *HTTPTransport {
	return &HTTPTransport{
		url:     cfg.URL,
		headers: cfg.Headers,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Start validates configuration (HTTP transport is stateless).
func (t *HTTPTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.url == "" {
		return fmt.Errorf("url required for HTTP transport")
	}
	t.running = true
	return nil
}

// Stop marks the transport as stopped.
func (t *HTTPTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	return nil
}

// Call sends a JSON-RPC request via HTTP POST.
func (t *HTTPTransport) Call(ctx context.Context, req jsonRPCRequest, result interface{}) error {
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

	return json.NewDecoder(resp.Body).Decode(result)
}

// WriteNotification sends a notification via HTTP POST (no response expected).
func (t *HTTPTransport) WriteNotification(msg map[string]any) error {
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
	return nil
}

// IsRunning returns true if the transport is active.
func (t *HTTPTransport) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

var _ Transport = (*HTTPTransport)(nil)
