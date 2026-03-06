package channels

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const (
	feishuWSEndpoint      = "wss://open.feishu.cn/open-apis/ws/v2/connect"
	feishuPingInterval    = 120 * time.Second
	feishuReconnectDelay  = 3 * time.Second
	feishuReconnectJitter = 30 * time.Second
	feishuTokenRefreshBuf = 300 * time.Second // Refresh 5 min before expiry
)

// FeishuWSChannel implements Channel for Feishu using WebSocket long connection.
// This is the recommended approach as it doesn't require a public webhook URL.
type FeishuWSChannel struct {
	agent  agent.Agent
	cfg    config.FeishuConfig
	client *http.Client
	onMsg  func(ctx context.Context, chName string, msg IncomingMessage) error

	conn   *websocket.Conn
	connMu sync.Mutex

	tokenMu       sync.Mutex
	token         string
	tokenExpireAt time.Time

	processedMu  sync.Mutex
	processedIDs map[string]time.Time // Message ID deduplication

	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
	running bool
}

// NewFeishuWS creates a Feishu WebSocket channel.
func NewFeishuWS(ag agent.Agent, cfg config.FeishuConfig, onMsg func(context.Context, string, IncomingMessage) error) *FeishuWSChannel {
	return &FeishuWSChannel{
		agent:        ag,
		cfg:          cfg,
		client:       &http.Client{Timeout: 30 * time.Second},
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		onMsg:        onMsg,
		processedIDs: make(map[string]time.Time),
	}
}

// Name returns the channel identifier.
func (c *FeishuWSChannel) Name() string {
	return "feishu"
}

// IsEnabled returns whether the channel is enabled.
func (c *FeishuWSChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.UseWebSocket && c.cfg.AppID != "" && c.cfg.AppSecret != ""
}

// Start establishes the WebSocket connection and begins processing messages.
func (c *FeishuWSChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	go c.runLoop(ctx)
	go c.cleanupLoop()
	logger.L().Info("feishu websocket channel started")
	return nil
}

// Stop gracefully shuts down the channel.
func (c *FeishuWSChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	close(c.stopCh)

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()

	select {
	case <-c.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Send sends a text message using the Feishu API.
func (c *FeishuWSChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
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

	contentJSON, _ := json.Marshal(map[string]string{"text": c.cfg.BotPrefix + text})
	body := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=%s", receiveIDType)
	return c.postWithToken(ctx, url, token, body)
}

func (c *FeishuWSChannel) postWithToken(ctx context.Context, url, token string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu API error: status %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

func (c *FeishuWSChannel) getToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != "" && time.Now().Add(feishuTokenRefreshBuf).Before(c.tokenExpireAt) {
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
	c.tokenExpireAt = time.Now().Add(time.Duration(data.Expire) * time.Second)
	return c.token, nil
}

// runLoop maintains the WebSocket connection with automatic reconnection.
func (c *FeishuWSChannel) runLoop(ctx context.Context) {
	defer close(c.doneCh)

	reconnectCount := 0
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		// Add jitter on reconnect (not first connect)
		if reconnectCount > 0 {
			jitter := time.Duration(reconnectCount%10) * time.Second
			select {
			case <-c.stopCh:
				return
			case <-time.After(feishuReconnectDelay + jitter):
			}
		}

		if err := c.connect(ctx); err != nil {
			logger.L().Warn("feishu ws connect failed", zap.Error(err), zap.Int("attempt", reconnectCount))
			reconnectCount++
			continue
		}

		reconnectCount = 0
		c.processMessages(ctx)
	}
}

// connect establishes the WebSocket connection.
func (c *FeishuWSChannel) connect(ctx context.Context) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	dialer := websocket.DefaultDialer
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, _, err := dialer.DialContext(ctx, feishuWSEndpoint, header)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	logger.L().Info("feishu websocket connected")
	return nil
}

// processMessages reads and processes messages from the WebSocket.
func (c *FeishuWSChannel) processMessages(ctx context.Context) {
	pingTicker := time.NewTicker(feishuPingInterval)
	defer pingTicker.Stop()

	msgCh := make(chan []byte)
	errCh := make(chan error)

	go func() {
		for {
			c.connMu.Lock()
			conn := c.conn
			c.connMu.Unlock()

			if conn == nil {
				return
			}

			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			msgCh <- msg
		}
	}()

	for {
		select {
		case <-c.stopCh:
			return

		case <-pingTicker.C:
			c.connMu.Lock()
			conn := c.conn
			c.connMu.Unlock()
			if conn != nil {
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.L().Warn("feishu ping failed", zap.Error(err))
					return
				}
			}

		case err := <-errCh:
			logger.L().Warn("feishu websocket read error", zap.Error(err))
			return

		case msg := <-msgCh:
			c.handleMessage(ctx, msg)
		}
	}
}

// feishuWSMessage represents a WebSocket message frame.
type feishuWSMessage struct {
	Type    string          `json:"type"` // "event", "card", "ping", "pong"
	Header  json.RawMessage `json:"header"`
	Event   json.RawMessage `json:"event"`
	Encrypt string          `json:"encrypt"` // Encrypted payload if encrypt_key is set
}

// feishuEventHeader represents the event header.
type feishuEventHeader struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Token     string `json:"token"`
}

