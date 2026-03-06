package channels

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForDiscordIntegration struct {
	runFunc func(ctx context.Context, chatID, text string) (string, error)
}

func (m *mockAgentForDiscordIntegration) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "response", nil
}

func (m *mockAgentForDiscordIntegration) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

func TestDiscordIntegration_Send_InvalidChatID(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()
	err := c.Send(ctx, "123", "test message", map[string]string{"channel_id": "abc"})
	if err != nil {
		t.Logf("Send with nil session returned error: %v", err)
	}
}

func TestDiscordIntegration_Send_MetaRouting(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name     string
		to       string
		meta     map[string]string
		expected string
	}{
		{
			name:     "direct_channel_id",
			to:       "987654321",
			meta:     map[string]string{"channel_id": "987654321"},
			expected: "987654321",
		},
		{
			name:     "dm_user",
			to:       "dm:123456789",
			meta:     map[string]string{"user_id": "123456789"},
			expected: "dm:123456789",
		},
		{
			name:     "guild_channel",
			to:       "987654321",
			meta:     map[string]string{"channel_id": "987654321", "guild_id": "123456"},
			expected: "987654321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Send(ctx, tt.to, "test", tt.meta)
			if err == nil && c.session != nil {
				t.Log("Send attempted (session not nil)")
			}
		})
	}
}

func TestDiscordIntegration_EditMessage_MetaRouting(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name      string
		to        string
		messageID string
		meta      map[string]string
	}{
		{
			name:      "channel_edit",
			to:        "987654321",
			messageID: "1234567890",
			meta:      map[string]string{"channel_id": "987654321"},
		},
		{
			name:      "dm_edit",
			to:        "dm:123456789",
			messageID: "1234567890",
			meta:      map[string]string{"user_id": "123456789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.EditMessage(ctx, tt.to, tt.messageID, "edited text", tt.meta)
			if err == nil && c.session != nil {
				t.Log("EditMessage attempted (session not nil)")
			}
		})
	}
}

func TestDiscordIntegration_DeleteMessage_MetaRouting(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name      string
		to        string
		messageID string
		meta      map[string]string
	}{
		{
			name:      "channel_delete",
			to:        "987654321",
			messageID: "1234567890",
			meta:      map[string]string{"channel_id": "987654321"},
		},
		{
			name:      "dm_delete",
			to:        "dm:123456789",
			messageID: "1234567890",
			meta:      map[string]string{"user_id": "123456789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.DeleteMessage(ctx, tt.to, tt.messageID, tt.meta)
			if err == nil && c.session != nil {
				t.Log("DeleteMessage attempted (session not nil)")
			}
		})
	}
}

func TestDiscordIntegration_React(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	emojis := []string{"👍", "🎉", "❤️", "🔥"}

	for _, emoji := range emojis {
		t.Run("emoji_"+emoji, func(t *testing.T) {
			err := c.React(ctx, "987654321", "1234567890", emoji, map[string]string{"channel_id": "987654321"})
			if err == nil && c.session != nil {
				t.Log("React attempted (session not nil)")
			}
		})
	}
}

func TestDiscordIntegration_SendTyping(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name string
		to   string
		meta map[string]string
	}{
		{
			name: "channel_typing",
			to:   "987654321",
			meta: map[string]string{"channel_id": "987654321"},
		},
		{
			name: "dm_typing",
			to:   "dm:123456789",
			meta: map[string]string{"user_id": "123456789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.SendTyping(ctx, tt.to, tt.meta)
			if err == nil && c.session != nil {
				t.Log("SendTyping attempted (session not nil)")
			}
		})
	}
}

func TestDiscordIntegration_SendFile(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	tests := []struct {
		name     string
		to       string
		filePath string
		mimeType string
		meta     map[string]string
	}{
		{
			name:     "send_image",
			to:       "987654321",
			filePath: "/tmp/test.jpg",
			mimeType: "image/jpeg",
			meta:     map[string]string{"channel_id": "987654321"},
		},
		{
			name:     "send_document",
			to:       "987654321",
			filePath: "/tmp/test.pdf",
			mimeType: "application/pdf",
			meta:     map[string]string{"channel_id": "987654321"},
		},
		{
			name:     "send_file_dm",
			to:       "dm:123456789",
			filePath: "/tmp/test.txt",
			mimeType: "text/plain",
			meta:     map[string]string{"user_id": "123456789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.SendFile(ctx, tt.to, tt.filePath, tt.mimeType, tt.meta)
			if err == nil && c.session != nil {
				t.Log("SendFile attempted (session not nil)")
			}
		})
	}
}

func TestDiscordIntegration_StartStop(t *testing.T) {
	t.Skip("Start() requires real WebSocket connection, skipping integration test")

	cfg := config.DiscordConfig{
		Enabled:  false,
		BotToken: "",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

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

func TestDiscordIntegration_ConcurrentOperations(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "test_token",
	}

	c := NewDiscord(&mockAgentForDiscordIntegration{}, cfg, nil)

	ctx := context.Background()

	done := make(chan bool, 5)

	operations := []func(){
		func() {
			c.Send(ctx, "987654321", "test", map[string]string{"channel_id": "987654321"})
			done <- true
		},
		func() {
			c.EditMessage(ctx, "987654321", "1234567890", "edited", map[string]string{"channel_id": "987654321"})
			done <- true
		},
		func() {
			c.DeleteMessage(ctx, "987654321", "1234567890", map[string]string{"channel_id": "987654321"})
			done <- true
		},
		func() {
			c.React(ctx, "987654321", "1234567890", "👍", map[string]string{"channel_id": "987654321"})
			done <- true
		},
		func() {
			c.SendTyping(ctx, "987654321", map[string]string{"channel_id": "987654321"})
			done <- true
		},
	}

	for _, op := range operations {
		go op()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	t.Log("Concurrent operations completed")
}
