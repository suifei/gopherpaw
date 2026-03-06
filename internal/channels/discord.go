package channels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// DiscordChannel implements Channel for Discord Bot API.
type DiscordChannel struct {
	agent   agent.Agent
	cfg     config.DiscordConfig
	session *discordgo.Session
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
	running bool
	onMsg   func(ctx context.Context, chName string, msg IncomingMessage) error
}

// NewDiscord creates a Discord channel.
func NewDiscord(ag agent.Agent, cfg config.DiscordConfig, onMsg func(context.Context, string, IncomingMessage) error) *DiscordChannel {
	return &DiscordChannel{
		agent:  ag,
		cfg:    cfg,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		onMsg:  onMsg,
	}
}

// Name returns the channel identifier.
func (c *DiscordChannel) Name() string {
	return "discord"
}

// IsEnabled returns whether the channel is enabled.
func (c *DiscordChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.BotToken != ""
}

// Start connects to Discord and begins listening.
func (c *DiscordChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	dg, err := discordgo.New("Bot " + c.cfg.BotToken)
	if err != nil {
		return fmt.Errorf("discord new session: %w", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent
	c.session = dg

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			return
		}
		meta := map[string]string{
			"user_id":    m.Author.ID,
			"channel_id": m.ChannelID,
			"message_id": m.ID,
		}
		if m.GuildID != "" {
			meta["guild_id"] = m.GuildID
		}
		chatID := m.ChannelID
		if m.GuildID == "" {
			chatID = "dm:" + m.Author.ID
		}
		msg := IncomingMessage{
			ChatID:    chatID,
			UserID:    m.Author.ID,
			UserName:  m.Author.Username,
			Content:   text,
			Channel:   "discord",
			Timestamp: time.Now().Unix(),
			Metadata:  meta,
		}
		if c.onMsg != nil {
			runCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()
			if err := c.onMsg(runCtx, "discord", msg); err != nil {
				logger.L().Warn("discord onMsg failed", zap.Error(err))
			}
		}
	})

	if err := dg.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}

	go func() {
		defer close(c.doneCh)
		<-c.stopCh
		if c.session != nil {
			c.session.Close()
		}
	}()
	logger.L().Info("discord channel started")
	return nil
}

// Stop disconnects from Discord.
func (c *DiscordChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	close(c.stopCh)
	select {
	case <-c.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// SendFile sends a file to the given Discord channel.
func (c *DiscordChannel) SendFile(ctx context.Context, to string, filePath string, mimeType string, meta map[string]string) error {
	if !c.IsEnabled() || c.session == nil {
		return nil
	}
	channelID := ""
	userID := ""
	if meta != nil {
		channelID = meta["channel_id"]
		userID = meta["user_id"]
	}
	if channelID == "" && to != "" {
		if strings.HasPrefix(to, "dm:") {
			userID = strings.TrimPrefix(to, "dm:")
		} else {
			channelID = to
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	fileName := filepath.Base(filePath)
	files := []*discordgo.File{{Name: fileName, Reader: f}}

	if channelID != "" {
		_, err := c.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Files: files})
		return err
	}
	if userID != "" {
		ch, err := c.session.UserChannelCreate(userID)
		if err != nil {
			return fmt.Errorf("discord create dm: %w", err)
		}
		_, err = c.session.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{Files: files})
		return err
	}
	return fmt.Errorf("discord send file: need channel_id or user_id in meta")
}

// Send sends a text message. Uses meta["channel_id"] or meta["user_id"] for routing.
func (c *DiscordChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	if !c.IsEnabled() || c.session == nil {
		return nil
	}
	channelID := ""
	userID := ""
	if meta != nil {
		channelID = meta["channel_id"]
		userID = meta["user_id"]
	}
	if channelID == "" && to != "" {
		if strings.HasPrefix(to, "dm:") {
			userID = strings.TrimPrefix(to, "dm:")
		} else {
			channelID = to
		}
	}
	if channelID != "" {
		chID, _ := strconv.ParseInt(channelID, 10, 64)
		if chID != 0 {
			_, err := c.session.ChannelMessageSend(channelID, text)
			return err
		}
	}
	if userID != "" {
		ch, err := c.session.UserChannelCreate(userID)
		if err != nil {
			return fmt.Errorf("discord create dm: %w", err)
		}
		_, err = c.session.ChannelMessageSend(ch.ID, text)
		return err
	}
	return fmt.Errorf("discord send: need channel_id or user_id in meta")
}

// resolveDiscordChannel resolves a channel ID from 'to' and 'meta'.
func (c *DiscordChannel) resolveDiscordChannel(to string, meta map[string]string) (string, error) {
	channelID := ""
	userID := ""
	if meta != nil {
		channelID = meta["channel_id"]
		userID = meta["user_id"]
	}
	if channelID == "" && to != "" {
		if strings.HasPrefix(to, "dm:") {
			userID = strings.TrimPrefix(to, "dm:")
		} else {
			channelID = to
		}
	}
	if channelID != "" {
		return channelID, nil
	}
	if userID != "" {
		ch, err := c.session.UserChannelCreate(userID)
		if err != nil {
			return "", fmt.Errorf("discord create dm: %w", err)
		}
		return ch.ID, nil
	}
	return "", fmt.Errorf("need channel_id or user_id in meta")
}

// EditMessage edits a previously sent message.
func (c *DiscordChannel) EditMessage(ctx context.Context, to string, messageID string, newText string, meta map[string]string) error {
	if !c.IsEnabled() || c.session == nil {
		return nil
	}
	channelID, err := c.resolveDiscordChannel(to, meta)
	if err != nil {
		return err
	}
	_, err = c.session.ChannelMessageEdit(channelID, messageID, newText)
	return err
}

// DeleteMessage deletes a message from the channel.
func (c *DiscordChannel) DeleteMessage(ctx context.Context, to string, messageID string, meta map[string]string) error {
	if !c.IsEnabled() || c.session == nil {
		return nil
	}
	channelID, err := c.resolveDiscordChannel(to, meta)
	if err != nil {
		return err
	}
	return c.session.ChannelMessageDelete(channelID, messageID)
}

// React adds a reaction emoji to a message.
func (c *DiscordChannel) React(ctx context.Context, to string, messageID string, emoji string, meta map[string]string) error {
	if !c.IsEnabled() || c.session == nil {
		return nil
	}
	channelID, err := c.resolveDiscordChannel(to, meta)
	if err != nil {
		return err
	}
	return c.session.MessageReactionAdd(channelID, messageID, emoji)
}

// SendTyping sends a typing indicator to the channel.
func (c *DiscordChannel) SendTyping(ctx context.Context, to string, meta map[string]string) error {
	if !c.IsEnabled() || c.session == nil {
		return nil
	}
	channelID, err := c.resolveDiscordChannel(to, meta)
	if err != nil {
		return err
	}
	return c.session.ChannelTyping(channelID)
}

// Ensure DiscordChannel implements optional interfaces.
var (
	_ Channel         = (*DiscordChannel)(nil)
	_ FileSender      = (*DiscordChannel)(nil)
	_ MessageEditor   = (*DiscordChannel)(nil)
	_ MessageDeleter  = (*DiscordChannel)(nil)
	_ Reactor         = (*DiscordChannel)(nil)
	_ TypingIndicator = (*DiscordChannel)(nil)
)
