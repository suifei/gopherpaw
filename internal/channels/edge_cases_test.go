package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

// 这个文件专注于测试通道的参数验证和边界情况

// TestConsoleChannelEdgeCases 测试 Console 通道的边界情况
func TestConsoleChannelEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		text    string
		meta    map[string]string
	}{
		{
			name:    "empty message",
			enabled: true,
			text:    "",
			meta:    nil,
		},
		{
			name:    "very long message",
			enabled: true,
			text:    string(make([]byte, 10000)),
			meta:    nil,
		},
		{
			name:    "unicode text",
			enabled: true,
			text:    "你好世界 🌍 مرحبا",
			meta:    nil,
		},
		{
			name:    "with empty meta",
			enabled: true,
			text:    "test",
			meta:    map[string]string{},
		},
		{
			name:    "disabled channel",
			enabled: false,
			text:    "test",
			meta:    nil,
		},
	}

	mockAg := newMockAgent()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConsole(mockAg, tt.enabled, nil)

			ctx := context.Background()
			// 验证不会 panic
			_ = c.Send(ctx, "user_id", tt.text, tt.meta)
		})
	}
}

// TestTelegramChannelEdgeCases 测试 Telegram 通道的边界情况
func TestTelegramChannelEdgeCases(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	tests := []struct {
		name     string
		chatID   string
		text     string
		shouldOK bool
	}{
		{
			name:     "valid chat ID",
			chatID:   "12345",
			text:     "hello",
			shouldOK: true,
		},
		{
			name:     "empty chat ID",
			chatID:   "",
			text:     "hello",
			shouldOK: false,
		},
		{
			name:     "empty message",
			chatID:   "12345",
			text:     "",
			shouldOK: true,
		},
		{
			name:     "very large chat ID",
			chatID:   "9223372036854775807",
			text:     "hello",
			shouldOK: true,
		},
		{
			name:     "chat ID with spaces",
			chatID:   "123 45",
			text:     "hello",
			shouldOK: false,
		},
	}

	mockAg := newMockAgent()
	c := NewTelegram(mockAg, cfg, nil)

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 因为 bot 为 nil，所以所有调用都应该返回 nil
			err := c.Send(ctx, tt.chatID, tt.text, nil)
			if err != nil {
				t.Logf("Send returned error (expected when bot is nil): %v", err)
			}
		})
	}
}

// TestDiscordChannelEdgeCases 测试 Discord 通道的边界情况
func TestDiscordChannelEdgeCases(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	tests := []struct {
		name      string
		channelID string
		text      string
	}{
		{
			name:      "valid channel ID",
			channelID: "123456789",
			text:      "hello",
		},
		{
			name:      "empty channel ID",
			channelID: "",
			text:      "hello",
		},
		{
			name:      "empty message",
			channelID: "123456789",
			text:      "",
		},
		{
			name:      "unicode message",
			channelID: "123456789",
			text:      "你好 🎉",
		},
	}

	mockAg := newMockAgent()
	c := NewDiscord(mockAg, cfg, nil)

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.channelID, tt.text, nil)
			if err != nil {
				t.Logf("Send returned error: %v", err)
			}
		})
	}
}

// TestDingTalkChannelEdgeCases 测试钉钉通道的边界情况
func TestDingTalkChannelEdgeCases(t *testing.T) {
	cfg := config.DingTalkConfig{
		Enabled: true,
	}

	tests := []struct {
		name   string
		userID string
		text   string
	}{
		{
			name:   "valid user ID",
			userID: "user123",
			text:   "hello",
		},
		{
			name:   "empty user ID",
			userID: "",
			text:   "hello",
		},
		{
			name:   "empty message",
			userID: "user123",
			text:   "",
		},
		{
			name:   "numeric user ID",
			userID: "123456789",
			text:   "hello",
		},
	}

	mockAg := newMockAgent()
	c := NewDingTalk(mockAg, cfg, nil)

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.userID, tt.text, nil)
			if err != nil {
				t.Logf("Send returned error: %v", err)
			}
		})
	}
}

