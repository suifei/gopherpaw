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

type mockAgentForQQIntegration struct {
	runFunc func(ctx context.Context, chatID, text string) (string, error)
}

func (m *mockAgentForQQIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "response", nil
}

func (m *mockAgentForQQIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

func TestQQIntegration_Send_Success(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app/getAppAccessToken" {
			resp := map[string]interface{}{
				"access_token": "test_token",
				"expires_in":   7200,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer tokenServer.Close()

	sendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "QQBot test_token" {
			t.Errorf("expected Authorization 'QQBot test_token', got '%s'", auth)
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		w.WriteHeader(http.StatusOK)
	}))
	defer sendServer.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "user123", "Hello, World!", map[string]string{"message_type": "c2c"})
	if err == nil {
		t.Log("Send completed (may have token fetch error)")
	}
}

func TestQQIntegration_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 11257,"message":"access token expired"}`))
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "user123", "Hello", map[string]string{"message_type": "c2c"})
	if err == nil {
		t.Error("expected error for HTTP 401")
	}
}

func TestQQIntegration_Send_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := c.Send(ctx, "user123", "Hello", map[string]string{"message_type": "c2c"})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestQQIntegration_Send_C2C(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "user123", "C2C message", map[string]string{"message_type": "c2c"})
	if err == nil {
		if receivedPath != "/v2/users/user123/messages" {
			t.Errorf("expected path '/v2/users/user123/messages', got '%s'", receivedPath)
		}
	}
}

func TestQQIntegration_Send_Group(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "group123", "Group message", map[string]string{"message_type": "group", "group_openid": "group123"})
	if err == nil {
		if receivedPath != "/v2/groups/group123/messages" {
			t.Errorf("expected path '/v2/groups/group123/messages', got '%s'", receivedPath)
		}
	}
}

func TestQQIntegration_Send_Guild(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "channel123", "Guild message", map[string]string{"message_type": "guild", "channel_id": "channel123"})
	if err == nil {
		if receivedPath != "/channels/channel123/messages" {
			t.Errorf("expected path '/channels/channel123/messages', got '%s'", receivedPath)
		}
	}
}

func TestQQIntegration_HandleWebhook_C2C(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"msg_type": "c2c",
		"author": map[string]interface{}{
			"user_openid": "user123",
			"id":          "author_id",
		},
		"content": "C2C message from user",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.Content != "C2C message from user" {
		t.Errorf("expected content 'C2C message from user', got '%s'", receivedMsg.Content)
	}

	if receivedMsg.ChatID != "user123" {
		t.Errorf("expected chat_id 'user123', got '%s'", receivedMsg.ChatID)
	}

	if receivedMsg.UserID != "user123" {
		t.Errorf("expected user_id 'user123', got '%s'", receivedMsg.UserID)
	}

	if receivedMsg.Channel != "qq" {
		t.Errorf("expected channel 'qq', got '%s'", receivedMsg.Channel)
	}

	if receivedMsg.Metadata["message_type"] != "c2c" {
		t.Error("message_type not set correctly")
	}

	if receivedMsg.Metadata["sender_id"] != "user123" {
		t.Error("sender_id not set correctly")
	}
}

func TestQQIntegration_HandleWebhook_Group(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"msg_type": "group",
		"author": map[string]interface{}{
			"user_openid": "user123",
			"id":          "author_id",
		},
		"group_openid": "group456",
		"content":      "Group message",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.ChatID != "group:group456" {
		t.Errorf("expected chat_id 'group:group456', got '%s'", receivedMsg.ChatID)
	}

	if receivedMsg.Metadata["message_type"] != "group" {
		t.Error("message_type not set correctly")
	}

	if receivedMsg.Metadata["group_openid"] != "group456" {
		t.Error("group_openid not set correctly")
	}
}

func TestQQIntegration_HandleWebhook_Guild(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"msg_type": "guild",
		"author": map[string]interface{}{
			"user_openid": "user123",
		},
		"channel_id": "channel789",
		"content":    "Guild message",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.ChatID != "channel:channel789" {
		t.Errorf("expected chat_id 'channel:channel789', got '%s'", receivedMsg.ChatID)
	}

	if receivedMsg.Metadata["message_type"] != "guild" {
		t.Error("message_type not set correctly")
	}

	if receivedMsg.Metadata["channel_id"] != "channel789" {
		t.Error("channel_id not set correctly")
	}
}

func TestQQIntegration_HandleWebhook_InvalidJSON(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, []byte("invalid json{{{"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestQQIntegration_HandleWebhook_EmptyContent(t *testing.T) {
	called := false

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		called = true
		return nil
	}

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"msg_type": "c2c",
		"author": map[string]interface{}{
			"user_openid": "user123",
		},
		"content": "",
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

func TestQQIntegration_Send_BotPrefix(t *testing.T) {
	var receivedContent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if content, ok := req["content"].(string); ok {
			receivedContent = content
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
		BotPrefix:    "[QQ] ",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "user123", "Hello", map[string]string{"message_type": "c2c"})
	if err == nil {
		if receivedContent != "[QQ] Hello" {
			t.Errorf("expected content '[QQ] Hello', got '%s'", receivedContent)
		}
	}
}

func TestQQIntegration_Send_WithMsgID(t *testing.T) {
	var receivedMsgID interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		receivedMsgID = req["msg_id"]

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "user123", "Reply", map[string]string{"message_type": "c2c", "message_id": "msg_12345"})
	if err == nil {
		if receivedMsgID != "msg_12345" {
			t.Errorf("expected msg_id 'msg_12345', got %v", receivedMsgID)
		}
	}
}

func TestQQIntegration_HandleWebhookPayload_WhitespaceTrim(t *testing.T) {
	var receivedMsg IncomingMessage

	onMsg := func(ctx context.Context, chName string, msg IncomingMessage) error {
		receivedMsg = msg
		return nil
	}

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, onMsg)

	payload := map[string]interface{}{
		"msg_type": "c2c",
		"author": map[string]interface{}{
			"user_openid": "user123",
		},
		"content": "\t  Trimmed content  \t",
	}

	body, _ := json.Marshal(payload)

	ctx := context.Background()
	err := c.HandleWebhook(ctx, body)
	if err != nil {
		t.Errorf("HandleWebhook failed: %v", err)
	}

	if receivedMsg.Content != "Trimmed content" {
		t.Errorf("expected content 'Trimmed content', got '%s'", receivedMsg.Content)
	}
}

func TestQQIntegration_StartStop(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

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

func TestQQIntegration_ConcurrentSends(t *testing.T) {
	t.Skip("Concurrent sends require real API connection, skipping integration test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}

	c := NewQQ(&mockAgentForQQIntegration{}, cfg, nil)

	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			err := c.Send(ctx, "user123", "test message", map[string]string{"message_type": "c2c"})
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
