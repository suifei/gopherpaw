package channels

import (
	"context"
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
