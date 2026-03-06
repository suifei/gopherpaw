package channels

import (
	"context"
	"testing"
	"time"
)

func BenchmarkMessageQueue_Enqueue(b *testing.B) {
	q := NewMessageQueue(DefaultQueueConfig(), DefaultDebounceConfig(), func(ctx context.Context, chName string, msg IncomingMessage) error {
		return nil
	})
	ctx := context.Background()
	q.Start(ctx)
	defer q.Stop(ctx)

	ch := &mockDebouncerChannel{}
	msg := &IncomingMessage{
		Channel:   "test",
		ChatID:    "chat-1",
		UserID:    "user-1",
		Content:   "test message",
		Timestamp: time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(ch, msg)
	}
}

func BenchmarkMessageQueue_Enqueue_Debounced(b *testing.B) {
	q := NewMessageQueue(DefaultQueueConfig(), DefaultDebounceConfig(), func(ctx context.Context, chName string, msg IncomingMessage) error {
		return nil
	})
	ctx := context.Background()
	q.Start(ctx)
	defer q.Stop(ctx)

	ch := &mockDebouncerChannel{enabled: true}
	msg := &IncomingMessage{
		Channel:   "test",
		ChatID:    "chat-1",
		UserID:    "user-1",
		Content:   "test message",
		Timestamp: time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(ch, msg)
	}
}

func BenchmarkMessageQueue_Stats_WithEnqueue(b *testing.B) {
	q := NewMessageQueue(DefaultQueueConfig(), DefaultDebounceConfig(), func(ctx context.Context, chName string, msg IncomingMessage) error {
		return nil
	})
	ctx := context.Background()
	q.Start(ctx)
	defer q.Stop(ctx)

	ch := &mockDebouncerChannel{}
	msg := &IncomingMessage{
		Channel:   "test",
		ChatID:    "chat-1",
		UserID:    "user-1",
		Content:   "test message",
		Timestamp: time.Now().Unix(),
	}

	for i := 0; i < 1000; i++ {
		q.Enqueue(ch, msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Stats()
	}
}

func BenchmarkMessageQueue_Stats(b *testing.B) {
	q := NewMessageQueue(DefaultQueueConfig(), DefaultDebounceConfig(), func(ctx context.Context, chName string, msg IncomingMessage) error {
		return nil
	})
	ctx := context.Background()
	q.Start(ctx)
	defer q.Stop(ctx)

	ch := &mockDebouncerChannel{}
	msg := &IncomingMessage{
		Channel:   "test",
		ChatID:    "chat-1",
		UserID:    "user-1",
		Content:   "test message",
		Timestamp: time.Now().Unix(),
	}

	for i := 0; i < 100; i++ {
		q.Enqueue(ch, msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Stats()
	}
}

func BenchmarkMergeMessages_Small(b *testing.B) {
	msgs := make([]*IncomingMessage, 10)
	for i := 0; i < 10; i++ {
		msgs[i] = &IncomingMessage{
			Channel:   "test",
			ChatID:    "chat-1",
			UserID:    "user-1",
			Content:   "test message",
			Timestamp: time.Now().Unix(),
		}
	}

	debouncer := &DefaultDebouncer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debouncer.MergeMessages(msgs)
	}
}

func BenchmarkMergeMessages_Medium(b *testing.B) {
	msgs := make([]*IncomingMessage, 100)
	for i := 0; i < 100; i++ {
		msgs[i] = &IncomingMessage{
			Channel:   "test",
			ChatID:    "chat-1",
			UserID:    "user-1",
			Content:   "test message",
			Timestamp: time.Now().Unix(),
		}
	}

	debouncer := &DefaultDebouncer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debouncer.MergeMessages(msgs)
	}
}

func BenchmarkMergeMessages_Large(b *testing.B) {
	msgs := make([]*IncomingMessage, 1000)
	for i := 0; i < 1000; i++ {
		msgs[i] = &IncomingMessage{
			Channel:   "test",
			ChatID:    "chat-1",
			UserID:    "user-1",
			Content:   "test message",
			Timestamp: time.Now().Unix(),
		}
	}

	debouncer := &DefaultDebouncer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debouncer.MergeMessages(msgs)
	}
}

type mockDebouncerChannel struct {
	name      string
	enabled   bool
	debouncer Debouncer
}

func (m *mockDebouncerChannel) Name() string {
	if m.name == "" {
		return "mock"
	}
	return m.name
}

func (m *mockDebouncerChannel) Start(ctx context.Context) error {
	return nil
}

func (m *mockDebouncerChannel) Stop(ctx context.Context) error {
	return nil
}

func (m *mockDebouncerChannel) Send(ctx context.Context, to string, text string, meta map[string]string) error {
	return nil
}

func (m *mockDebouncerChannel) SendFile(ctx context.Context, to string, filePath string, mime string, meta map[string]string) error {
	return nil
}

func (m *mockDebouncerChannel) IsEnabled() bool {
	return m.enabled
}

func (m *mockDebouncerChannel) GetDebounceKey(msg *IncomingMessage) string {
	return msg.Channel + ":" + msg.ChatID
}

func (m *mockDebouncerChannel) MergeMessages(msgs []*IncomingMessage) *IncomingMessage {
	if len(msgs) == 0 {
		return nil
	}
	if len(msgs) == 1 {
		return msgs[0]
	}
	merged := *msgs[0]
	for i := 1; i < len(msgs); i++ {
		merged.Content += "\n" + msgs[i].Content
	}
	merged.Timestamp = msgs[len(msgs)-1].Timestamp
	return &merged
}

func (m *mockDebouncerChannel) ShouldDebounce(msg *IncomingMessage) bool {
	return true
}
