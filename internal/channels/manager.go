package channels

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// Manager starts and stops all enabled channels and routes messages to the agent.
type Manager struct {
	agent      agent.Agent
	channels   []Channel
	chMap      map[string]Channel
	mu         sync.RWMutex
	wg         sync.WaitGroup
	daemonInfo *agent.DaemonInfo

	// lastDispatch records the last message source (channel, userID, sessionID).
	lastDispatch struct {
		mu        sync.RWMutex
		channel   string
		userID    string
		sessionID string
		at        time.Time
	}
}

// NewManager creates a channel manager with enabled channels from config.
func NewManager(ag agent.Agent, cfg config.ChannelsConfig) *Manager {
	m := &Manager{
		agent:    ag,
		channels: nil,
		chMap:    make(map[string]Channel),
	}
	m.channels = m.buildChannels(cfg)
	for _, ch := range m.channels {
		m.chMap[ch.Name()] = ch
	}
	return m
}

func (m *Manager) buildChannels(cfg config.ChannelsConfig) []Channel {
	var out []Channel
	onMsg := m.handleMessage

	if cfg.Console.Enabled {
		out = append(out, NewConsole(m.agent, true, onMsg))
	}
	if cfg.Telegram.Enabled && cfg.Telegram.BotToken != "" {
		out = append(out, NewTelegram(m.agent, cfg.Telegram, onMsg))
	}
	if cfg.Discord.Enabled && cfg.Discord.BotToken != "" {
		out = append(out, NewDiscord(m.agent, cfg.Discord, onMsg))
	}
	if cfg.DingTalk.Enabled && cfg.DingTalk.ClientID != "" && cfg.DingTalk.ClientSecret != "" {
		out = append(out, NewDingTalk(m.agent, cfg.DingTalk, onMsg))
	}
	if cfg.Feishu.Enabled && cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		out = append(out, NewFeishu(m.agent, cfg.Feishu, onMsg))
	}
	if cfg.QQ.Enabled && cfg.QQ.AppID != "" && cfg.QQ.ClientSecret != "" {
		out = append(out, NewQQ(m.agent, cfg.QQ, onMsg))
	}
	return out
}

// Channels returns all registered channels (for webhook registration).
func (m *Manager) Channels() []Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Channel, len(m.channels))
	copy(out, m.channels)
	return out
}

// SetDaemonInfo sets DaemonInfo for /daemon magic commands (reload-config, restart).
func (m *Manager) SetDaemonInfo(info *agent.DaemonInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.daemonInfo = info
}

// handleMessage routes an incoming message to the agent and sends the reply.
func (m *Manager) handleMessage(ctx context.Context, chName string, msg IncomingMessage) error {
	m.mu.RLock()
	info := m.daemonInfo
	ch := m.chMap[chName]
	m.mu.RUnlock()
	if info != nil {
		ctx = agent.WithDaemonInfo(ctx, info)
	}

	meta := msg.Metadata
	if meta == nil {
		meta = make(map[string]string)
	}

	ctx = agent.WithFileSender(ctx, func(fctx context.Context, att agent.Attachment) error {
		if ch == nil {
			return nil
		}
		if sender, ok := ch.(FileSender); ok {
			return sender.SendFile(fctx, msg.ChatID, att.FilePath, att.MimeType, meta)
		}
		return ch.Send(fctx, msg.ChatID, fmt.Sprintf("[file: %s]", att.FilePath), meta)
	})

	sessionID := msg.Channel + ":" + msg.ChatID
	if msg.ChatID == "" {
		sessionID = msg.Channel + ":" + msg.UserID
	}
	response, err := m.agent.Run(ctx, sessionID, msg.Content)
	if err != nil {
		response = "Error: " + err.Error()
		logger.L().Warn("agent run failed", zap.String("channel", chName), zap.Error(err))
	}
	if ch != nil {
		if err := ch.Send(ctx, msg.ChatID, response, meta); err != nil {
			logger.L().Warn("channel send failed", zap.String("channel", chName), zap.Error(err))
		}
	}
	m.recordLastDispatch(chName, msg.UserID, sessionID)
	return nil
}

func (m *Manager) recordLastDispatch(channel, userID, sessionID string) {
	m.lastDispatch.mu.Lock()
	m.lastDispatch.channel = channel
	m.lastDispatch.userID = userID
	m.lastDispatch.sessionID = sessionID
	m.lastDispatch.at = time.Now()
	m.lastDispatch.mu.Unlock()
}

// LastDispatch returns the last message source (channel, userID, sessionID).
func (m *Manager) LastDispatch() (channel, userID, sessionID string) {
	m.lastDispatch.mu.RLock()
	defer m.lastDispatch.mu.RUnlock()
	return m.lastDispatch.channel, m.lastDispatch.userID, m.lastDispatch.sessionID
}

// Send sends a text message to the given channel and target. Implements scheduler.HeartbeatDispatcher.
func (m *Manager) Send(ctx context.Context, channel, to, text string) error {
	m.mu.RLock()
	ch := m.chMap[channel]
	m.mu.RUnlock()
	if ch == nil {
		return nil
	}
	return ch.Send(ctx, to, text, nil)
}

// Register adds or replaces a channel dynamically.
func (m *Manager) Register(ch Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := ch.Name()
	for i, c := range m.channels {
		if c.Name() == name {
			m.channels[i] = ch
			m.chMap[name] = ch
			return
		}
	}
	m.channels = append(m.channels, ch)
	m.chMap[name] = ch
}

// Unregister removes a channel by name.
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.channels {
		if c.Name() == name {
			m.channels = append(m.channels[:i], m.channels[i+1:]...)
			delete(m.chMap, name)
			return
		}
	}
}

// Start starts all channels in separate goroutines.
func (m *Manager) Start(ctx context.Context) error {
	for _, ch := range m.channels {
		ch := ch
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := ch.Start(ctx); err != nil {
				logger.L().Warn("channel start error", zap.String("channel", ch.Name()), zap.Error(err))
			}
		}()
	}
	return nil
}

// Stop stops all channels and waits for them to finish.
func (m *Manager) Stop(ctx context.Context) error {
	for _, ch := range m.channels {
		ch.Stop(ctx)
	}
	m.wg.Wait()
	return nil
}
