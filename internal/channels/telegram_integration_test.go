package channels

import (
	"context"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForTelegramIntegration struct {
	runFunc func(ctx context.Context, chatID, text string) (string, error)
}

func (m *mockAgentForTelegramIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "response", nil
}

func (m *mockAgentForTelegramIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

func TestTelegramIntegration_SendDisabled(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "test_token",
	}

	c := NewTelegram(&mockAgentForTelegramIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123456", "Hello, World!", nil)
	if err != nil {
		t.Errorf("Send with disabled channel should not error, got: %v", err)
	}
}

func TestTelegramIntegration_InvalidChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ctx := context.Background()
	err := c.Send(ctx, "invalid", "Hello, World!", nil)
	if err != nil {
		t.Errorf("Send with nil bot should not error, got: %v", err)
	}
}

func TestTelegramIntegration_StartStop(t *testing.T) {
	t.Skip("Start() requires real API connection, skipping integration test")

	cfg := config.TelegramConfig{
		Enabled:  false,
		BotToken: "",
	}

	c := NewTelegram(&mockAgentForTelegramIntegration{}, cfg, nil)

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

func TestTelegramIntegration_ContextCancellation(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Send(ctx, "123456", "Hello, World!", nil)
	if err != nil {
		t.Logf("Send with cancelled context returned error: %v", err)
	}
}

func TestTelegramIntegration_Timeout(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	c := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	err := c.Send(ctx, "123456", "Hello, World!", nil)
	if err != nil {
		t.Logf("Send with cancelled context: %v", err)
	}
}

func TestTelegramIntegration_NilBot(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ctx := context.Background()
	err := c.Send(ctx, "123456", "Hello, World!", nil)
	if err != nil {
		t.Errorf("Send with nil bot should not error, got: %v", err)
	}
}

func TestTelegramIntegration_MetaChatID(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ctx := context.Background()

	tests := []struct {
		name string
		to   string
		meta map[string]string
	}{
		{
			name: "meta_chat_id_override",
			to:   "123456",
			meta: map[string]string{"chat_id": "999999"},
		},
		{
			name: "empty_to_with_meta",
			to:   "",
			meta: map[string]string{"chat_id": "999999"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.to, "Hello, World!", tt.meta)
			if err != nil {
				t.Errorf("Send with nil bot should not error, got: %v", err)
			}
		})
	}
}

func TestTelegramIntegration_MultipleChannels(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	ch1 := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ch2 := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ctx := context.Background()

	err1 := ch1.Send(ctx, "123456", "Message 1", nil)
	err2 := ch2.Send(ctx, "789012", "Message 2", nil)

	if err1 != nil || err2 != nil {
		t.Errorf("Send with nil bot should not error, got: %v, %v", err1, err2)
	}
}

func TestTelegramIntegration_RapidSend(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := &TelegramChannel{
		agent:  &mockAgentForTelegramIntegration{},
		cfg:    cfg,
		bot:    nil,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		err := c.Send(ctx, "123456", "Message", nil)
		if err != nil {
			t.Errorf("Send %d with nil bot should not error, got: %v", i, err)
		}
	}
}
