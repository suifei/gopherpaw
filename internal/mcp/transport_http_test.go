package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestHTTPTransport_Start(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.MCPServerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     config.MCPServerConfig{URL: "http://example.com/mcp", Transport: "streamable_http"},
			wantErr: false,
		},
		{
			name:    "empty url",
			cfg:     config.MCPServerConfig{Transport: "streamable_http"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewHTTPTransport(tt.cfg)
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

func TestHTTPTransport_Stop(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp", Transport: "streamable_http"}
	transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_IsRunning(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp", Transport: "streamable_http"}
	transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_Call(t *testing.T) {
	tests := []struct {
		name       string
		serverFunc http.HandlerFunc
		wantErr    bool
	}{
		{
			name: "successful call",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"result":  map[string]any{"success": true},
				})
			},
			wantErr: false,
		},
		{
			name: "http error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
			},
			wantErr: true,
		},
		{
			name: "invalid response",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverFunc)
			defer server.Close()

			cfg := config.MCPServerConfig{URL: server.URL, Transport: "streamable_http"}
			transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_Call_NotRunning(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp", Transport: "streamable_http"}
	transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_WriteNotification(t *testing.T) {
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

			cfg := config.MCPServerConfig{URL: server.URL, Transport: "streamable_http"}
			transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_WriteNotification_NotRunning(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp", Transport: "streamable_http"}
	transport := NewHTTPTransport(cfg)

	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  "test/notification",
	}
	err := transport.WriteNotification(msg)

	if err == nil {
		t.Fatal("expected error when transport not running, got nil")
	}
}

func TestHTTPTransport_Call_WithContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"result":  map[string]any{},
		})
	}))
	defer server.Close()

	cfg := config.MCPServerConfig{URL: server.URL, Transport: "streamable_http"}
	transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_WithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"result":  map[string]any{},
		})
	}))
	defer server.Close()

	cfg := config.MCPServerConfig{
		URL:       server.URL,
		Transport: "streamable_http",
		Headers:   map[string]string{"X-Custom-Header": "test-value"},
	}
	transport := NewHTTPTransport(cfg)
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

func TestHTTPTransport_Call_InvalidRequestMarshal(t *testing.T) {
	cfg := config.MCPServerConfig{URL: "http://example.com/mcp", Transport: "streamable_http"}
	transport := NewHTTPTransport(cfg)
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
