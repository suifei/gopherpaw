package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForDingTalkStream struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForDingTalkStream) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "dingtalk stream mock response", nil
}

func (m *mockAgentForDingTalkStream) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "dingtalk stream response"
	close(ch)
	return ch, nil
}

func TestNewDingTalkStream(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		UseStream:    true,
	}
	c := NewDingTalkStream(&mockAgentForDingTalkStream{}, cfg, nil)
	if c.Name() != "dingtalk" {
		t.Errorf("Name() = %v, want dingtalk", c.Name())
	}
}

func TestDingTalkStreamEnabled(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		UseStream:    true,
	}
	c := NewDingTalkStream(&mockAgentForDingTalkStream{}, cfg, nil)
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() should be true")
	}
}

func TestDingTalkStreamDisabled(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      false,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		UseStream:    true,
	}
	c := NewDingTalkStream(&mockAgentForDingTalkStream{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false when disabled")
	}
}

func TestDingTalkStreamDisabledNoCredentials(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "",
		ClientSecret: "",
		UseStream:    true,
	}
	c := NewDingTalkStream(&mockAgentForDingTalkStream{}, cfg, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() should be false without credentials")
	}
}

func TestDingTalkStreamStopWithoutStart(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled:      true,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		UseStream:    true,
	}
	c := NewDingTalkStream(&mockAgentForDingTalkStream{}, cfg, nil)

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

var _ Channel = (*DingTalkStreamChannel)(nil)