// TestFeishuChannelEdgeCases 测试飞书通道的边界情况
func TestFeishuChannelEdgeCases(t *testing.T) {
	cfg := config.FeishuConfig{
		Enabled: true,
	}

	tests := []struct {
		name   string
		userID string
		text   string
	}{
		{
			name:   "valid user ID",
			userID: "user123",
			text:   "hello",
		},
		{
			name:   "empty user ID",
			userID: "",
			text:   "hello",
		},
		{
			name:   "very long message",
			userID: "user123",
			text:   string(make([]byte, 5000)),
		},
		{
			name:   "special characters",
			userID: "user@123",
			text:   "test!@#$%^&*()",
		},
	}

	mockAg := newMockAgent()
	c := NewFeishu(mockAg, cfg, nil)

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.userID, tt.text, nil)
			if err != nil {
				t.Logf("Send returned error: %v", err)
			}
		})
	}
}

// TestQQChannelEdgeCases 测试 QQ 通道的边界情况
func TestQQChannelEdgeCases(t *testing.T) {
	cfg := config.QQConfig{
		Enabled:      true,
		AppID:        "app123",
		ClientSecret: "secret123",
	}

	tests := []struct {
		name    string
		guildID string
		text    string
	}{
		{
			name:    "valid guild ID",
			guildID: "guild123",
			text:    "hello",
		},
		{
			name:    "empty guild ID",
			guildID: "",
			text:    "hello",
		},
		{
			name:    "empty message",
			guildID: "guild123",
			text:    "",
		},
		{
			name:    "numeric guild ID",
			guildID: "123456789",
			text:    "hello",
		},
	}

	mockAg := newMockAgent()
	c := NewQQ(mockAg, cfg, nil)

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.guildID, tt.text, nil)
			if err != nil {
				t.Logf("Send returned error: %v", err)
			}
		})
	}
}

// TestChannelNameMethods 测试所有通道的 Name() 和 IsEnabled() 方法
func TestChannelNameMethods(t *testing.T) {
	mockAg := newMockAgent()

	tests := []struct {
		name         string
		createFunc   func() Channel
		expectedName string
	}{
		{
			name: "Console",
			createFunc: func() Channel {
				return NewConsole(mockAg, true, nil)
			},
			expectedName: "console",
		},
		{
			name: "Telegram",
			createFunc: func() Channel {
				return NewTelegram(mockAg, config.TelegramConfig{Enabled: true, BotToken: "token"}, nil)
			},
			expectedName: "telegram",
		},
		{
			name: "Discord",
			createFunc: func() Channel {
				return NewDiscord(mockAg, config.DiscordConfig{Enabled: true, BotToken: "token"}, nil)
			},
			expectedName: "discord",
		},
		{
			name: "DingTalk",
			createFunc: func() Channel {
				return NewDingTalk(mockAg, config.DingTalkConfig{Enabled: true}, nil)
			},
			expectedName: "dingtalk",
		},
		{
			name: "Feishu",
			createFunc: func() Channel {
				return NewFeishu(mockAg, config.FeishuConfig{Enabled: true}, nil)
			},
			expectedName: "feishu",
		},
		{
			name: "QQ",
			createFunc: func() Channel {
				return NewQQ(mockAg, config.QQConfig{Enabled: true}, nil)
			},
			expectedName: "qq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := tt.createFunc()

			// 验证 Name() 方法
			if ch.Name() != tt.expectedName {
				t.Errorf("Name() = %q, want %q", ch.Name(), tt.expectedName)
			}

			// 验证 IsEnabled() 返回 bool
			_ = ch.IsEnabled()

			// 验证 Send 方法存在且不 panic
			ctx := context.Background()
			_ = ch.Send(ctx, "test_id", "test message", nil)
		})
	}
}

