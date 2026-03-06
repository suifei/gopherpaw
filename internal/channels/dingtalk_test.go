package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForDingTalk struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForDingTalk) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "dingtalk mock response", nil
}

func (m *mockAgentForDingTalk) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "dingtalk stream"
	close(ch)
	return ch, nil
}

func TestNewDingTalk(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)
	if c.Name() != "dingtalk" {
		t.Errorf("Name() = %v, want dingtalk", c.Name())
	}
}

func TestDingTalkEnabled(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestDingTalkDisabled(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      false,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestDingTalkStopWithoutStart(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*DingTalkChannel)(nil)

func TestDingTalkDisabledNoClientID(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "",
		ClientSecret: "test_secret",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without ClientID")
	}
}

func TestDingTalkDisabledNoSecret(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without ClientSecret")
	}
}

func TestDingTalkSendDisabled(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      false,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
	}
	c := NewDingTalk(&mockAgentForDingTalk{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test", nil)

	if err != nil {
		t.Errorf("Send with disabled channel should not error, got: %v", err)
	}
}
