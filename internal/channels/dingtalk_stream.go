package channels

import (
	"context"
	"encoding/json"
	"fmt"
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
	dingtalkStreamEndpoint   = "https://api.dingtalk.com/v1.0/gateway/connections/open"
	dingtalkKeepAliveTimeout = 90 * time.Second
	dingtalkReconnectDelay   = 3 * time.Second
	dingtalkPingInterval     = 30 * time.Second
)

// DingTalkStreamChannel implements Channel for DingTalk Stream mode using WebSocket.
// This is the recommended approach as it doesn't require a public webhook URL.
type DingTalkStreamChannel struct {
	agent  agent.Agent
	cfg    config.DingTalkConfig
	client *http.Client
	onMsg  func(ctx context.Context, chName string, msg IncomingMessage) error

	conn   *websocket.Conn
	connMu sync.Mutex

	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
	running bool
}

// NewDingTalkStream creates a DingTalk Stream channel.
func NewDingTalkStream(ag agent.Agent, cfg config.DingTalkConfig, onMsg func(context.Context, string, IncomingMessage) error) *DingTalkStreamChannel {
	return &DingTalkStreamChannel{
		agent:  ag,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		onMsg:  onMsg,
	}
}

// Name returns the channel identifier.
func (c *DingTalkStreamChannel) Name() string {
	return "dingtalk"
}

// IsEnabled returns whether the channel is enabled.
func (c *DingTalkStreamChannel) IsEnabled() bool {
	return c.cfg.Enabled && c.cfg.UseStream && c.cfg.ClientID != "" && c.cfg.ClientSecret != ""
}

// Start establishes the WebSocket connection and begins processing messages.
func (c *DingTalkStreamChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	go c.runLoop(ctx)
	logger.L().Info("dingtalk stream channel started")
	return nil
}

// Stop gracefully shuts down the channel.
func (c *DingTalkStreamChannel) Stop(ctx context.Context) error {
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

// Send sends a text message via session_webhook (same as non-stream DingTalk).
func (c *DingTalkStreamChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
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
		return fmt.Errorf("dingtalk send: session_webhook required in meta")
	}
	body := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": c.cfg.BotPrefix + text},
	}
	return c.postJSON(ctx, webhook, body)
}

func (c *DingTalkStreamChannel) postJSON(ctx context.Context, url string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Body = &nopCloser{data: data}
	req.ContentLength = int64(len(data))

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("dingtalk API error: status %d", resp.StatusCode)
	}
	return nil
}

// runLoop maintains the WebSocket connection with automatic reconnection.
func (c *DingTalkStreamChannel) runLoop(ctx context.Context) {
	defer close(c.doneCh)

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		if err := c.connect(ctx); err != nil {
			logger.L().Warn("dingtalk stream connect failed", zap.Error(err))
			select {
			case <-c.stopCh:
				return
			case <-time.After(dingtalkReconnectDelay):
				continue
			}
		}

		c.processMessages(ctx)

		// Connection lost, reconnect
		select {
		case <-c.stopCh:
			return
		case <-time.After(dingtalkReconnectDelay):
		}
	}
}

// connect establishes the WebSocket connection.
func (c *DingTalkStreamChannel) connect(ctx context.Context) error {
	// Get connection endpoint
	endpoint, err := c.getConnectionEndpoint(ctx)
	if err != nil {
		return fmt.Errorf("get endpoint: %w", err)
	}

	// Connect to WebSocket
	dialer := websocket.DefaultDialer
	wsURL := fmt.Sprintf("%s?ticket=%s", endpoint.Endpoint, endpoint.Ticket)

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	logger.L().Info("dingtalk stream connected", zap.String("endpoint", endpoint.Endpoint))
	return nil
}

// dingtalkEndpointResponse represents the response from the connection endpoint API.
type dingtalkEndpointResponse struct {
	Endpoint string `json:"endpoint"`
	Ticket   string `json:"ticket"`
}

// getConnectionEndpoint fetches the WebSocket endpoint and ticket.
func (c *DingTalkStreamChannel) getConnectionEndpoint(ctx context.Context) (*dingtalkEndpointResponse, error) {
	body := map[string]interface{}{
		"clientId":     c.cfg.ClientID,
		"clientSecret": c.cfg.ClientSecret,
		"subscriptions": []map[string]string{
			{"type": "CALLBACK", "topic": "/v1.0/im/bot/messages/get"},
		},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", dingtalkStreamEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Body = &nopCloser{data: data}
	req.ContentLength = int64(len(data))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("endpoint API status %d", resp.StatusCode)
	}

	var result dingtalkEndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Endpoint == "" || result.Ticket == "" {
		return nil, fmt.Errorf("invalid endpoint response")
	}

	return &result, nil
}