// TestChannelSendFileEdgeCases 测试 SendFile 方法的边界情况
func TestChannelSendFileEdgeCases(t *testing.T) {
	mockAg := newMockAgent()

	tests := []struct {
		name     string
		filePath string
		mimeType string
	}{
		{
			name:     "valid file path",
			filePath: "/tmp/file.txt",
			mimeType: "text/plain",
		},
		{
			name:     "empty file path",
			filePath: "",
			mimeType: "text/plain",
		},
		{
			name:     "relative path",
			filePath: "file.txt",
			mimeType: "text/plain",
		},
		{
			name:     "image mime type",
			filePath: "/tmp/image.jpg",
			mimeType: "image/jpeg",
		},
		{
			name:     "empty mime type",
			filePath: "/tmp/file",
			mimeType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewTelegram(mockAg, config.TelegramConfig{Enabled: true, BotToken: "token"}, nil)

			ctx := context.Background()
			// 验证不会 panic
			_ = c.SendFile(ctx, "user_id", tt.filePath, tt.mimeType, nil)
		})
	}
}

// TestChannelContextCancellation 测试通道对 context 取消的处理
func TestChannelContextCancellation(t *testing.T) {
	mockAg := newMockAgent()
	c := NewConsole(mockAg, true, nil)

	// 创建一个已取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// 尝试 Send - 应该不会 panic
	err := c.Send(ctx, "user_id", "test", nil)
	if err != nil {
		t.Logf("Send with canceled context returned: %v", err)
	}

	// 尝试 Stop - 应该不会 panic
	err = c.Stop(ctx)
	if err != nil {
		t.Logf("Stop with canceled context returned: %v", err)
	}
}

// TestChannelWithNilMeta 测试通道对 nil meta 的处理
func TestChannelWithNilMeta(t *testing.T) {
	mockAg := newMockAgent()

	tests := []struct {
		name string
		ch   Channel
	}{
		{
			name: "Console",
			ch:   NewConsole(mockAg, true, nil),
		},
		{
			name: "Telegram",
			ch:   NewTelegram(mockAg, config.TelegramConfig{Enabled: true, BotToken: "token"}, nil),
		},
		{
			name: "Discord",
			ch:   NewDiscord(mockAg, config.DiscordConfig{Enabled: true, BotToken: "token"}, nil),
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// nil meta 应该不会导致 panic
			_ = tt.ch.Send(ctx, "user_id", "test message", nil)
		})
	}
}

// TestChannelWithEmptyMeta 测试通道对空 meta 的处理
func TestChannelWithEmptyMeta(t *testing.T) {
	mockAg := newMockAgent()

	tests := []struct {
		name string
		ch   Channel
	}{
		{
			name: "Console",
			ch:   NewConsole(mockAg, true, nil),
		},
		{
			name: "Telegram",
			ch:   NewTelegram(mockAg, config.TelegramConfig{Enabled: true, BotToken: "token"}, nil),
		},
	}

	ctx := context.Background()
	emptyMeta := map[string]string{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 空 meta 应该不会导致 panic
			_ = tt.ch.Send(ctx, "user_id", "test message", emptyMeta)
		})
	}
}

// TestChannelDisabledOperations 测试禁用通道的操作
func TestChannelDisabledOperations(t *testing.T) {
	mockAg := newMockAgent()

	tests := []struct {
		name    string
		ch      Channel
		enabled bool
	}{
		{
			name:    "Console disabled",
			ch:      NewConsole(mockAg, false, nil),
			enabled: false,
		},
		{
			name:    "Telegram disabled",
			ch:      NewTelegram(mockAg, config.TelegramConfig{Enabled: false, BotToken: "token"}, nil),
			enabled: false,
		},
		{
			name:    "Discord disabled",
			ch:      NewDiscord(mockAg, config.DiscordConfig{Enabled: false, BotToken: "token"}, nil),
			enabled: false,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证 IsEnabled() 返回正确的值
			if tt.ch.IsEnabled() != tt.enabled {
				t.Errorf("IsEnabled() = %v, want %v", tt.ch.IsEnabled(), tt.enabled)
			}

			// 禁用通道的 Send 应该返回 nil
			err := tt.ch.Send(ctx, "user_id", "test", nil)
			if err != nil {
				t.Errorf("Send on disabled channel should return nil, got %v", err)
			}
		})
	}
}
