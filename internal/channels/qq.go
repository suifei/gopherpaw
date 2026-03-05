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

const qqTokenURL = "https://bots.qq.com/app/getAppAccessToken"
const qqAPIBase = "https://api.sgroup.qq.com"

// QQChannel implements Channel for QQ Bot API.
// Uses HTTP API for token and sending; receives via WebSocket (requires separate setup).
type QQChannel struct {
	agent   agent.Agent
	cfg     config.QQConfig
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

// NewQQ creates a QQ channel.
func NewQQ(ag agent.Agent, cfg config.QQConfig, onMsg func(context.Context, string, IncomingMessage) error) *QQChannel {
	return &QQChannel{
		agent:  ag,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		onMsg:  onMsg,
	}
}

// Name returns the channel identifier.
func (c *QQChannel) Name() string {
	return "qq"
}

// IsEnabled returns whether the channel is enabled.
func (c *QQChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.AppID != "" && c.cfg.ClientSecret != ""
}

// Start starts the QQ channel. Receiving requires WebSocket connection to QQ gateway.
func (c *QQChannel) Start(ctx context.Context) error {
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
	logger.L().Info("qq channel started (send-ready; receive via websocket)")
	return nil
}

// Stop stops the QQ channel.
func (c *QQChannel) Stop(ctx context.Context) error {
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

// Send sends a text message. Meta must contain message_type (c2c/group/guild), and
// sender_id (for c2c), group_openid (for group), or channel_id (for guild).
func (c *QQChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	if !c.IsEnabled() {
		return nil
	}
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	msgType := "c2c"
	senderID := to
	if meta != nil {
		if t := meta["message_type"]; t != "" {
			msgType = t
		}
		if id := meta["sender_id"]; id != "" {
			senderID = id
		}
	}
	body := map[string]interface{}{
		"content":  c.cfg.BotPrefix + text,
		"msg_type": 0,
	}
	if meta != nil && meta["message_id"] != "" {
		body["msg_id"] = meta["message_id"]
	}
	var path string
	switch msgType {
	case "c2c":
		path = fmt.Sprintf("/v2/users/%s/messages", senderID)
	case "group":
		gid := meta["group_openid"]
		if gid == "" {
			gid = to
		}
		path = fmt.Sprintf("/v2/groups/%s/messages", gid)
	case "guild":
		cid := meta["channel_id"]
		if cid == "" {
			cid = to
		}
		path = fmt.Sprintf("/channels/%s/messages", cid)
	default:
		path = fmt.Sprintf("/v2/users/%s/messages", senderID)
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", qqAPIBase+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "QQBot "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("qq send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qq send status %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

func (c *QQChannel) getToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpireAt) {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"appId":        c.cfg.AppID,
		"clientSecret": c.cfg.ClientSecret,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", qqTokenURL, bytes.NewReader(body))
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
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", fmt.Errorf("qq token empty")
	}
	c.token = data.AccessToken
	c.tokenExpireAt = time.Now().Add(time.Duration(data.ExpiresIn-300) * time.Second)
	return c.token, nil
}

// HandleWebhook processes an incoming QQ event from JSON body. Implements WebhookHandler.
// Body format: {"msg_type":"c2c"|"group"|"guild", "author":{...}, "content":"...", ...}
func (c *QQChannel) HandleWebhook(ctx context.Context, body []byte) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	msgType := "c2c"
	if t, ok := payload["msg_type"].(string); ok && t != "" {
		msgType = t
	}
	return c.HandleWebhookPayload(ctx, msgType, payload)
}

// HandleWebhookPayload processes an incoming QQ event. Call from your WebSocket message handler.
func (c *QQChannel) HandleWebhookPayload(ctx context.Context, msgType string, payload map[string]interface{}) error {
	author, _ := payload["author"].(map[string]interface{})
	content, _ := payload["content"].(string)
	content = trimSpace(content)
	if content == "" {
		return nil
	}
	senderID := ""
	if author != nil {
		if id, ok := author["user_openid"].(string); ok {
			senderID = id
		}
		if senderID == "" {
			if id, ok := author["id"].(string); ok {
				senderID = id
			}
		}
	}
	if senderID == "" {
		return nil
	}
	chatID := senderID
	meta := map[string]string{"sender_id": senderID, "message_type": msgType}
	if msgType == "group" {
		if gid, ok := payload["group_openid"].(string); ok {
			chatID = "group:" + gid
			meta["group_openid"] = gid
		}
	} else if msgType == "guild" {
		if cid, ok := payload["channel_id"].(string); ok {
			chatID = "channel:" + cid
			meta["channel_id"] = cid
		}
	}
	msg := IncomingMessage{
		ChatID:    chatID,
		UserID:    senderID,
		Content:   content,
		Channel:   "qq",
		Timestamp: time.Now().Unix(),
		Metadata:  meta,
	}
	if c.onMsg != nil {
		return c.onMsg(ctx, "qq", msg)
	}
	return nil
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
