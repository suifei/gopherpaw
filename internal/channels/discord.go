package channels

import (
	"context"
	"fmt"
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
			runCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
