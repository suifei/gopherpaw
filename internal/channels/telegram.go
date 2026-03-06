package channels

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

const telegramChunkSize = 4000

// TelegramChannel implements Channel for Telegram Bot API.
type TelegramChannel struct {
	agent   agent.Agent
	cfg     config.TelegramConfig
	bot     *telebot.Bot
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
	running bool
	onMsg   func(ctx context.Context, chName string, msg IncomingMessage) error
}

// NewTelegram creates a Telegram channel.
func NewTelegram(ag agent.Agent, cfg config.TelegramConfig, onMsg func(context.Context, string, IncomingMessage) error) *TelegramChannel {
	return &TelegramChannel{
		agent:  ag,
		cfg:    cfg,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		onMsg:  onMsg,
	}
}

// Name returns the channel identifier.
func (c *TelegramChannel) Name() string {
	return "telegram"
}

// IsEnabled returns whether the channel is enabled.
func (c *TelegramChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.BotToken != ""
}

// Start begins polling for Telegram messages.
func (c *TelegramChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	pref := telebot.Settings{
		Token:  c.cfg.BotToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}
	if c.cfg.HTTPProxy != "" {
		proxyURL, err := url.Parse(c.cfg.HTTPProxy)
		if err == nil {
			pref.Client = &http.Client{
				Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
			}
		}
	}
	bot, err := telebot.NewBot(pref)
	if err != nil {
		return fmt.Errorf("telegram new bot: %w", err)
	}
	c.bot = bot

	bot.Handle(telebot.OnText, func(m telebot.Context) error {
		text := strings.TrimSpace(m.Text())
		if text == "" {
			return nil
		}
		chat := m.Chat()
		sender := m.Sender()
		chatID := ""
		if chat != nil {
			chatID = strconv.FormatInt(chat.ID, 10)
		}
		userID := ""
		userName := ""
		if sender != nil {
			userID = strconv.FormatInt(sender.ID, 10)
			userName = sender.Username
		}
		msg := IncomingMessage{
			ChatID:    chatID,
			UserID:    userID,
			UserName:  userName,
			Content:   text,
			Channel:   "telegram",
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"chat_id": chatID,
			},
		}
		if c.onMsg != nil {
			runCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()
			if err := c.onMsg(runCtx, "telegram", msg); err != nil {
				logger.L().Warn("telegram onMsg failed", zap.Error(err))
			}
		}
		return nil
	})

	go func() {
		defer close(c.doneCh)
		bot.Start()
	}()
	logger.L().Info("telegram channel started")
	return nil
}

