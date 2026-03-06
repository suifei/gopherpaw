package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForIntegration struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "integration test response", nil
}

func (m *mockAgentForIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, chatID, text)
	}
	ch := make(chan string, 1)
	ch <- "integration stream"
	close(ch)
	return ch, nil
}

// TestChannelInterface verifies Channel interface compliance
func TestChannelInterface(t *testing.T) {
	agent := &mockAgentForIntegration{}

	channels := []struct {
		channel Channel
		name    string
	}{
		{NewConsole(agent, true, nil), "console"},
		{NewTelegram(agent, config.TelegramConfig{Enabled: true, BotToken: "token"}, nil), "telegram"},
		{NewDiscord(agent, config.DiscordConfig{Enabled: true, BotToken: "token"}, nil), "discord"},
	}

	for _, tc := range channels {
		ch := tc.channel
		if ch == nil {
			t.Errorf("Channel creation returned nil for %s", tc.name)
			continue
		}

		name := ch.Name()
		if name == "" {
			t.Errorf("Channel.Name() returned empty string")
		}

		enabled := ch.IsEnabled()
		if !enabled {
			t.Errorf("Channel %s should be enabled", name)
		}

		ctx := context.Background()
		err := ch.Send(ctx, "user123", "test", nil)
		if err != nil && name == "telegram" {
			t.Logf("Channel %s Send error (expected): %v", name, err)
		}

		err = ch.Stop(ctx)
		if err != nil {
			t.Logf("Channel %s Stop error (expected): %v", name, err)
		}
	}
}

// TestFileSenderInterface verifies FileSender interface compliance
func TestFileSenderInterface(t *testing.T) {
	agent := &mockAgentForIntegration{}

	channels := []struct {
		name     string
		channel  Channel
		isSender bool
	}{
		{
			name:     "console",
			channel:  NewConsole(agent, true, nil),
			isSender: true,
		},
		{
			name:     "telegram",
			channel:  NewTelegram(agent, config.TelegramConfig{Enabled: true, BotToken: "token"}, nil),
			isSender: true,
		},
		{
			name:     "discord",
			channel:  NewDiscord(agent, config.DiscordConfig{Enabled: true, BotToken: "token"}, nil),
			isSender: true,
		},
	}

	ctx := context.Background()
	for _, tc := range channels {
		if sender, ok := tc.channel.(FileSender); ok {
			err := sender.SendFile(ctx, "user123", "/tmp/file.txt", "text/plain", nil)
			if err != nil {
				t.Logf("FileSender %s SendFile error (expected for some): %v", tc.name, err)
			}
		} else if tc.isSender {
			t.Errorf("%s should implement FileSender", tc.name)
		}
	}
}

// TestManagerChannelBuilding tests Manager.buildChannels
func TestManagerChannelBuilding(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.ChannelsConfig
		expectedMin   int
		checkChannels []string
	}{
		{
			name: "console_only",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: true},
			},
			expectedMin:   1,
			checkChannels: []string{"console"},
		},
		{
			name: "console_and_telegram",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: true},
				Telegram: config.TelegramConfig{
					Enabled:  true,
					BotToken: "token",
				},
			},
			expectedMin:   2,
			checkChannels: []string{"console", "telegram"},
		},
		{
			name: "disabled_channels_excluded",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: true},
				Telegram: config.TelegramConfig{
					Enabled:  false,
					BotToken: "token",
				},
			},
			expectedMin:   1,
			checkChannels: []string{"console"},
		},
		{
			name: "no_channels",
			cfg: config.ChannelsConfig{
				Console: config.ConsoleConfig{Enabled: false},
			},
			expectedMin:   0,
			checkChannels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(&mockAgentForIntegration{}, tt.cfg)
			channels := m.Channels()

			if len(channels) < tt.expectedMin {
				t.Errorf("expected at least %d channels, got %d", tt.expectedMin, len(channels))
			}

			found := make(map[string]bool)
			for _, ch := range channels {
				found[ch.Name()] = true
			}

			for _, chName := range tt.checkChannels {
				if !found[chName] {
					t.Errorf("expected channel %s not found", chName)
				}
			}
		})
	}
}

// TestManagerSendNonExistentChannel tests sending to non-existent channel
func TestManagerSendNonExistentChannel(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForIntegration{}, cfg)

	ctx := context.Background()
	err := m.Send(ctx, "nonexistent", "user123", "test message")
	if err != nil {
		t.Errorf("Send to non-existent channel should not error, got: %v", err)
	}
}

// TestMultipleManagerInstances tests multiple independent manager instances
func TestMultipleManagerInstances(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}

	m1 := NewManager(&mockAgentForIntegration{}, cfg)
	m2 := NewManager(&mockAgentForIntegration{}, cfg)

	if m1 == m2 {
		t.Errorf("different managers should be different instances")
	}

	channels1 := m1.Channels()
	channels2 := m2.Channels()

	if len(channels1) != len(channels2) {
		t.Errorf("managers with same config should have same number of channels")
	}
}
