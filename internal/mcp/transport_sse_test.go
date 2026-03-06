package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestSSETransport_Start(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.MCPServerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     config.MCPServerConfig{URL: "http://example.com/mcp/sse", Transport: "sse"},
			wantErr: false,
		},
		{
			name:    "empty url",
			cfg:     config.MCPServerConfig{Transport: "sse"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewSSETransport(tt.cfg)
			ctx := context.Background()
			err := transport.Start(ctx)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSSETransport_Stop(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp/sse", Transport: "sse"}
	transport := NewSSETransport(cfg)
	ctx := context.Background()

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := transport.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := transport.Stop(); err != nil {
		t.Fatalf("Stop again: %v", err)
	}
}

func TestSSETransport_IsRunning(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp/sse", Transport: "sse"}
	transport := NewSSETransport(cfg)
	ctx := context.Background()

	if transport.IsRunning() {
		t.Error("expected IsRunning to be false before Start")
	}

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !transport.IsRunning() {
		t.Error("expected IsRunning to be true after Start")
	}

	if err := transport.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if transport.IsRunning() {
		t.Error("expected IsRunning to be false after Stop")
	}
}

func TestSSETransport_Call(t *testing.T) {
	tests := []struct {
		name       string
		serverFunc http.HandlerFunc
		wantErr    bool
	}{
		{
			name: "successful call",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"success\":true}}\n\n"))
			},
			wantErr: false,
		},
		{
			name: "http error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal error"))
			},
			wantErr: true,
		},
		{
			name: "no sse data",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
		},
		{
			name: "invalid json in data",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("data: invalid json\n\n"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverFunc)
			defer server.Close()

			cfg := config.MCPServerConfig{URL: server.URL, Transport: "sse"}
			transport := NewSSETransport(cfg)
			ctx := context.Background()

			if err := transport.Start(ctx); err != nil {
				t.Fatalf("Start: %v", err)
			}

			req := jsonRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "test",
				Params:  map[string]any{},
			}
			var result map[string]any
			err := transport.Call(ctx, req, &result)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSSETransport_Call_NotRunning(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp/sse", Transport: "sse"}
	transport := NewSSETransport(cfg)
	ctx := context.Background()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]any{},
	}
	var result map[string]any
	err := transport.Call(ctx, req, &result)

	if err == nil {
		t.Fatal("expected error when transport not running, got nil")
	}
}

func TestSSETransport_WriteNotification(t *testing.T) {
	tests := []struct {
		name       string
		serverFunc http.HandlerFunc
		wantErr    bool
	}{
		{
			name: "successful notification",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "http error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverFunc)
			defer server.Close()

			cfg := config.MCPServerConfig{URL: server.URL, Transport: "sse"}
			transport := NewSSETransport(cfg)
			ctx := context.Background()

			if err := transport.Start(ctx); err != nil {
				t.Fatalf("Start: %v", err)
			}

			msg := map[string]any{
				"jsonrpc": "2.0",
				"method":  "test/notification",
			}
			err := transport.WriteNotification(msg)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSSETransport_WriteNotification_NotRunning(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp/sse", Transport: "sse"}
	transport := NewSSETransport(cfg)

	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  "test/notification",
	}
	err := transport.WriteNotification(msg)

	if err == nil {
		t.Fatal("expected error when transport not running, got nil")
	}
}

func TestSSETransport_Call_MultiLineData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"jsonrpc\":\"2.0\"\n"))
		w.Write([]byte("data: ,\"result\":{\"success\":true}}\n\n"))
	}))
	defer server.Close()

	cfg := config.MCPServerConfig{URL: server.URL, Transport: "sse"}
	transport := NewSSETransport(cfg)
	ctx := context.Background()

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]any{},
	}
	var result map[string]any
	err := transport.Call(ctx, req, &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSETransport_Call_WithContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {}\n\n"))
	}))
	defer server.Close()

	cfg := config.MCPServerConfig{URL: server.URL, Transport: "sse"}
	transport := NewSSETransport(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]any{},
	}
	var result map[string]any
	err := transport.Call(ctx, req, &result)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestSSETransport_WithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {}\n\n"))
	}))
	defer server.Close()

	cfg := config.MCPServerConfig{
		URL:       server.URL,
		Transport: "sse",
		Headers:   map[string]string{"X-Custom-Header": "test-value"},
	}
	transport := NewSSETransport(cfg)
	ctx := context.Background()

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]any{},
	}
	var result map[string]any
	err := transport.Call(ctx, req, &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSETransport_Call_InvalidRequestMarshal(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp/sse", Transport: "sse"}
	transport := NewSSETransport(cfg)
	ctx := context.Background()

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params: map[string]any{
			"invalid": make(chan int),
		},
	}
	var result map[string]any
	err := transport.Call(ctx, req, &result)

	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
}
