package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForTelegram struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForTelegram) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "telegram mock response", nil
}

func (m *mockAgentForTelegram) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "telegram stream"
	close(ch)
	return ch, nil
}

func TestNewTelegram(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegram{}, cfg, nil)
	if c.Name() != "telegram" {
		t.Errorf("Name() = %v, want telegram", c.Name())
	}
}

func TestTelegramEnabled(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegram{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestTelegramDisabledNoToken(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "",
	}
	c := NewTelegram(&mockAgentForTelegram{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without token")
	}
}

func TestTelegramDisabled(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegram{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestTelegramSendDisabled(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegram{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test message", nil)

	if err != nil {
		t.Errorf("Send should not error for disabled channel, got: %v", err)
	}
}

func TestTelegramStopWithoutStart(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegram{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*TelegramChannel)(nil)

type mockAgentForTelegramAdvanced struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForTelegramAdvanced) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "telegram mock response", nil
}

func (m *mockAgentForTelegramAdvanced) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, chatID, text)
	}
	ch := make(chan string, 1)
	ch <- "telegram stream"
	close(ch)
	return ch, nil
}

func TestTelegramWithProxy(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:   true,
		BotToken:  "test_token",
		HTTPProxy: "http://proxy.example.com:8080",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)
	if c.Name() != "telegram" {
		t.Errorf("Name() = %v, want telegram", c.Name())
	}
}

func TestTelegramSendWithMeta(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"msg_id": "12345",
	}
	err := c.Send(ctx, "123", "test message", meta)

	if err != nil {
		t.Errorf("Send with disabled channel should not error, got: %v", err)
	}
}

func TestTelegramSendMarkdown(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	err := c.SendMarkdown(ctx, "123", "**bold** text", nil)

	if err != nil {
		t.Errorf("SendMarkdown with disabled channel should not error, got: %v", err)
	}
}

func TestTelegramEditMessage(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	err := c.EditMessage(ctx, "123", "456", "edited text", nil)

	if err != nil {
		t.Errorf("EditMessage with disabled channel should not error, got: %v", err)
	}
}

func TestTelegramDeleteMessage(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	err := c.DeleteMessage(ctx, "123", "456", nil)

	if err != nil {
		t.Errorf("DeleteMessage with disabled channel should not error, got: %v", err)
	}
}

func TestTelegramSendTyping(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	err := c.SendTyping(ctx, "123", nil)

	if err != nil {
		t.Errorf("SendTyping with disabled channel should not error, got: %v", err)
	}
}

func TestTelegramSendMetaWithChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"chat_id": "999",
	}
	err := c.Send(ctx, "123", "test", meta)

	// Should not error even with different chat_id in meta
	if err != nil {
		t.Errorf("Send with meta chat_id should not error, got: %v", err)
	}
}

func TestTelegramEditMessageMetaWithChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"chat_id": "999",
	}
	err := c.EditMessage(ctx, "123", "456", "edited", meta)

	if err != nil {
		t.Errorf("EditMessage with meta should not error, got: %v", err)
	}
}

func TestTelegramDeleteMessageMetaWithChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"chat_id": "999",
	}
	err := c.DeleteMessage(ctx, "123", "456", meta)

	if err != nil {
		t.Errorf("DeleteMessage with meta should not error, got: %v", err)
	}
}

func TestTelegramSendTypeMetaWithChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"chat_id": "999",
	}
	err := c.SendTyping(ctx, "123", meta)

	if err != nil {
		t.Errorf("SendTyping with meta should not error, got: %v", err)
	}
}

func TestTelegramSendMarkdownMetaWithChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"chat_id": "999",
	}
	err := c.SendMarkdown(ctx, "123", "**bold**", meta)

	if err != nil {
		t.Errorf("SendMarkdown with meta should not error, got: %v", err)
	}
}

func TestTelegramSendFileMetaWithChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewTelegram(&mockAgentForTelegramAdvanced{}, cfg, nil)

	ctx := context.Background()
	meta := map[string]string{
		"chat_id": "999",
	}
	err := c.SendFile(ctx, "123", "/tmp/file.pdf", "application/pdf", meta)

	if err != nil {
		t.Errorf("SendFile with meta should not error, got: %v", err)
	}
}
