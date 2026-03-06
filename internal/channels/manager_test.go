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

func TestManagerHandleMessage(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ChannelsConfig
		msg     IncomingMessage
		wantErr bool
	}{
		{
			name: "successful message handling",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: true},
			},
			msg: IncomingMessage{
				ChatID:   "console:default",
				UserID:   "user123",
				UserName: "testuser",
				Content:  "hello",
				Channel:  "console",
			},
			wantErr: false,
		},
		{
			name: "message with nil metadata",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: true},
			},
			msg: IncomingMessage{
				ChatID:   "console:default",
				UserID:   "user123",
				Content:  "hello",
				Channel:  "console",
				Metadata: nil,
			},
			wantErr: false,
		},
		{
			name: "message with empty chat_id",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: true},
			},
			msg: IncomingMessage{
				ChatID:  "",
				UserID:  "user123",
				Content: "hello",
				Channel: "console",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(&mockAgentForManager{}, tt.cfg)
			ctx := context.Background()
			err := m.handleMessage(ctx, "console", tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerHandleMessageWithFileSender(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	msg := IncomingMessage{
		ChatID:  "console:default",
		UserID:  "user123",
		Content: "test",
		Channel: "console",
	}

	ctx := context.Background()
	err := m.handleMessage(ctx, "console", msg)
	if err != nil {
		t.Errorf("handleMessage with FileSender context error = %v", err)
	}
}

func TestManagerHandleMessageRecordsDispatch(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	msg := IncomingMessage{
		ChatID:  "chat123",
		UserID:  "user456",
		Content: "test message",
		Channel: "console",
	}

	ctx := context.Background()
	m.handleMessage(ctx, "console", msg)

	ch, userID, sessionID := m.LastDispatch()
	if ch != "console" {
		t.Errorf("LastDispatch channel = %v, want console", ch)
	}
	if userID != "user456" {
		t.Errorf("LastDispatch userID = %v, want user456", userID)
	}
	if sessionID != "console:chat123" {
		t.Errorf("LastDispatch sessionID = %v, want console:chat123", sessionID)
	}
}

func TestManagerStart(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: false}, // Disable console to avoid blocking
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start should not return error
	err := m.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Stop after starting
	err = m.Stop(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestManagerSendToUnregisteredChannel(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForManager{}, cfg)

	ctx := context.Background()
	err := m.Send(ctx, "nonexistent", "user123", "test message")
	if err != nil {
		t.Errorf("Send to unregistered channel should not error, got: %v", err)
	}
}

func TestManagerBuildChannelsWithMultipleChannels(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
		Discord: config.DiscordConfig{
			Enabled:  true,
			BotToken: "discord_token",
		},
		DingTalk: config.DingTalkConfig{
			Enabled:      true,
			ClientID:     "id",
			ClientSecret: "secret",
		},
		QQ: config.QQConfig{
			Enabled:      true,
			AppID:        "qq_id",
			ClientSecret: "qq_secret",
		},
	}
	m := NewManager(&mockAgentForManager{}, cfg)
	channels := m.Channels()

	// At least console, telegram, discord, dingtalk, qq
	if len(channels) < 5 {
		t.Errorf("expected at least 5 channels, got %d", len(channels))
	}
}
