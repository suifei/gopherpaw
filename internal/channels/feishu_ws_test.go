package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForFeishuWS struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForFeishuWS) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "feishu ws mock response", nil
}

func (m *mockAgentForFeishuWS) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "feishu ws stream"
	close(ch)
	return ch, nil
}

func TestNewFeishuWS(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		AppSecret:    "test_secret",
		UseWebSocket: true,
	}
	c := NewFeishuWS(&mockAgentForFeishuWS{}, cfg, nil)
	if c.Name() != "feishu" {
		t.Errorf("Name() = %v, want feishu", c.Name())
	}
}

func TestFeishuWSEnabled(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		AppSecret:    "test_secret",
		UseWebSocket: true,
	}
	c := NewFeishuWS(&mockAgentForFeishuWS{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestFeishuWSDisabled(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:      false,
		AppID:        "test_app_id",
		AppSecret:    "test_secret",
		UseWebSocket: true,
	}
	c := NewFeishuWS(&mockAgentForFeishuWS{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestFeishuWSDisabledNoSecret(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		AppSecret:    "",
		UseWebSocket: true,
	}
	c := NewFeishuWS(&mockAgentForFeishuWS{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without AppSecret")
	}
}

func TestFeishuWSStopWithoutStart(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		AppSecret:    "test_secret",
		UseWebSocket: true,
	}
	c := NewFeishuWS(&mockAgentForFeishuWS{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*FeishuWSChannel)(nil)
