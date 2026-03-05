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

const feishuTokenURL = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

// FeishuChannel implements Channel for Feishu (Lark) Bot.
// Uses HTTP API for token and sending; receives via Event subscription (requires public URL).
type FeishuChannel struct {
	agent   agent.Agent
	cfg     config.FeishuConfig
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

// NewFeishu creates a Feishu channel.
func NewFeishu(ag agent.Agent, cfg config.FeishuConfig, onMsg func(context.Context, string, IncomingMessage) error) *FeishuChannel {
	return &FeishuChannel{
		agent:  ag,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		onMsg:  onMsg,
	}
}

// Name returns the channel identifier.
func (c *FeishuChannel) Name() string {
	return "feishu"
}

// IsEnabled returns whether the channel is enabled.
func (c *FeishuChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.AppID != "" && c.cfg.AppSecret != ""
}

// Start starts the Feishu channel. Receiving requires Event subscription webhook.
func (c *FeishuChannel) Start(ctx context.Context) error {
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
	logger.L().Info("feishu channel started (send-ready; receive via event webhook)")
	return nil
}

// Stop stops the Feishu channel.
func (c *FeishuChannel) Stop(ctx context.Context) error {
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

// Send sends a text message. Meta must contain "receive_id" and "receive_id_type" (open_id/chat_id).
func (c *FeishuChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	if !c.IsEnabled() {
		return nil
	}
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	receiveID := to
	receiveIDType := "open_id"
	if meta != nil {
		if id := meta["receive_id"]; id != "" {
			receiveID = id
		}
		if t := meta["receive_id_type"]; t != "" {
			receiveIDType = t
		}
	}
	if receiveID == "" {
		return fmt.Errorf("feishu send: receive_id required")
	}
	body := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    map[string]string{"text": c.cfg.BotPrefix + text},
	}
	b, _ := json.Marshal(body)
	url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveIDType
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("feishu send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu send status %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

func (c *FeishuChannel) getToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpireAt) {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"app_id":     c.cfg.AppID,
		"app_secret": c.cfg.AppSecret,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", feishuTokenURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data struct {
		Code              int    `json:"code"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.Code != 0 || data.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu token error code=%d", data.Code)
	}
	c.token = data.TenantAccessToken
	c.tokenExpireAt = time.Now().Add(time.Duration(data.Expire-300) * time.Second)
	return c.token, nil
}

// HandleWebhook processes an incoming Feishu event. Implements WebhookHandler.
func (c *FeishuChannel) HandleWebhook(ctx context.Context, body []byte) error {
	return c.HandleEvent(ctx, body)
}

// HandleEvent processes an incoming Feishu event. Call from your HTTP handler for Event 2.0 subscription.
func (c *FeishuChannel) HandleEvent(ctx context.Context, body []byte) error {
	var evt struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Event     struct {
			Message struct {
				MessageID   string `json:"message_id"`
				MessageType string `json:"message_type"`
				Content     string `json:"content"`
				ChatID      string `json:"chat_id"`
				ChatType    string `json:"chat_type"`
			} `json:"message"`
			Sender struct {
				SenderID   string `json:"sender_id"`
				SenderType string `json:"sender_type"`
			} `json:"sender"`
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &evt); err != nil {
		return err
	}
	if evt.Type == "url_verification" && evt.Challenge != "" {
		return nil
	}
	if evt.Event.Sender.SenderType == "app" {
		return nil
	}
	var content struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(evt.Event.Message.Content), &content)
	text := content.Text
	if text == "" {
		return nil
	}
	msg := IncomingMessage{
		ChatID:    evt.Event.Message.ChatID,
		UserID:    evt.Event.Sender.SenderID,
		Content:   text,
		Channel:   "feishu",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"chat_id":        evt.Event.Message.ChatID,
			"chat_type":      evt.Event.Message.ChatType,
			"receive_id":     evt.Event.Message.ChatID,
			"receive_id_type": "chat_id",
		},
	}
	if evt.Event.Message.ChatType == "p2p" {
		msg.Metadata["receive_id"] = evt.Event.Sender.SenderID
		msg.Metadata["receive_id_type"] = "open_id"
	}
	if c.onMsg != nil {
		return c.onMsg(ctx, "feishu", msg)
	}
	return nil
}
