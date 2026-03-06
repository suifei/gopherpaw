package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForQQ struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForQQ) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "qq mock response", nil
}

func (m *mockAgentForQQ) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "qq stream"
	close(ch)
	return ch, nil
}

func TestNewQQ(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)
	if c.Name() != "qq" {
		t.Errorf("Name() = %v, want qq", c.Name())
	}
}

func TestQQEnabled(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestQQDisabled(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      false,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestQQDisabledNoAppID(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "",
		ClientSecret: "test_secret",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without AppID")
	}
}

func TestQQStopWithoutStart(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*QQChannel)(nil)

func TestQQDisabledNoSecret(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "test_app_id",
		ClientSecret: "",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without ClientSecret")
	}
}

func TestQQSendDisabled(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      false,
		AppID:        "test_app_id",
		ClientSecret: "test_secret",
	}
	c := NewQQ(&mockAgentForQQ{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test", nil)

	if err != nil {
		t.Errorf("Send with disabled channel should not error, got: %v", err)
	}
}
