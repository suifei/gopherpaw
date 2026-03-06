package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForManagerIntegration struct {
	runFunc func(ctx context.Context, chatID, text string) (string, error)
}

func (m *mockAgentForManagerIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "manager response", nil
}

func (m *mockAgentForManagerIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

func TestManagerIntegration_StartStopAllChannels(t *testing.T) {
	cfg := config.ChannelsConfig{
		DingTalk: config.DingTalkConfig{
			Enabled:      true,
			ClientID:     "test_id",
			ClientSecret: "test_secret",
		},
		Feishu: config.FeishuConfig{
			Enabled:   true,
			AppID:     "test_app_id",
			AppSecret: "test_secret",
		},
		QQ: config.QQConfig{
			Enabled:      true,
			AppID:        "test_app_id",
			ClientSecret: "test_secret",
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	ctx := context.Background()

	err := m.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	channels := m.Channels()
	if len(channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(channels))
	}

	err = m.Stop(ctx)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestManagerIntegration_RouteToChannel(t *testing.T) {
	cfg := config.ChannelsConfig{
		DingTalk: config.DingTalkConfig{
			Enabled:      true,
			ClientID:     "test_id",
			ClientSecret: "test_secret",
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	ctx := context.Background()

	err := m.Send(ctx, "dingtalk", "user123", "test message to dingtalk")
	if err != nil {
		t.Logf("Send to dingtalk (may fail): %v", err)
	}

	err = m.Send(ctx, "nonexistent", "user789", "test message to nonexistent channel")
	if err != nil {
		t.Errorf("Send to nonexistent channel should not error, got: %v", err)
	}
}

func TestManagerIntegration_RegisterChannel(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	initialChannels := m.Channels()
	if len(initialChannels) != 1 {
		t.Errorf("expected 1 initial channel, got %d", len(initialChannels))
	}

	newChannel := NewConsole(&mockAgentForManagerIntegration{}, true, nil)
	m.Register(newChannel)

	updatedChannels := m.Channels()
	if len(updatedChannels) != 1 {
		t.Errorf("expected 1 channel after register (replace), got %d", len(updatedChannels))
	}

	if updatedChannels[0] != newChannel {
		t.Error("console channel should be replaced")
	}
}

func TestManagerIntegration_UnregisterChannel(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	initialChannels := m.Channels()
	if len(initialChannels) != 2 {
		t.Errorf("expected 2 initial channels, got %d", len(initialChannels))
	}

	m.Unregister("console")

	updatedChannels := m.Channels()
	if len(updatedChannels) != 1 {
		t.Errorf("expected 1 channel after unregister, got %d", len(updatedChannels))
	}

	if updatedChannels[0].Name() != "telegram" {
		t.Errorf("expected 'telegram' channel, got '%s'", updatedChannels[0].Name())
	}
}

func TestManagerIntegration_ReplaceChannel(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	newChannel := NewConsole(&mockAgentForManagerIntegration{}, true, nil)
	m.Register(newChannel)

	channels := m.Channels()
	if len(channels) != 1 {
		t.Errorf("expected 1 channel after replace, got %d", len(channels))
	}

	if channels[0] != newChannel {
		t.Error("channel should be replaced")
	}
}

func TestManagerIntegration_LastDispatch(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	channel, userID, sessionID := m.LastDispatch()

	if channel != "" || userID != "" || sessionID != "" {
		t.Error("expected empty last dispatch before any message")
	}

	m.recordLastDispatch("console", "user123", "console:user123")

	channel, userID, sessionID = m.LastDispatch()

	if channel != "console" {
		t.Errorf("expected channel 'console', got '%s'", channel)
	}

	if userID != "user123" {
		t.Errorf("expected userID 'user123', got '%s'", userID)
	}

	if sessionID != "console:user123" {
		t.Errorf("expected sessionID 'console:user123', got '%s'", sessionID)
	}
}

func TestManagerIntegration_SetDaemonInfo(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	info := &agent.DaemonInfo{
		ReloadConfig: func() error {
			return nil
		},
		Restart: func() error {
			return nil
		},
	}

	m.SetDaemonInfo(info)

	t.Log("DaemonInfo set successfully")
}

func TestManagerIntegration_ConcurrentSend(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
		Discord: config.DiscordConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	ctx := context.Background()
	done := make(chan bool, 20)

	channels := []string{"console", "telegram", "discord"}

	for i := 0; i < 20; i++ {
		go func(idx int) {
			ch := channels[idx%len(channels)]
			err := m.Send(ctx, ch, "user123", "test message")
			if err != nil && ch == "console" {
				t.Errorf("Send %d to %s failed: %v", idx, ch, err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	t.Log("Concurrent sends completed")
}

func TestManagerIntegration_MultipleInstances(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}

	m1 := NewManager(&mockAgentForManagerIntegration{}, cfg)
	m2 := NewManager(&mockAgentForManagerIntegration{}, cfg)

	if m1 == m2 {
		t.Error("different managers should be different instances")
	}

	channels1 := m1.Channels()
	channels2 := m2.Channels()

	if len(channels1) != len(channels2) {
		t.Errorf("managers with same config should have same number of channels")
	}

	m1.Register(NewConsole(&mockAgentForManagerIntegration{}, true, nil))

	channels1 = m1.Channels()
	channels2 = m2.Channels()

	if len(channels1) != len(channels2) {
		t.Error("modifying one manager should not affect the other (channel count should remain same after replace)")
	}

	if &channels1[0] == &channels2[0] {
		t.Error("channels in different managers should be different instances")
	}
}

func TestManagerIntegration_DisabledChannelsExcluded(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		Telegram: config.TelegramConfig{
			Enabled:  false,
			BotToken: "test_token",
		},
		Discord: config.DiscordConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	channels := m.Channels()

	if len(channels) != 2 {
		t.Errorf("expected 2 enabled channels, got %d", len(channels))
	}

	channelNames := make(map[string]bool)
	for _, ch := range channels {
		channelNames[ch.Name()] = true
	}

	if !channelNames["console"] {
		t.Error("console channel should be present")
	}

	if !channelNames["discord"] {
		t.Error("discord channel should be present")
	}

	if channelNames["telegram"] {
		t.Error("telegram channel should not be present (disabled)")
	}
}

func TestManagerIntegration_EmptyConfig(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: false},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	channels := m.Channels()

	if len(channels) != 0 {
		t.Errorf("expected 0 channels for empty config, got %d", len(channels))
	}

	ctx := context.Background()

	err := m.Start(ctx)
	if err != nil {
		t.Errorf("Start with empty config failed: %v", err)
	}

	err = m.Stop(ctx)
	if err != nil {
		t.Errorf("Stop with empty config failed: %v", err)
	}
}

func TestManagerIntegration_GetChannelByName(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test_token",
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	allChannels := m.Channels()

	channelMap := make(map[string]Channel)
	for _, ch := range allChannels {
		channelMap[ch.Name()] = ch
	}

	if _, ok := channelMap["console"]; !ok {
		t.Error("console channel not found")
	}

	if _, ok := channelMap["telegram"]; !ok {
		t.Error("telegram channel not found")
	}

	if _, ok := channelMap["discord"]; ok {
		t.Error("discord channel should not be present (not enabled)")
	}
}

func TestManagerIntegration_StreamModeChannels(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
		DingTalk: config.DingTalkConfig{
			Enabled:      true,
			ClientID:     "test_id",
			ClientSecret: "test_secret",
			UseStream:    true,
		},
		Feishu: config.FeishuConfig{
			Enabled:      true,
			AppID:        "test_app_id",
			AppSecret:    "test_secret",
			UseWebSocket: true,
		},
	}

	m := NewManager(&mockAgentForManagerIntegration{}, cfg)

	channels := m.Channels()

	if len(channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(channels))
	}

	channelNames := make(map[string]bool)
	for _, ch := range channels {
		channelNames[ch.Name()] = true
	}

	if !channelNames["console"] {
		t.Error("console channel should be present")
	}

	if !channelNames["dingtalk"] {
		t.Error("dingtalk channel should be present (stream mode)")
	}

	if !channelNames["feishu"] {
		t.Error("feishu channel should be present (websocket mode)")
	}
}
