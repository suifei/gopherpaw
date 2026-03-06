// Package channels provides message channel adapters.
package channels

import (
	"context"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// QueueConfig configures the message queue behavior.
type QueueConfig struct {
	// MaxSize is the maximum queue capacity per channel (default 1000).
	MaxSize int
	// Workers is the number of consumer workers per channel (default 4).
	Workers int
}

// DefaultQueueConfig returns the default queue configuration.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		MaxSize: 1000,
		Workers: 4,
	}
}

// DebounceConfig configures debounce behavior for rapid messages.
type DebounceConfig struct {
	// Enabled enables debounce for rapid messages.
	Enabled bool
	// DelayMs is the debounce delay in milliseconds (default 300).
	DelayMs int
	// MaxBufferSize is the maximum messages to buffer before forced flush.
	MaxBufferSize int
}

// DefaultDebounceConfig returns the default debounce configuration.
func DefaultDebounceConfig() DebounceConfig {
	return DebounceConfig{
		Enabled:       true,
		DelayMs:       300,
		MaxBufferSize: 10,
	}
}

// Debouncer is an optional Channel extension for message debouncing.
// Channels that support message merging can implement this interface.
type Debouncer interface {
	// GetDebounceKey returns a key used to group messages for debouncing.
	// Messages with the same key within the debounce window will be merged.
	// Typically returns sessionID (channel:chatID or channel:userID).
	GetDebounceKey(msg *IncomingMessage) string

	// MergeMessages merges multiple messages into one.
	// Called when multiple messages with the same debounce key arrive rapidly.
	MergeMessages(msgs []*IncomingMessage) *IncomingMessage

	// ShouldDebounce returns whether this message should be debounced.
	// Some message types (commands, system messages) may skip debouncing.
	ShouldDebounce(msg *IncomingMessage) bool
}

// MessageQueue manages a bounded queue of incoming messages with debouncing support.
type MessageQueue struct {
	queue  chan *queueItem
	config QueueConfig

	// debounce state
	debounceConfig  DebounceConfig
	pending         map[string][]*IncomingMessage // debounceKey -> buffered messages
	debounceTimers  map[string]*time.Timer        // debounceKey -> flush timer
	inProgress      map[string]bool               // debounceKey -> is processing
	pendingAfterRun map[string][]*IncomingMessage // messages arrived during processing
	mu              sync.Mutex

	handler func(ctx context.Context, chName string, msg IncomingMessage) error
	stopCh  chan struct{}
	doneCh  chan struct{}
	wg      sync.WaitGroup
}

// queueItem wraps a message with its channel reference.
type queueItem struct {
	channelName string
	msg         *IncomingMessage
	channel     Channel
}

// NewMessageQueue creates a new message queue with the given configuration.
func NewMessageQueue(queueCfg QueueConfig, debounceCfg DebounceConfig, handler func(ctx context.Context, chName string, msg IncomingMessage) error) *MessageQueue {
	if queueCfg.MaxSize <= 0 {
		queueCfg.MaxSize = DefaultQueueConfig().MaxSize
	}
	if queueCfg.Workers <= 0 {
		queueCfg.Workers = DefaultQueueConfig().Workers
	}
	if debounceCfg.DelayMs <= 0 {
		debounceCfg.DelayMs = DefaultDebounceConfig().DelayMs
	}
	if debounceCfg.MaxBufferSize <= 0 {
		debounceCfg.MaxBufferSize = DefaultDebounceConfig().MaxBufferSize
	}

	return &MessageQueue{
		queue:           make(chan *queueItem, queueCfg.MaxSize),
		config:          queueCfg,
		debounceConfig:  debounceCfg,
		pending:         make(map[string][]*IncomingMessage),
		debounceTimers:  make(map[string]*time.Timer),
		inProgress:      make(map[string]bool),
		pendingAfterRun: make(map[string][]*IncomingMessage),
		handler:         handler,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start starts the queue consumer workers.
func (q *MessageQueue) Start(ctx context.Context) {
	for i := 0; i < q.config.Workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i)
	}
	go func() {
		q.wg.Wait()
		close(q.doneCh)
	}()
}

