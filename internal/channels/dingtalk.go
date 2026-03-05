package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
)

const dingtalkTokenURL = "https://api.dingtalk.com/v1.0/oauth2/accessToken"

// DingTalkChannel implements Channel for DingTalk Stream mode.
// Uses HTTP API for token and sending; receives via HTTP callback (requires public URL).
type DingTalkChannel struct {
	agent   agent.Agent
	cfg     config.DingTalkConfig
	client  *http.Client
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
	running bool
	onMsg   func(ctx context.Context, chName string, msg IncomingMessage) error

	tokenMu       sync.Mutex
	token         string
	tokenExpireAt time.Time
}

// NewDingTalk creates a DingTalk channel.
func NewDingTalk(ag agent.Agent, cfg config.DingTalkConfig, onMsg func(context.Context, string, IncomingMessage) error) *DingTalkChannel {
	return &DingTalkChannel{
		agent:  ag,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		onMsg:  onMsg,
	}
}

// Name returns the channel identifier.
func (c *DingTalkChannel) Name() string {
	return "dingtalk"
}

// IsEnabled returns whether the channel is enabled.
func (c *DingTalkChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.ClientID != "" && c.cfg.ClientSecret != ""
}

// Start starts the DingTalk channel. Receiving requires an HTTP server to be set up separately
// (e.g. via webhook URL configured in DingTalk console). This implementation provides
// the send capability; receive is handled when the app receives POST callbacks.
func (c *DingTalkChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	go func() {
		defer close(c.doneCh)
		<-c.stopCh
	}()
	logger.L().Info("dingtalk channel started (send-only; receive via webhook)")
	return nil
}

// Stop stops the DingTalk channel.
func (c *DingTalkChannel) Stop(ctx context.Context) error {
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

// Send sends a text message via session_webhook. The meta must contain "session_webhook" with
// the webhook URL from an incoming message, or "webhook_url" for proactive send.
func (c *DingTalkChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	if !c.IsEnabled() {
		return nil
	}
	webhook := ""
	if meta != nil {
		webhook = meta["session_webhook"]
		if webhook == "" {
			webhook = meta["webhook_url"]
		}
	}
	if webhook == "" {
		return fmt.Errorf("dingtalk send: session_webhook required in meta (user must have chatted first)")
	}
	body := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": c.cfg.BotPrefix + text},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", webhook, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk send status %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

// HandleWebhook processes an incoming DingTalk webhook POST. Implements WebhookHandler.
func (c *DingTalkChannel) HandleWebhook(ctx context.Context, body []byte) error {
	var payload struct {
		ConversationID string `json:"conversationId"`
		SenderID       string `json:"senderId"`
		SenderNick     string `json:"senderNick"`
		Text           struct {
			Content string `json:"content"`
		} `json:"text"`
		SessionWebhook string `json:"sessionWebhook"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	content := payload.Text.Content
	if content == "" {
		return nil
	}
	msg := IncomingMessage{
		ChatID:    payload.ConversationID,
		UserID:    payload.SenderID,
		UserName:  payload.SenderNick,
		Content:   content,
		Channel:   "dingtalk",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"session_webhook": payload.SessionWebhook,
		},
	}
	if c.onMsg != nil {
		return c.onMsg(ctx, "dingtalk", msg)
	}
	return nil
}
