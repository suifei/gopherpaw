package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForFeishu struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForFeishu) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "feishu mock response", nil
}

func (m *mockAgentForFeishu) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "feishu stream"
	close(ch)
	return ch, nil
}

func TestNewFeishu(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)
	if c.Name() != "feishu" {
		t.Errorf("Name() = %v, want feishu", c.Name())
	}
}

func TestFeishuEnabled(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestFeishuDisabled(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   false,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestFeishuDisabledNoAppID(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "",
		AppSecret: "test_secret",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without AppID")
	}
}

func TestFeishuStopWithoutStart(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*FeishuChannel)(nil)

func TestFeishuDisabledNoSecret(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without AppSecret")
	}
}

func TestFeishuSendDisabled(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled:   false,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	}
	c := NewFeishu(&mockAgentForFeishu{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test", nil)

	if err != nil {
		t.Errorf("Send with disabled channel should not error, got: %v", err)
	}
}