// Stop gracefully stops the queue, waiting for pending messages to be processed.
func (q *MessageQueue) Stop(ctx context.Context) error {
	close(q.stopCh)

	// Cancel all pending debounce timers and flush immediately
	q.mu.Lock()
	for key, timer := range q.debounceTimers {
		timer.Stop()
		delete(q.debounceTimers, key)
	}
	// Flush any remaining pending messages
	for key, msgs := range q.pending {
		if len(msgs) > 0 {
			merged := q.mergeMessages(nil, msgs)
			if merged != nil {
				select {
				case q.queue <- &queueItem{channelName: msgs[0].Channel, msg: merged}:
				default:
					logger.L().Warn("queue full during shutdown, dropping message", zap.String("key", key))
				}
			}
		}
	}
	q.pending = make(map[string][]*IncomingMessage)
	q.mu.Unlock()

	// Wait for workers to finish
	select {
	case <-q.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Enqueue adds a message to the queue with optional debouncing.
func (q *MessageQueue) Enqueue(ch Channel, msg *IncomingMessage) bool {
	if msg == nil {
		return false
	}

	// Check if debouncing applies
	if q.debounceConfig.Enabled {
		debouncer, ok := ch.(Debouncer)
		if ok && debouncer.ShouldDebounce(msg) {
			return q.enqueueWithDebounce(ch, debouncer, msg)
		}
	}

	// Direct enqueue without debounce
	return q.enqueueDirectly(ch.Name(), msg, ch)
}

// enqueueWithDebounce handles debounced message enqueueing.
func (q *MessageQueue) enqueueWithDebounce(ch Channel, debouncer Debouncer, msg *IncomingMessage) bool {
	key := debouncer.GetDebounceKey(msg)

	q.mu.Lock()
	defer q.mu.Unlock()

	// If this session is currently being processed, buffer for later
	if q.inProgress[key] {
		q.pendingAfterRun[key] = append(q.pendingAfterRun[key], msg)
		return true
	}

	// Add to pending buffer
	q.pending[key] = append(q.pending[key], msg)

	// Cancel existing timer if any
	if timer, ok := q.debounceTimers[key]; ok {
		timer.Stop()
	}

	// Check if we should flush immediately (buffer full)
	if len(q.pending[key]) >= q.debounceConfig.MaxBufferSize {
		q.flushPendingLocked(ch, debouncer, key)
		return true
	}

	// Set new debounce timer
	delay := time.Duration(q.debounceConfig.DelayMs) * time.Millisecond
	q.debounceTimers[key] = time.AfterFunc(delay, func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.flushPendingLocked(ch, debouncer, key)
	})

	return true
}

// flushPendingLocked flushes pending messages for a key. Must be called with lock held.
func (q *MessageQueue) flushPendingLocked(ch Channel, debouncer Debouncer, key string) {
	delete(q.debounceTimers, key)

	msgs := q.pending[key]
	if len(msgs) == 0 {
		return
	}
	delete(q.pending, key)

	// Merge messages
	merged := debouncer.MergeMessages(msgs)
	if merged == nil {
		return
	}

	// Mark as in progress
	q.inProgress[key] = true

	// Try to enqueue
	select {
	case q.queue <- &queueItem{channelName: ch.Name(), msg: merged, channel: ch}:
		// Success
	default:
		// Queue full, try to drop oldest and retry
		logger.L().Warn("message queue full, dropping oldest message",
			zap.String("channel", ch.Name()),
			zap.String("key", key))
		select {
		case <-q.queue:
			select {
			case q.queue <- &queueItem{channelName: ch.Name(), msg: merged, channel: ch}:
			default:
			}
		default:
		}
	}
}

// enqueueDirectly adds a message to the queue without debouncing.
func (q *MessageQueue) enqueueDirectly(chName string, msg *IncomingMessage, ch Channel) bool {
	select {
	case q.queue <- &queueItem{channelName: chName, msg: msg, channel: ch}:
		return true
	default:
		logger.L().Warn("message queue full, dropping message",
			zap.String("channel", chName),
			zap.String("chat_id", msg.ChatID))
		return false
	}
}

// worker processes messages from the queue.
func (q *MessageQueue) worker(ctx context.Context, id int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.stopCh:
			// Drain remaining messages
			for {
				select {
				case item := <-q.queue:
					q.processItem(ctx, item)
				default:
					return
				}
			}
		case <-ctx.Done():
			return
		case item := <-q.queue:
			q.processItem(ctx, item)
		}
	}
}

