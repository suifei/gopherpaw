package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForDiscord struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForDiscord) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "discord mock response", nil
}

func (m *mockAgentForDiscord) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "discord stream"
	close(ch)
	return ch, nil
}

func TestNewDiscord(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)
	if c.Name() != "discord" {
		t.Errorf("Name() = %v, want discord", c.Name())
	}
}

func TestDiscordEnabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestDiscordDisabledNoToken(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without token")
	}
}

func TestDiscordDisabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestDiscordSendDisabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test message", nil)

	if err != nil {
		t.Errorf("Send should not error for disabled channel, got: %v", err)
	}
}

func TestDiscordStopWithoutStart(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*DiscordChannel)(nil)

func TestDiscordSendDisabledWithNilBot(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test", nil)

	// Should not error even though bot is nil (not initialized)
	if err != nil {
		t.Errorf("Send with nil bot should not error, got: %v", err)
	}
}

func TestDiscordEditMessageDisabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.EditMessage(ctx, "123", "msg456", "edited", nil)

	if err != nil {
		t.Errorf("EditMessage with disabled channel should not error, got: %v", err)
	}
}

func TestDiscordDeleteMessageDisabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.DeleteMessage(ctx, "123", "msg456", nil)

	if err != nil {
		t.Errorf("DeleteMessage with disabled channel should not error, got: %v", err)
	}
}

func TestDiscordReactDisabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.React(ctx, "123", "msg456", "👍", nil)

	if err != nil {
		t.Errorf("React with disabled channel should not error, got: %v", err)
	}
}

func TestDiscordSendTypingDisabled(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "test_token",
	}
	c := NewDiscord(&mockAgentForDiscord{}, cfg, nil)

	ctx := context.Background()
	err := c.SendTyping(ctx, "123", nil)

	if err != nil {
		t.Errorf("SendTyping with disabled channel should not error, got: %v", err)
	}
}