// processMessages reads and processes messages from the WebSocket.
func (c *DingTalkStreamChannel) processMessages(ctx context.Context) {
	pingTicker := time.NewTicker(dingtalkPingInterval)
	defer pingTicker.Stop()

	// Message read channel
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
					logger.L().Warn("dingtalk ping failed", zap.Error(err))
					return
				}
			}

		case err := <-errCh:
			logger.L().Warn("dingtalk websocket read error", zap.Error(err))
			return

		case msg := <-msgCh:
			c.handleMessage(ctx, msg)
		}
	}
}

// dingtalkStreamMessage represents a message from the DingTalk Stream.
type dingtalkStreamMessage struct {
	SpecVersion string                 `json:"specVersion"`
	Type        string                 `json:"type"`
	Headers     map[string]interface{} `json:"headers"`
	Data        json.RawMessage        `json:"data"`
}

// dingtalkBotMessage represents a bot message payload.
type dingtalkBotMessage struct {
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	SenderNick     string `json:"senderNick"`
	Text           struct {
		Content string `json:"content"`
	} `json:"text"`
	SessionWebhook string `json:"sessionWebhook"`
	MsgType        string `json:"msgtype"`
}

// handleMessage processes a single WebSocket message.
func (c *DingTalkStreamChannel) handleMessage(ctx context.Context, data []byte) {
	var msg dingtalkStreamMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.L().Warn("dingtalk unmarshal message failed", zap.Error(err))
		return
	}

	// Handle different message types
	switch msg.Type {
	case "SYSTEM":
		c.handleSystemMessage(msg)
	case "CALLBACK":
		c.handleCallbackMessage(ctx, msg)
	default:
		logger.L().Debug("dingtalk unknown message type", zap.String("type", msg.Type))
	}
}

func (c *DingTalkStreamChannel) handleSystemMessage(msg dingtalkStreamMessage) {
	// System messages (connection established, pong, etc.)
	logger.L().Debug("dingtalk system message", zap.Any("headers", msg.Headers))
}

func (c *DingTalkStreamChannel) handleCallbackMessage(ctx context.Context, msg dingtalkStreamMessage) {
	var botMsg dingtalkBotMessage
	if err := json.Unmarshal(msg.Data, &botMsg); err != nil {
		logger.L().Warn("dingtalk unmarshal bot message failed", zap.Error(err))
		return
	}

	content := botMsg.Text.Content
	if content == "" {
		return
	}

	incoming := IncomingMessage{
		ChatID:    botMsg.ConversationID,
		UserID:    botMsg.SenderID,
		UserName:  botMsg.SenderNick,
		Content:   content,
		Channel:   "dingtalk",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"session_webhook": botMsg.SessionWebhook,
		},
	}

	if c.onMsg != nil {
		go func() {
			runCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()
			if err := c.onMsg(runCtx, "dingtalk", incoming); err != nil {
				logger.L().Warn("dingtalk onMsg failed", zap.Error(err))
			}
		}()
	}

	// Send ACK
	c.sendAck(msg)
}

func (c *DingTalkStreamChannel) sendAck(msg dingtalkStreamMessage) {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		return
	}

	// Extract message ID from headers
	msgID := ""
	if headers := msg.Headers; headers != nil {
		if id, ok := headers["messageId"].(string); ok {
			msgID = id
		}
	}

	ack := map[string]interface{}{
		"code":      200,
		"headers":   map[string]string{"contentType": "application/json"},
		"message":   "OK",
		"data":      "{}",
		"messageId": msgID,
	}

	data, _ := json.Marshal(ack)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.L().Warn("dingtalk send ack failed", zap.Error(err))
	}
}

// Ensure DingTalkStreamChannel implements Channel interface.
var _ Channel = (*DingTalkStreamChannel)(nil)

// nopCloser is a simple io.ReadCloser wrapper for []byte.
type nopCloser struct {
	data   []byte
	offset int
}

func (n *nopCloser) Read(p []byte) (int, error) {
	if n.offset >= len(n.data) {
		return 0, fmt.Errorf("EOF")
	}
	nn := copy(p, n.data[n.offset:])
	n.offset += nn
	return nn, nil
}

func (n *nopCloser) Close() error {
	return nil
}
