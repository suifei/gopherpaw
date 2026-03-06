package channels

import (
	"context"
	"sync"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgentForConcurrency struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForConcurrency) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "response", nil
}

func (m *mockAgentForConcurrency) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "stream"
	close(ch)
	return ch, nil
}

// TestManagerConcurrentSends tests concurrent send operations
func TestManagerConcurrentSends(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForConcurrency{}, cfg)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx := context.Background()
			err := m.Send(ctx, "console", "user", "test message")
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Errorf("concurrent send error: %v", err)
		}
	}
}

// TestManagerConcurrentRegisterUnregister tests concurrent register/unregister
func TestManagerConcurrentRegisterUnregister(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForConcurrency{}, cfg)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ch := NewConsole(&mockAgentForConcurrency{}, true, nil)
			m.Register(ch)
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m.Unregister("console")
		}(i)
	}

	wg.Wait()
}

// TestManagerConcurrentChannels tests concurrent access to channels
func TestManagerConcurrentChannels(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForConcurrency{}, cfg)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			channels := m.Channels()
			if len(channels) == 0 {
				t.Errorf("expected channels")
			}
		}()
	}

	wg.Wait()
}

// TestManagerConcurrentLastDispatch tests concurrent lastDispatch access
func TestManagerConcurrentLastDispatch(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForConcurrency{}, cfg)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ch, userID, sessionID := m.LastDispatch()
			_ = ch
			_ = userID
			_ = sessionID
		}(i)
	}

	wg.Wait()
}

// TestWebhookServerConcurrentRegister tests concurrent handler registration
func TestWebhookServerConcurrentRegister(t *testing.T) {
	server := NewWebhookServer("localhost", 8080)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			handler := &mockWebhookHandler{}
			server.Register("test", handler)
		}(i)
	}

	wg.Wait()
}

// TestManagerConcurrentSetDaemonInfo tests concurrent DaemonInfo setting
func TestManagerConcurrentSetDaemonInfo(t *testing.T) {
	cfg := config.ChannelsConfig{
		Console: config.ConsoleConfig{Enabled: true},
	}
	m := NewManager(&mockAgentForConcurrency{}, cfg)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info := &agent.DaemonInfo{}
			m.SetDaemonInfo(info)
		}()
	}

	wg.Wait()
}