// Stop stops the Telegram bot.
func (c *TelegramChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	if c.bot != nil {
		c.bot.Stop()
	}
	select {
	case <-c.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Send sends a text message to the given chat.
func (c *TelegramChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	if !c.IsEnabled() || c.bot == nil {
		return nil
	}
	chatIDStr := to
	if meta != nil {
		if id := meta["chat_id"]; id != "" {
			chatIDStr = id
		}
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}
	recipient := &telebot.Chat{ID: chatID}
	chunks := chunkText(text, telegramChunkSize)
	for _, chunk := range chunks {
		if _, err := c.bot.Send(recipient, chunk); err != nil {
			return fmt.Errorf("telegram send: %w", err)
		}
	}
	return nil
}

// SendFile sends a file to the given Telegram chat.
func (c *TelegramChannel) SendFile(ctx context.Context, to string, filePath string, mimeType string, meta map[string]string) error {
	if !c.IsEnabled() || c.bot == nil {
		return nil
	}
	chatIDStr := to
	if meta != nil {
		if id := meta["chat_id"]; id != "" {
			chatIDStr = id
		}
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}
	recipient := &telebot.Chat{ID: chatID}

	doc := &telebot.Document{
		File: telebot.FromDisk(filePath),
	}
	if strings.HasPrefix(mimeType, "image/") {
		photo := &telebot.Photo{
			File: telebot.FromDisk(filePath),
		}
		_, err = c.bot.Send(recipient, photo)
	} else {
		_, err = c.bot.Send(recipient, doc)
	}
	return err
}

func chunkText(s string, maxLen int) []string {
	if s == "" {
		return nil
	}
	if len(s) <= maxLen {
		return []string{s}
	}
	var out []string
	for len(s) > maxLen {
		chunk := s[:maxLen]
		last := strings.LastIndex(chunk, "\n")
		if last > maxLen/2 {
			chunk = chunk[:last+1]
		} else {
			last = strings.LastIndex(chunk, " ")
			if last > maxLen/2 {
				chunk = chunk[:last+1]
			}
		}
		out = append(out, chunk)
		s = strings.TrimLeft(s[len(chunk):], "\n ")
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

// SendMarkdown sends a Markdown-formatted message to the given chat.
func (c *TelegramChannel) SendMarkdown(ctx context.Context, to string, markdown string, meta map[string]string) error {
	if !c.IsEnabled() || c.bot == nil {
		return nil
	}
	chatIDStr := to
	if meta != nil {
		if id := meta["chat_id"]; id != "" {
			chatIDStr = id
		}
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}
	recipient := &telebot.Chat{ID: chatID}
	chunks := chunkText(markdown, telegramChunkSize)
	for _, chunk := range chunks {
		if _, err := c.bot.Send(recipient, chunk, telebot.ModeMarkdownV2); err != nil {
			return fmt.Errorf("telegram send markdown: %w", err)
		}
	}
	return nil
}

// EditMessage edits a previously sent message.
func (c *TelegramChannel) EditMessage(ctx context.Context, to string, messageID string, newText string, meta map[string]string) error {
	if !c.IsEnabled() || c.bot == nil {
		return nil
	}
	chatIDStr := to
	if meta != nil {
		if id := meta["chat_id"]; id != "" {
			chatIDStr = id
		}
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}
	msgID, err := strconv.Atoi(messageID)
	if err != nil {
		return fmt.Errorf("invalid message_id: %w", err)
	}
	msg := &telebot.Message{
		ID:   msgID,
		Chat: &telebot.Chat{ID: chatID},
	}
	_, err = c.bot.Edit(msg, newText)
	return err
}

// DeleteMessage deletes a message from the chat.
func (c *TelegramChannel) DeleteMessage(ctx context.Context, to string, messageID string, meta map[string]string) error {
	if !c.IsEnabled() || c.bot == nil {
		return nil
	}
	chatIDStr := to
	if meta != nil {
		if id := meta["chat_id"]; id != "" {
			chatIDStr = id
		}
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}
	msgID, err := strconv.Atoi(messageID)
	if err != nil {
		return fmt.Errorf("invalid message_id: %w", err)
	}
	msg := &telebot.Message{
		ID:   msgID,
		Chat: &telebot.Chat{ID: chatID},
	}
	return c.bot.Delete(msg)
}

// SendTyping sends a typing indicator to the chat.
func (c *TelegramChannel) SendTyping(ctx context.Context, to string, meta map[string]string) error {
	if !c.IsEnabled() || c.bot == nil {
		return nil
	}
	chatIDStr := to
	if meta != nil {
		if id := meta["chat_id"]; id != "" {
			chatIDStr = id
		}
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}
	recipient := &telebot.Chat{ID: chatID}
	return c.bot.Notify(recipient, telebot.Typing)
}

// Ensure TelegramChannel implements optional interfaces.
var (
	_ Channel         = (*TelegramChannel)(nil)
	_ FileSender      = (*TelegramChannel)(nil)
	_ MarkdownSender  = (*TelegramChannel)(nil)
	_ MessageEditor   = (*TelegramChannel)(nil)
	_ MessageDeleter  = (*TelegramChannel)(nil)
	_ TypingIndicator = (*TelegramChannel)(nil)
)
