package channels

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockChannel is a test channel implementation.
type mockChannel struct {
	name      string
	enabled   bool
	debouncer *DefaultDebouncer
}

func (c *mockChannel) Start(ctx context.Context) error { return nil }
func (c *mockChannel) Stop(ctx context.Context) error  { return nil }
func (c *mockChannel) Name() string                    { return c.name }
func (c *mockChannel) IsEnabled() bool                 { return c.enabled }
func (c *mockChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	return nil
}

// Debouncer implementation
func (c *mockChannel) GetDebounceKey(msg *IncomingMessage) string {
	return c.debouncer.GetDebounceKey(msg)
}
func (c *mockChannel) MergeMessages(msgs []*IncomingMessage) *IncomingMessage {
	return c.debouncer.MergeMessages(msgs)
}
func (c *mockChannel) ShouldDebounce(msg *IncomingMessage) bool {
	return c.debouncer.ShouldDebounce(msg)
}

var (
	_ Channel   = (*mockChannel)(nil)
	_ Debouncer = (*mockChannel)(nil)
)

func TestMessageQueue_BasicEnqueue(t *testing.T) {
	var processed int32
	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		atomic.AddInt32(&processed, 1)
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 100, Workers: 2},
		DebounceConfig{Enabled: false},
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q.Start(ctx)

	ch := &mockChannel{name: "test", enabled: true, debouncer: &DefaultDebouncer{}}
	msg := &IncomingMessage{
		ChatID:    "chat1",
		UserID:    "user1",
		Content:   "hello",
		Channel:   "test",
		Timestamp: time.Now().Unix(),
	}

	if !q.Enqueue(ch, msg) {
		t.Error("expected enqueue to succeed")
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&processed) != 1 {
		t.Errorf("expected 1 processed message, got %d", processed)
	}

	q.Stop(ctx)
}

func TestMessageQueue_Debounce(t *testing.T) {
	var mu sync.Mutex
	var received []string

	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		mu.Lock()
		received = append(received, msg.Content)
		mu.Unlock()
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 100, Workers: 1},
		DebounceConfig{Enabled: true, DelayMs: 100, MaxBufferSize: 10},
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q.Start(ctx)

	ch := &mockChannel{name: "test", enabled: true, debouncer: &DefaultDebouncer{}}

	// Send 3 messages rapidly
	for i := 0; i < 3; i++ {
		msg := &IncomingMessage{
			ChatID:    "chat1",
			Content:   "msg" + string(rune('A'+i)),
			Channel:   "test",
			Timestamp: time.Now().Unix(),
		}
		q.Enqueue(ch, msg)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should receive 1 merged message
	if len(received) != 1 {
		t.Errorf("expected 1 merged message, got %d: %v", len(received), received)
		return
	}

	// Merged content should contain all messages
	expected := "msgA\nmsgB\nmsgC"
	if received[0] != expected {
		t.Errorf("expected merged content %q, got %q", expected, received[0])
	}

	q.Stop(ctx)
}

func TestMessageQueue_DebounceSkipCommands(t *testing.T) {
	var mu sync.Mutex
	var received []string

	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		mu.Lock()
		received = append(received, msg.Content)
		mu.Unlock()
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 100, Workers: 1},
		DebounceConfig{Enabled: true, DelayMs: 100, MaxBufferSize: 10},
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q.Start(ctx)

	ch := &mockChannel{name: "test", enabled: true, debouncer: &DefaultDebouncer{}}

	// Send a command (starts with /)
	cmdMsg := &IncomingMessage{
		ChatID:    "chat1",
		Content:   "/help",
		Channel:   "test",
		Timestamp: time.Now().Unix(),
	}
	q.Enqueue(ch, cmdMsg)

	// Send a regular message
	regularMsg := &IncomingMessage{
		ChatID:    "chat1",
		Content:   "hello",
		Channel:   "test",
		Timestamp: time.Now().Unix(),
	}
	q.Enqueue(ch, regularMsg)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Command should be processed immediately (not debounced)
	// Regular message should be processed after debounce
	if len(received) != 2 {
		t.Errorf("expected 2 messages, got %d: %v", len(received), received)
		return
	}

	// Command should be first (immediate processing)
	if received[0] != "/help" {
		t.Errorf("expected first message to be /help, got %q", received[0])
	}

	q.Stop(ctx)
}

