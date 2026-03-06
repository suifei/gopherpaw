package channels

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/suifei/gopherpaw/internal/agent"
)

// consoleProgressReporter prints real-time feedback to stderr during Agent processing.
type consoleProgressReporter struct{}

func (c *consoleProgressReporter) OnThinking() {
	fmt.Fprintln(os.Stderr, "🤔 Thinking...")
}

func (c *consoleProgressReporter) OnToolCall(toolName string, args string) {
	preview := truncateForConsole(args, 80)
	fmt.Fprintf(os.Stderr, "🔧 Using tool: %s %s\n", toolName, preview)
}

func (c *consoleProgressReporter) OnToolResult(toolName string, result string) {
	fmt.Fprintf(os.Stderr, "✅ Tool %s result received\n", toolName)
}

func (c *consoleProgressReporter) OnFinalReply(content string) {
	fmt.Fprintln(os.Stderr, "─────────────────────────────────────")
}

func truncateForConsole(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if maxLen <= 0 || utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen]) + "..."
}

// ConsoleChannel reads from stdin and writes to stdout.
// Used for development and testing.
type ConsoleChannel struct {
	agent   agent.Agent
	enabled bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      sync.Mutex
	running bool
	onMsg   func(ctx context.Context, chName string, msg IncomingMessage) error
}

// NewConsole creates a console channel.
func NewConsole(ag agent.Agent, enabled bool, onMsg func(context.Context, string, IncomingMessage) error) *ConsoleChannel {
	return &ConsoleChannel{
		agent:   ag,
		enabled: enabled,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		onMsg:   onMsg,
	}
}

// Name returns the channel identifier.
func (c *ConsoleChannel) Name() string {
	return "console"
}

// Start begins reading from stdin and processing messages.
func (c *ConsoleChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	go func() {
		defer close(c.doneCh)
		lineCh := make(chan string, 1)
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				lineCh <- strings.TrimSpace(scanner.Text())
			}
			close(lineCh)
		}()
		fmt.Fprintln(os.Stderr, "Console channel ready. Type a message and press Enter (empty line to exit):")
		for {
			select {
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			case line, ok := <-lineCh:
				if !ok {
					return
				}
				if line == "" {
					return
				}
				msg := IncomingMessage{
					ChatID:    "console:default",
					UserID:    "default",
					UserName:  "user",
					Content:   line,
					Channel:   "console",
					Timestamp: time.Now().Unix(),
				}
				if c.onMsg != nil {
					runCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
					runCtx = agent.WithProgressReporter(runCtx, &consoleProgressReporter{})
					if err := c.onMsg(runCtx, "console", msg); err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					}
					cancel()
				} else {
					runCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
					runCtx = agent.WithProgressReporter(runCtx, &consoleProgressReporter{})
					response, err := c.agent.Run(runCtx, "console:default", line)
					cancel()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						continue
					}
					fmt.Println(response)
				}
				fmt.Fprintln(os.Stderr, "")
			}
		}
	}()
	return nil
}

// Send prints the text to stdout (console has no real "send" target).
func (c *ConsoleChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	if text != "" {
		fmt.Println(text)
	}
	return nil
}

// SendFile prints file info to stdout.
func (c *ConsoleChannel) SendFile(ctx context.Context, to string, filePath string, mime string, meta map[string]string) error {
	fmt.Fprintf(os.Stdout, "📎 File: %s (%s)\n", filePath, mime)
	return nil
}

// IsEnabled returns whether the console channel is enabled.
func (c *ConsoleChannel) IsEnabled() bool {
	return c.enabled
}

// Stop signals the channel to stop.
func (c *ConsoleChannel) Stop(ctx context.Context) error {
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

// Ensure ConsoleChannel implements interfaces.
var (
	_ Channel    = (*ConsoleChannel)(nil)
	_ FileSender = (*ConsoleChannel)(nil)
)