// processItem handles a single queue item.
func (q *MessageQueue) processItem(ctx context.Context, item *queueItem) {
	if item == nil || item.msg == nil {
		return
	}

	// Get debounce key for progress tracking
	var key string
	if debouncer, ok := item.channel.(Debouncer); ok && q.debounceConfig.Enabled {
		key = debouncer.GetDebounceKey(item.msg)
	}

	// Process message
	if q.handler != nil {
		if err := q.handler(ctx, item.channelName, *item.msg); err != nil {
			logger.L().Warn("message handler error",
				zap.String("channel", item.channelName),
				zap.Error(err))
		}
	}

	// Handle post-processing for debounced messages
	if key != "" {
		q.handlePostProcessing(item.channel, key)
	}
}

// handlePostProcessing handles messages that arrived during processing.
func (q *MessageQueue) handlePostProcessing(ch Channel, key string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.inProgress, key)

	// Check if there are pending messages that arrived during processing
	pending := q.pendingAfterRun[key]
	if len(pending) == 0 {
		return
	}
	delete(q.pendingAfterRun, key)

	// Merge and re-enqueue
	debouncer, ok := ch.(Debouncer)
	if !ok {
		return
	}

	merged := debouncer.MergeMessages(pending)
	if merged == nil {
		return
	}

	select {
	case q.queue <- &queueItem{channelName: ch.Name(), msg: merged, channel: ch}:
		q.inProgress[key] = true
	default:
		// Queue full, add back to pending for next cycle
		q.pending[key] = pending
	}
}

// mergeMessages is a fallback merge when no Debouncer is available.
func (q *MessageQueue) mergeMessages(ch Channel, msgs []*IncomingMessage) *IncomingMessage {
	if len(msgs) == 0 {
		return nil
	}
	if len(msgs) == 1 {
		return msgs[0]
	}

	// Try to use channel's debouncer
	if debouncer, ok := ch.(Debouncer); ok {
		return debouncer.MergeMessages(msgs)
	}

	// Fallback: concatenate content
	merged := *msgs[0]
	for i := 1; i < len(msgs); i++ {
		merged.Content += "\n" + msgs[i].Content
	}
	return &merged
}

// Stats returns queue statistics.
func (q *MessageQueue) Stats() QueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()

	pendingCount := 0
	for _, msgs := range q.pending {
		pendingCount += len(msgs)
	}
	pendingAfterCount := 0
	for _, msgs := range q.pendingAfterRun {
		pendingAfterCount += len(msgs)
	}

	return QueueStats{
		QueueLength:     len(q.queue),
		QueueCapacity:   cap(q.queue),
		PendingDebounce: pendingCount,
		PendingAfterRun: pendingAfterCount,
		InProgress:      len(q.inProgress),
		ActiveTimers:    len(q.debounceTimers),
	}
}

// QueueStats contains queue statistics.
type QueueStats struct {
	QueueLength     int // Current queue length
	QueueCapacity   int // Queue capacity
	PendingDebounce int // Messages waiting for debounce flush
	PendingAfterRun int // Messages arrived during processing
	InProgress      int // Sessions currently being processed
	ActiveTimers    int // Active debounce timers
}

// DefaultDebouncer provides a default Debouncer implementation that can be embedded.
type DefaultDebouncer struct{}

// GetDebounceKey returns the session ID as the debounce key.
func (d *DefaultDebouncer) GetDebounceKey(msg *IncomingMessage) string {
	if msg.ChatID != "" {
		return msg.Channel + ":" + msg.ChatID
	}
	return msg.Channel + ":" + msg.UserID
}

// MergeMessages concatenates message contents.
func (d *DefaultDebouncer) MergeMessages(msgs []*IncomingMessage) *IncomingMessage {
	if len(msgs) == 0 {
		return nil
	}
	if len(msgs) == 1 {
		return msgs[0]
	}

	// Use first message as base, concatenate content
	merged := *msgs[0]
	for i := 1; i < len(msgs); i++ {
		merged.Content += "\n" + msgs[i].Content
	}
	merged.Timestamp = msgs[len(msgs)-1].Timestamp // Use latest timestamp
	return &merged
}

// ShouldDebounce returns true for regular text messages.
func (d *DefaultDebouncer) ShouldDebounce(msg *IncomingMessage) bool {
	// Skip debounce for command messages (starting with /)
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		return false
	}
	return true
}
