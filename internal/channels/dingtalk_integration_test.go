package channels

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForDingTalkIntegration struct {
	runFunc func(ctx context.Context, chatID, text string) (string, error)
}

func (m *mockAgentForDingTalkIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "response", nil
}

func (m *mockAgentForDingTalkIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

func TestDingTalkIntegration_Send_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if req["msgtype"] != "text" {
			t.Errorf("expected msgtype 'text', got %v", req["msgtype"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	webhookURL := server.URL
	err := c.Send(ctx, "conv123", "Hello, World!", map[string]string{"session_webhook": webhookURL})
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}
}

func TestDingTalkIntegration_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errcode":40001,"errmsg":"invalid credential"}`))
	}))
	defer server.Close()

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "conv123", "Hello", map[string]string{"session_webhook": server.URL})
	if err == nil {
		t.Error("expected error for HTTP 401")
	}
}

func TestDingTalkIntegration_Send_NoWebhook(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "conv123", "Hello", nil)
	if err == nil {
		t.Error("expected error for missing webhook URL")
	}
}

func TestDingTalkIntegration_Send_InvalidWebhook(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "conv123", "Hello", map[string]string{"session_webhook": "http://invalid-url-that-does-not-exist.local"})
	if err == nil {
		t.Error("expected error for invalid webhook URL")
	}
}

func TestDingTalkIntegration_Send_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := c.Send(ctx, "conv123", "Hello", map[string]string{"session_webhook": server.URL})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestDingTalkIntegration_HandleWebhook_Success(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"conversationId": "conv123",
		"senderId":       "user456",
		"senderNick":     "Test User",
		"text": map[string]string{
			"content": "Hello from webhook",
		},
		"sessionWebhook": "https://example.com/webhook",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.Content != "Hello from webhook" {
		t.Errorf("expected content 'Hello from webhook', got '%s'", receivedMsg.Content)
	}

	if receivedMsg.ChatID != "conv123" {
		t.Errorf("expected chat_id 'conv123', got '%s'", receivedMsg.ChatID)
	}

	if receivedMsg.UserID != "user456" {
		t.Errorf("expected user_id 'user456', got '%s'", receivedMsg.UserID)
	}

	if receivedMsg.Channel != "dingtalk" {
		t.Errorf("expected channel 'dingtalk', got '%s'", receivedMsg.Channel)
	}

	if receivedMsg.Metadata["session_webhook"] != "https://example.com/webhook" {
		t.Error("session_webhook not preserved in metadata")
	}
}

func TestDingTalkIntegration_HandleWebhook_EmptyContent(t *testing.T) {
	called := false

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		called = true
		return nil
	}

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"conversationId": "conv123",
		"senderId":       "user456",
		"senderNick":     "Test User",
		"text": map[string]string{
			"content": "",
		},
		"sessionWebhook": "https://example.com/webhook",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if called {
		t.Error("should not call onMsg for empty content")
	}
}

func TestDingTalkIntegration_HandleWebhook_InvalidJSON(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, []byte("invalid json{{{"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDingTalkIntegration_Send_BotPrefix(t *testing.T) {
	var receivedText string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if textObj, ok := req["text"].(map[string]interface{}); ok {
			if content, ok := textObj["content"].(string); ok {
				receivedText = content
			}
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		BotPrefix:    "[Bot] ",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "conv123", "Hello", map[string]string{"session_webhook": server.URL})
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	if receivedText != "[Bot] Hello" {
		t.Errorf("expected text '[Bot] Hello', got '%s'", receivedText)
	}
}

func TestDingTalkIntegration_ConcurrentSends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			err := c.Send(ctx, "conv123", "test message", map[string]string{"session_webhook": server.URL})
			if err != nil && idx < 5 {
				t.Errorf("Send %d failed: %v", idx, err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	t.Log("Concurrent sends completed")
}

func TestDingTalkIntegration_WebhookURL_Meta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name string
		meta map[string]string
	}{
		{
			name: "session_webhook",
			meta: map[string]string{"session_webhook": server.URL},
		},
		{
			name: "webhook_url",
			meta: map[string]string{"webhook_url": server.URL},
		},
		{
			name: "both_with_session_webhook_priority",
			meta: map[string]string{
				"session_webhook": server.URL + "/session",
				"webhook_url":     server.URL + "/webhook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, "conv123", "test", tt.meta)
			if err != nil {
				t.Errorf("Send failed: %v", err)
			}
		})
	}
}

func TestDingTalkIntegration_StartStop(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}

	c := NewDingTalk(&mockAgentForDingTalkIntegration{}, cfg, nil)

	ctx := context.Background()

	err := c.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	err = c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}