func TestMessageQueue_MaxBufferFlush(t *testing.T) {
	var mu sync.Mutex
	var received []string

	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		mu.Lock()
		received = append(received, msg.Content)
		mu.Unlock()
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 100, Workers: 1},
		DebounceConfig{Enabled: true, DelayMs: 1000, MaxBufferSize: 3}, // Long delay, small buffer
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q.Start(ctx)

	ch := &mockChannel{name: "test", enabled: true, debouncer: &DefaultDebouncer{}}

	// Send 3 messages rapidly (hits MaxBufferSize)
	for i := 0; i < 3; i++ {
		msg := &IncomingMessage{
			ChatID:    "chat1",
			Content:   "msg" + string(rune('A'+i)),
			Channel:   "test",
			Timestamp: time.Now().Unix(),
		}
		q.Enqueue(ch, msg)
	}

	// Should flush immediately due to MaxBufferSize
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Errorf("expected 1 message (immediate flush), got %d", len(received))
	}

	q.Stop(ctx)
}

func TestMessageQueue_Stats(t *testing.T) {
	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		time.Sleep(50 * time.Millisecond) // Slow handler
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 100, Workers: 1},
		DebounceConfig{Enabled: false},
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q.Start(ctx)

	ch := &mockChannel{name: "test", enabled: true, debouncer: &DefaultDebouncer{}}

	// Enqueue several messages
	for i := 0; i < 5; i++ {
		msg := &IncomingMessage{
			ChatID:    "chat" + string(rune('1'+i)),
			Content:   "hello",
			Channel:   "test",
			Timestamp: time.Now().Unix(),
		}
		q.Enqueue(ch, msg)
	}

	stats := q.Stats()
	if stats.QueueCapacity != 100 {
		t.Errorf("expected capacity 100, got %d", stats.QueueCapacity)
	}

	q.Stop(ctx)
}

func TestDefaultDebouncer_GetDebounceKey(t *testing.T) {
	d := &DefaultDebouncer{}

	tests := []struct {
		name string
		msg  *IncomingMessage
		want string
	}{
		{
			name: "with chat_id",
			msg:  &IncomingMessage{Channel: "telegram", ChatID: "123", UserID: "456"},
			want: "telegram:123",
		},
		{
			name: "without chat_id",
			msg:  &IncomingMessage{Channel: "telegram", ChatID: "", UserID: "456"},
			want: "telegram:456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.GetDebounceKey(tt.msg)
			if got != tt.want {
				t.Errorf("GetDebounceKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultDebouncer_MergeMessages(t *testing.T) {
	d := &DefaultDebouncer{}

	tests := []struct {
		name string
		msgs []*IncomingMessage
		want string
	}{
		{
			name: "empty",
			msgs: nil,
			want: "",
		},
		{
			name: "single",
			msgs: []*IncomingMessage{{Content: "hello"}},
			want: "hello",
		},
		{
			name: "multiple",
			msgs: []*IncomingMessage{
				{Content: "hello", Timestamp: 100},
				{Content: "world", Timestamp: 200},
			},
			want: "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.MergeMessages(tt.msgs)
			if tt.want == "" {
				if got != nil {
					t.Errorf("MergeMessages() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Error("MergeMessages() returned nil, want non-nil")
				return
			}
			if got.Content != tt.want {
				t.Errorf("MergeMessages().Content = %q, want %q", got.Content, tt.want)
			}
		})
	}
}

func TestDefaultDebouncer_ShouldDebounce(t *testing.T) {
	d := &DefaultDebouncer{}

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"regular message", "hello world", true},
		{"command", "/help", false},
		{"command with args", "/search foo", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &IncomingMessage{Content: tt.content}
			got := d.ShouldDebounce(msg)
			if got != tt.want {
				t.Errorf("ShouldDebounce(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestMessageQueue_ConcurrentEnqueue(t *testing.T) {
	var processed int32

	handler := func(ctx context.Context, chName string, msg IncomingMessage) error {
		atomic.AddInt32(&processed, 1)
		return nil
	}

	q := NewMessageQueue(
		QueueConfig{MaxSize: 1000, Workers: 4},
		DebounceConfig{Enabled: false},
		handler,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q.Start(ctx)

	ch := &mockChannel{name: "test", enabled: true, debouncer: &DefaultDebouncer{}}

	// Concurrent enqueue from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				msg := &IncomingMessage{
					ChatID:    "chat" + string(rune('A'+id)),
					Content:   "message",
					Channel:   "test",
					Timestamp: time.Now().Unix(),
				}
				q.Enqueue(ch, msg)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	if atomic.LoadInt32(&processed) != 100 {
		t.Errorf("expected 100 processed messages, got %d", processed)
	}

	q.Stop(ctx)
}
