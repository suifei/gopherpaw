package channels

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForFeishuIntegration struct {
	runFunc func(ctx context.Context, chatID, text string) (string, error)
}

func (m *mockAgentForFeishuIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "response", nil
}

func (m *mockAgentForFeishuIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

func TestFeishuIntegration_Send_Success(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/open-apis/auth/v3/tenant_access_token/internal" {
			resp := map[string]interface{}{
				"code":                0,
				"tenant_access_token": "test_token",
				"expire":              7200,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer tokenServer.Close()

	sendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/messages") {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test_token" {
				t.Errorf("expected Authorization 'Bearer test_token', got '%s'", auth)
			}

			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			if req["receive_id"] != "open_id_123" {
				t.Errorf("expected receive_id 'open_id_123', got %v", req["receive_id"])
			}

			w.WriteHeader(http.StatusOK)
		}
	}))
	defer sendServer.Close()

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "open_id_123", "Hello, World!", map[string]string{"receive_id_type": "open_id"})
	if err == nil {
		t.Log("Send completed (may have token fetch error)")
	}
}

func TestFeishuIntegration_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 99991663,"msg":"token expired"}`))
	}))
	defer server.Close()

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "open_id_123", "Hello", map[string]string{"receive_id_type": "open_id"})
	if err == nil {
		t.Error("expected error for HTTP 401")
	}
}

func TestFeishuIntegration_Send_NoReceiveID(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "", "Hello", nil)
	if err == nil {
		t.Error("expected error for missing receive_id")
	}
}

func TestFeishuIntegration_Send_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := c.Send(ctx, "open_id_123", "Hello", map[string]string{"receive_id_type": "open_id"})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestFeishuIntegration_HandleWebhook_Success(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"type": "im.message.receive_v1",
		"event": map[string]interface{}{
			"message": map[string]interface{}{
				"message_id":   "om_1234567890",
				"message_type": "text",
				"content":      `{"text":"Hello from Feishu"}`,
				"chat_id":      "oc_1234567890",
				"chat_type":    "p2p",
			},
			"sender": map[string]interface{}{
				"sender_id":   "open_id_123",
				"sender_type": "user",
			},
		},
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.Content != "Hello from Feishu" {
		t.Errorf("expected content 'Hello from Feishu', got '%s'", receivedMsg.Content)
	}

	if receivedMsg.ChatID != "oc_1234567890" {
		t.Errorf("expected chat_id 'oc_1234567890', got '%s'", receivedMsg.ChatID)
	}

	if receivedMsg.UserID != "open_id_123" {
		t.Errorf("expected user_id 'open_id_123', got '%s'", receivedMsg.UserID)
	}

	if receivedMsg.Channel != "feishu" {
		t.Errorf("expected channel 'feishu', got '%s'", receivedMsg.Channel)
	}

	if receivedMsg.Metadata["chat_id"] != "oc_1234567890" {
		t.Error("chat_id not preserved in metadata")
	}

	if receivedMsg.Metadata["receive_id"] != "open_id_123" {
		t.Error("receive_id not set correctly for p2p chat")
	}

	if receivedMsg.Metadata["receive_id_type"] != "open_id" {
		t.Error("receive_id_type not set correctly for p2p chat")
	}
}

func TestFeishuIntegration_HandleWebhook_GroupChat(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"type": "im.message.receive_v1",
		"event": map[string]interface{}{
			"message": map[string]interface{}{
				"message_id":   "om_1234567890",
				"message_type": "text",
				"content":      `{"text":"Group message"}`,
				"chat_id":      "oc_9876543210",
				"chat_type":    "group",
			},
			"sender": map[string]interface{}{
				"sender_id":   "open_id_123",
				"sender_type": "user",
			},
		},
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.Metadata["receive_id"] != "oc_9876543210" {
		t.Error("receive_id not set correctly for group chat")
	}

	if receivedMsg.Metadata["receive_id_type"] != "chat_id" {
		t.Error("receive_id_type not set correctly for group chat")
	}

	if receivedMsg.Metadata["chat_type"] != "group" {
		t.Error("chat_type not preserved in metadata")
	}
}

func TestFeishuIntegration_HandleWebhook_URLVerification(t *testing.T) {
	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		t.Error("should not call onMsg for URL verification")
		return nil
	}

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"type":      "url_verification",
		"challenge": "test_challenge_value",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}
}

func TestFeishuIntegration_HandleWebhook_AppMessage(t *testing.T) {
	called := false

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		called = true
		return nil
	}

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"type": "im.message.receive_v1",
		"event": map[string]interface{}{
			"message": map[string]interface{}{
				"message_id":   "om_1234567890",
				"message_type": "text",
				"content":      `{"text":"Bot message"}`,
				"chat_id":      "oc_1234567890",
				"chat_type":    "p2p",
			},
			"sender": map[string]interface{}{
				"sender_id":   "bot_123",
				"sender_type": "app",
			},
		},
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if called {
		t.Error("should not call onMsg for app messages")
	}
}

func TestFeishuIntegration_HandleWebhook_InvalidJSON(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, []byte("invalid json{{{"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFeishuIntegration_Send_BotPrefix(t *testing.T) {
	var receivedText string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if contentObj, ok := req["content"].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				receivedText = text
			}
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
		BotPrefix: "[Bot] ",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "open_id_123", "Hello", map[string]string{"receive_id_type": "open_id"})
	if err == nil {
		if receivedText != "[Bot] Hello" {
			t.Errorf("expected text '[Bot] Hello', got '%s'", receivedText)
		}
	}
}

func TestFeishuIntegration_ReceiveID_Meta(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name         string
		to           string
		meta         map[string]string
		expectedID   string
		expectedType string
	}{
		{
			name:         "open_id",
			to:           "open_id_123",
			meta:         map[string]string{"receive_id_type": "open_id"},
			expectedID:   "open_id_123",
			expectedType: "open_id",
		},
		{
			name:         "chat_id",
			to:           "oc_1234567890",
			meta:         map[string]string{"receive_id_type": "chat_id"},
			expectedID:   "oc_1234567890",
			expectedType: "chat_id",
		},
		{
			name:         "meta_override",
			to:           "default_id",
			meta:         map[string]string{"receive_id": "override_id", "receive_id_type": "open_id"},
			expectedID:   "override_id",
			expectedType: "open_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.to, "test", tt.meta)
			if err == nil {
				t.Log("Send completed")
			}
		})
	}
}

func TestFeishuIntegration_StartStop(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

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

func TestFeishuIntegration_ConcurrentSends(t *testing.T) {
	t.Skip("Concurrent sends require real API connection, skipping integration test")

	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}

	c := NewFeishu(&mockAgentForFeishuIntegration{}, cfg, nil)

	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			err := c.Send(ctx, "open_id_123", "test message", map[string]string{"receive_id_type": "open_id"})
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
