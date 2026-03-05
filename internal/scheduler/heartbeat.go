// Package scheduler provides cron-based job scheduling.
package scheduler

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// HeartbeatDispatcher sends messages to channels. Used when target="last".
type HeartbeatDispatcher interface {
	Send(ctx context.Context, channel, to, text string) error
	LastDispatch() (channel, userID, sessionID string)
}

// HeartbeatRunner runs the heartbeat job: read HEARTBEAT.md → Agent.Run → optionally send to channel.
type HeartbeatRunner struct {
	agent   agent.Agent
	loader  *agent.PromptLoader
	config  config.HeartbeatConfig
	send    HeartbeatDispatcher
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewHeartbeatRunner creates a HeartbeatRunner.
func NewHeartbeatRunner(ag agent.Agent, loader *agent.PromptLoader, cfg config.HeartbeatConfig, send HeartbeatDispatcher) *HeartbeatRunner {
	return &HeartbeatRunner{
		agent:  ag,
		loader: loader,
		config: cfg,
		send:   send,
	}
}

// Start begins the heartbeat loop. Runs at config.Every interval. Returns immediately.
func (h *HeartbeatRunner) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return nil
	}
	if h.config.Every == "" {
		h.mu.Unlock()
		return nil
	}
	_, err := parseEveryToCron(h.config.Every)
	if err != nil {
		h.mu.Unlock()
		return fmt.Errorf("parse heartbeat every %q: %w", h.config.Every, err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	h.running = true
	h.mu.Unlock()

	interval := parseEveryToDuration(h.config.Every)
	go func() {
		defer func() {
			h.mu.Lock()
			h.running = false
			h.mu.Unlock()
		}()
		h.runOnce(runCtx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				h.runOnce(runCtx)
			}
		}
	}()
	return nil
}

// Stop stops the heartbeat runner.
func (h *HeartbeatRunner) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
	h.running = false
	return nil
}

func (h *HeartbeatRunner) runOnce(ctx context.Context) {
	if !h.inActiveHours() {
		return
	}
	content, err := h.loader.LoadHEARTBEAT()
	if err != nil || content == "" {
		return
	}
	chatID := "heartbeat:main"
	if h.config.Target == "last" && h.send != nil {
		ch, _, sid := h.send.LastDispatch()
		if ch != "" && sid != "" {
			chatID = sid
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	response, err := h.agent.Run(runCtx, chatID, content)
	if err != nil {
		logger.L().Warn("heartbeat run failed", zap.Error(err))
		return
	}
	if h.config.Target == "last" && h.send != nil {
		ch, uid, sid := h.send.LastDispatch()
		if ch != "" {
			to := uid
			if sid != "" && len(sid) > len(ch)+1 {
				to = sid[len(ch)+1:] // sessionID is "channel:chatID"
			}
			if to != "" {
				if err := h.send.Send(ctx, ch, to, response); err != nil {
					logger.L().Warn("heartbeat send failed", zap.Error(err))
				}
			}
		}
	}
}

func (h *HeartbeatRunner) inActiveHours() bool {
	ah := h.config.ActiveHours
	if ah == nil || ah.Start == "" || ah.End == "" {
		return true
	}
	now := time.Now()
	start, err1 := parseTime(ah.Start)
	end, err2 := parseTime(ah.End)
	if err1 != nil || err2 != nil {
		return true
	}
	t := now.Hour()*60 + now.Minute()
	return t >= start && t <= end
}

func parseTime(s string) (minutes int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format")
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m, nil
}

var everyRe = regexp.MustCompile(`(\d+)(h|m|s)`)

// parseEveryToCron converts "30m", "1h", "2h30m" to cron spec.
func parseEveryToCron(every string) (string, error) {
	d := parseEveryToDuration(every)
	if d <= 0 {
		return "", fmt.Errorf("invalid every: %s", every)
	}
	mins := int(d / time.Minute)
	if mins < 1 {
		mins = 1
	}
	if mins >= 60 {
		hours := mins / 60
		mins = mins % 60
		if mins == 0 {
			return fmt.Sprintf("0 */%d * * *", hours), nil
		}
		return fmt.Sprintf("%d */%d * * *", mins, hours), nil
	}
	return fmt.Sprintf("*/%d * * * *", mins), nil
}

func parseEveryToDuration(every string) time.Duration {
	var total time.Duration
	matches := everyRe.FindAllStringSubmatch(every, -1)
	for _, m := range matches {
		if len(m) != 3 {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "h":
			total += time.Duration(n) * time.Hour
		case "m":
			total += time.Duration(n) * time.Minute
		case "s":
			total += time.Duration(n) * time.Second
		}
	}
	if total == 0 {
		return 30 * time.Minute
	}
	return total
}
