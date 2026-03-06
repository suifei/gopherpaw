package channels

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestManagerHandleMessageWithAgent(t *testing.T) {
	mockAg := &mockAgentForManager{
		runFunc: func(ctx context.Context, chatID, text string) (string, error) {
			return "processed: " + text, nil
		},
	}

	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(mockAg, cfg)

	msg := IncomingMessage{
		ChatID:   "test_chat",
		UserID:   "test_user",
		Content:  "hello agent",
		Channel:  "console",
		Metadata: nil,
	}

	ctx := context.Background()
	err := m.handleMessage(ctx, "console", msg)
	if err != nil {
		t.Errorf("handleMessage failed: %v", err)
	}
}

func TestManagerHandleMessageWithDaemonInfo(t *testing.T) {
	mockAg := &mockAgentForManager{}

	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(mockAg, cfg)

	daemonInfo := &agent.DaemonInfo{}
	m.SetDaemonInfo(daemonInfo)

	msg := IncomingMessage{
		ChatID:  "test_chat",
		UserID:  "test_user",
		Content: "test",
		Channel: "console",
	}

	ctx := context.Background()
	err := m.handleMessage(ctx, "console", msg)
	if err != nil {
		t.Errorf("handleMessage with DaemonInfo failed: %v", err)
	}
}

func TestManagerHandleMessageWithEmptyChatID(t *testing.T) {
	mockAg := &mockAgentForManager{}

	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(mockAg, cfg)

	msg := IncomingMessage{
		ChatID:  "",
		UserID:  "test_user",
		Content: "test",
		Channel: "console",
	}

	ctx := context.Background()
	err := m.handleMessage(ctx, "console", msg)
	if err != nil {
		t.Errorf("handleMessage with empty ChatID failed: %v", err)
	}

	ch, userID, sessionID := m.LastDispatch()
	if ch != "console" {
		t.Errorf("Expected channel 'console', got %q", ch)
	}
	if userID != "test_user" {
		t.Errorf("Expected userID 'test_user', got %q", userID)
	}
	if sessionID != "console:test_user" {
		t.Errorf("Expected sessionID 'console:test_user', got %q", sessionID)
	}
}

func TestManagerHandleMessageWithErrorFromAgent(t *testing.T) {
	mockAg := &mockAgentForManager{
		runFunc: func(ctx context.Context, chatID, text string) (string, error) {
			return "", fmt.Errorf("agent error")
		},
	}

	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(mockAg, cfg)

	msg := IncomingMessage{
		ChatID:  "test_chat",
		UserID:  "test_user",
		Content: "test",
		Channel: "console",
	}

	ctx := context.Background()
	err := m.handleMessage(ctx, "console", msg)
	if err != nil {
		t.Errorf("handleMessage should handle agent errors gracefully, got: %v", err)
	}
}

func TestManagerHandleMessageRecordsLastDispatchProperly(t *testing.T) {
	mockAg := &mockAgentForManager{}

	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(mockAg, cfg)

	msg1 := IncomingMessage{
		ChatID:  "chat1",
		UserID:  "user1",
		Content: "msg1",
		Channel: "console",
	}

	msg2 := IncomingMessage{
		ChatID:  "chat2",
		UserID:  "user2",
		Content: "msg2",
		Channel: "console",
	}

	ctx := context.Background()
	m.handleMessage(ctx, "console", msg1)

	ch, userID, sessionID := m.LastDispatch()
	if userID != "user1" || ch != "console" {
		t.Errorf("LastDispatch should record first message")
	}

	m.handleMessage(ctx, "console", msg2)

	ch, userID, sessionID = m.LastDispatch()
	if userID != "user2" {
		t.Errorf("LastDispatch should update to second message user")
	}
	if sessionID != "console:chat2" {
		t.Errorf("LastDispatch sessionID should be updated")
	}
}

func TestQueueDefaultConfig(t *testing.T) {
	cfg := DefaultQueueConfig()
	if cfg.MaxSize != 1000 {
		t.Errorf("Expected MaxSize 1000, got %d", cfg.MaxSize)
	}
	if cfg.Workers != 4 {
		t.Errorf("Expected Workers 4, got %d", cfg.Workers)
	}
}

func TestDebounceDefaultConfig(t *testing.T) {
	cfg := DefaultDebounceConfig()
	if !cfg.Enabled {
		t.Errorf("Expected Enabled true by default")
	}
	if cfg.DelayMs != 300 {
		t.Errorf("Expected DelayMs 300, got %d", cfg.DelayMs)
	}
	if cfg.MaxBufferSize != 10 {
		t.Errorf("Expected MaxBufferSize 10, got %d", cfg.MaxBufferSize)
	}
}

func TestTelegramChannelProperties(t *testing.T) {
	mockAg := newMockAgent()

	cfg1 := config.TelegramConfig{Enabled: true, BotToken: "token"}
	c1 := NewTelegram(mockAg, cfg1, nil)
	if !c1.IsEnabled() {
		t.Errorf("Telegram with token should be enabled")
	}

	cfg2 := config.TelegramConfig{Enabled: true, BotToken: ""}
	c2 := NewTelegram(mockAg, cfg2, nil)
	if c2.IsEnabled() {
		t.Errorf("Telegram without token should be disabled")
	}

	cfg3 := config.TelegramConfig{Enabled: false, BotToken: "token"}
	c3 := NewTelegram(mockAg, cfg3, nil)
	if c3.IsEnabled() {
		t.Errorf("Disabled Telegram should be disabled")
	}
}

func TestMessageQueueStartStop(t *testing.T) {
	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 100, Workers: 1},
		DebounceConfig{Enabled: false},
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	q.Start(ctx)

	err := q.Stop(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Stop() error = %v", err)
	}
}