// feishuMessageEvent represents an im.message.receive_v1 event.
type feishuMessageEvent struct {
	Sender struct {
		SenderID struct {
			OpenID string `json:"open_id"`
		} `json:"sender_id"`
		SenderType string `json:"sender_type"`
	} `json:"sender"`
	Message struct {
		MessageID   string `json:"message_id"`
		ChatID      string `json:"chat_id"`
		ChatType    string `json:"chat_type"` // "p2p" or "group"
		MessageType string `json:"message_type"`
		Content     string `json:"content"`
	} `json:"message"`
}

// handleMessage processes a single WebSocket message.
func (c *FeishuWSChannel) handleMessage(ctx context.Context, data []byte) {
	var msg feishuWSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.L().Warn("feishu unmarshal message failed", zap.Error(err))
		return
	}

	// Handle encrypted messages
	if msg.Encrypt != "" {
		decrypted, err := c.decryptMessage(msg.Encrypt)
		if err != nil {
			logger.L().Warn("feishu decrypt failed", zap.Error(err))
			return
		}
		if err := json.Unmarshal(decrypted, &msg); err != nil {
			logger.L().Warn("feishu unmarshal decrypted message failed", zap.Error(err))
			return
		}
	}

	switch msg.Type {
	case "event":
		c.handleEventMessage(ctx, msg)
	case "ping":
		c.sendPong()
	default:
		logger.L().Debug("feishu unknown message type", zap.String("type", msg.Type))
	}
}

func (c *FeishuWSChannel) handleEventMessage(ctx context.Context, msg feishuWSMessage) {
	var header feishuEventHeader
	if err := json.Unmarshal(msg.Header, &header); err != nil {
		logger.L().Warn("feishu unmarshal event header failed", zap.Error(err))
		return
	}

	// Verify token if configured
	if c.cfg.VerificationToken != "" && header.Token != c.cfg.VerificationToken {
		logger.L().Warn("feishu verification token mismatch")
		return
	}

	// Check for duplicate message
	if c.isDuplicate(header.EventID) {
		return
	}

	// Only handle im.message.receive_v1 events
	if header.EventType != "im.message.receive_v1" {
		return
	}

	var event feishuMessageEvent
	if err := json.Unmarshal(msg.Event, &event); err != nil {
		logger.L().Warn("feishu unmarshal message event failed", zap.Error(err))
		return
	}

	// Skip bot messages
	if event.Sender.SenderType == "app" {
		return
	}

	// Extract text content
	text := c.extractTextContent(event.Message.Content)
	if text == "" {
		return
	}

	incoming := IncomingMessage{
		ChatID:    event.Message.ChatID,
		UserID:    event.Sender.SenderID.OpenID,
		Content:   text,
		Channel:   "feishu",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"chat_id":         event.Message.ChatID,
			"chat_type":       event.Message.ChatType,
			"message_id":      event.Message.MessageID,
			"receive_id":      c.getReceiveID(event),
			"receive_id_type": c.getReceiveIDType(event),
		},
	}

	if c.onMsg != nil {
		go func() {
			runCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()
			if err := c.onMsg(runCtx, "feishu", incoming); err != nil {
				logger.L().Warn("feishu onMsg failed", zap.Error(err))
			}
		}()
	}
}

func (c *FeishuWSChannel) extractTextContent(content string) string {
	var contentObj struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &contentObj); err != nil {
		return ""
	}
	return contentObj.Text
}

func (c *FeishuWSChannel) getReceiveID(event feishuMessageEvent) string {
	if event.Message.ChatType == "p2p" {
		return event.Sender.SenderID.OpenID
	}
	return event.Message.ChatID
}

func (c *FeishuWSChannel) getReceiveIDType(event feishuMessageEvent) string {
	if event.Message.ChatType == "p2p" {
		return "open_id"
	}
	return "chat_id"
}

func (c *FeishuWSChannel) sendPong() {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		return
	}

	pong := map[string]string{"type": "pong"}
	data, _ := json.Marshal(pong)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.L().Warn("feishu send pong failed", zap.Error(err))
	}
}

// isDuplicate checks if the event has already been processed.
func (c *FeishuWSChannel) isDuplicate(eventID string) bool {
	if eventID == "" {
		return false
	}

	c.processedMu.Lock()
	defer c.processedMu.Unlock()

	if _, ok := c.processedIDs[eventID]; ok {
		return true
	}

	c.processedIDs[eventID] = time.Now()
	return false
}

// cleanupLoop periodically cleans up old processed message IDs.
func (c *FeishuWSChannel) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.processedMu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for id, t := range c.processedIDs {
				if t.Before(cutoff) {
					delete(c.processedIDs, id)
				}
			}
			c.processedMu.Unlock()
		}
	}
}

// decryptMessage decrypts an encrypted message using AES-256-CBC.
func (c *FeishuWSChannel) decryptMessage(encrypted string) ([]byte, error) {
	if c.cfg.EncryptKey == "" {
		return nil, fmt.Errorf("encrypt_key not configured")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	// Key is SHA256 of encrypt_key
	key := sha256.Sum256([]byte(c.cfg.EncryptKey))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Remove PKCS7 padding
	padding := int(ciphertext[len(ciphertext)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	return ciphertext[:len(ciphertext)-padding], nil
}

// Ensure FeishuWSChannel implements Channel interface.
var _ Channel = (*FeishuWSChannel)(nil)
