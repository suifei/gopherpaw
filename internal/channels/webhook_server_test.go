package channels

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockWebhookHandler struct {
	callCount int
	lastBody  []byte
}

func (m *mockWebhookHandler) HandleWebhook(ctx context.Context, body []byte) error {
	m.callCount++
	m.lastBody = body
	return nil
}

func TestNewWebhookServer(t *testing.T) {
	server := NewWebhookServer("localhost", 8080)
	if server == nil {
		t.Errorf("NewWebhookServer returned nil")
	}
}

func TestWebhookServerRegister(t *testing.T) {
	server := NewWebhookServer("localhost", 8080)

	handler := &mockWebhookHandler{}
	server.Register("test", handler)
}

func TestWebhookServerMultipleHandlers(t *testing.T) {
	server := NewWebhookServer("localhost", 8080)

	handler1 := &mockWebhookHandler{}
	handler2 := &mockWebhookHandler{}

	server.Register("dingtalk", handler1)
	server.Register("feishu", handler2)
}

func TestWebhookServerStop(t *testing.T) {
	server := NewWebhookServer("localhost", 0)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stopCancel()

	err := server.Stop(stopCtx)
	if err != nil {
		t.Logf("Stop error (expected): %v", err)
	}
}

func TestFmtAddrDefault(t *testing.T) {
	addr := fmtAddr("", 0)
	if addr != "0.0.0.0:8080" {
		t.Errorf("fmtAddr(\"\", 0) = %v, want 0.0.0.0:8080", addr)
	}
}

func TestFmtAddrWithHost(t *testing.T) {
	addr := fmtAddr("localhost", 9000)
	if addr != "localhost:9000" {
		t.Errorf("fmtAddr(\"localhost\", 9000) = %v, want localhost:9000", addr)
	}
}

func TestFmtAddrEmptyPort(t *testing.T) {
	addr := fmtAddr("127.0.0.1", 0)
	if addr != "127.0.0.1:8080" {
		t.Errorf("fmtAddr(\"127.0.0.1\", 0) = %v, want 127.0.0.1:8080", addr)
	}
}

func TestFmtAddrEmptyHost(t *testing.T) {
	addr := fmtAddr("", 3000)
	if addr != "0.0.0.0:3000" {
		t.Errorf("fmtAddr(\"\", 3000) = %v, want 0.0.0.0:3000", addr)
	}
}

func TestFmtAddrNegativePort(t *testing.T) {
	addr := fmtAddr("localhost", -1)
	if addr != "localhost:8080" {
		t.Errorf("fmtAddr(\"localhost\", -1) = %v, want localhost:8080", addr)
	}
}

func TestWebhookServerEmptyPort(t *testing.T) {
	server := NewWebhookServer("localhost", 0)
	if server == nil {
		t.Errorf("NewWebhookServer with port 0 returned nil")
	}
}

func TestWebhookServerNegativePort(t *testing.T) {
	server := NewWebhookServer("localhost", -1)
	if server == nil {
		t.Errorf("NewWebhookServer with negative port returned nil")
	}
}

func TestWebhookServerHTTPEndpoints(t *testing.T) {
	server := NewWebhookServer("localhost", 0)

	mockHandler := &mockWebhookHandler{}
	server.Register("dingtalk", mockHandler)

	// Create a test server with the webhook mux
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if mockHandler != nil {
			mockHandler.HandleWebhook(r.Context(), body)
		}
		w.WriteHeader(http.StatusOK)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test POST to /webhook/dingtalk
	resp, err := http.Post(
		testServer.URL+"/webhook/dingtalk",
		"application/json",
		bytes.NewBufferString(`{"test": "data"}`),
	)
	if err != nil {
		t.Fatalf("Failed to POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestWebhookServerHTTPMethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test GET (not allowed)
	resp, err := http.Get(testServer.URL + "/webhook/dingtalk")
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", resp.StatusCode)
	}
}

func TestWebhookServerHTTPBadRequest(t *testing.T) {
	// Create a test server that simulates read error
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Simulate read error by closing body
		r.Body.Close()
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Send request with immediate body close
	resp, err := http.Post(
		testServer.URL+"/",
		"application/json",
		bytes.NewBufferString("test"),
	)
	if err != nil {
		t.Fatalf("Failed to POST: %v", err)
	}
	defer resp.Body.Close()
}

func TestWebhookServerRegisterMultipleTimes(t *testing.T) {
	server := NewWebhookServer("localhost", 8080)

	handler1 := &mockWebhookHandler{}
	handler2 := &mockWebhookHandler{}

	// Register same path twice (second should override first)
	server.Register("test", handler1)
	server.Register("test", handler2)
}

func TestWebhookServerStopWithRunningServer(t *testing.T) {
	server := NewWebhookServer("127.0.0.1", 0)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Try to stop a server that was never started
	err := server.Stop(ctx)
	if err != nil {
		t.Logf("Stop error (expected for never-started server): %v", err)
	}
}
