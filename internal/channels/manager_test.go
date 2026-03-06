package channels

import (
	"context"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForManager struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForManager) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "manager mock response", nil
}

func (m *mockAgentForManager) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, chatID, text)
	}
	ch := make(chan string, 1)
	ch <- "manager stream response"
	close(ch)
	return ch, nil
}

func TestNewManager(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)
	if m == nil {
		t.Errorf("NewManager returned nil")
	}
}

func TestManagerChannels(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)
	channels := m.Channels()

	if len(channels) == 0 {
		t.Errorf("Manager should have at least console channel")
	}

	found := false
	for _, ch := range channels {
		if ch.Name() == "console" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("console channel not found in manager")
	}
}

func TestManagerRegister(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	newChannel := NewConsole(&mockAgentForManager{}, true, nil)
	m.Register(newChannel)

	channels := m.Channels()
	found := false
	for _, ch := range channels {
		if ch.Name() == "console" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("console channel not found after register")
	}
}

func TestManagerUnregister(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	channels := m.Channels()
	if len(channels) == 0 {
		t.Fatalf("expected channels, got none")
	}

	m.Unregister("console")

	channels = m.Channels()
	for _, ch := range channels {
		if ch.Name() == "console" {
			t.Errorf("console channel still found after unregister")
		}
	}
}

func TestManagerSend(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	ctx := context.Background()
	err := m.Send(ctx, "console", "user123", "test message")
	if err != nil {
		t.Errorf("Send error = %v", err)
	}
}

func TestManagerSetDaemonInfo(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	info := &agent.DaemonInfo{}
	m.SetDaemonInfo(info)
}

func TestManagerLastDispatch(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	ch, userID, sessionID := m.LastDispatch()
	if ch != "" || userID != "" || sessionID != "" {
		t.Errorf("LastDispatch should return empty initially")
	}
}

func TestManagerStopWithoutStart(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := m.Stop(ctx)
	if err != nil {
		t.Errorf("Stop error = %v", err)
	}
}

func TestManagerBuildChannels(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	channels := m.Channels()
	if len(channels) < 2 {
		t.Errorf("expected at least 2 channels, got %d", len(channels))
	}
}
