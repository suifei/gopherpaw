// Package memory provides async compaction queue for non-blocking memory compression.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// CompactionTask represents a pending compaction job.
type CompactionTask struct {
	ChatID          string
	Messages        []agent.Message
	PreviousSummary string
	ResultChan      chan CompactionResult
}

// CompactionResult contains the outcome of a compaction task.
type CompactionResult struct {
	ChatID   string
	Summary  string
	Messages []agent.Message
	Err      error
}

// AsyncCompactionQueue manages background compaction tasks.
type AsyncCompactionQueue struct {
	compactor   *Compactor
	tasks       chan CompactionTask
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	workerCount int
	queueSize   int
	mu          sync.RWMutex
	pending     map[string]struct{} // tracks pending tasks by chatID to avoid duplicates
}

// AsyncCompactionConfig configures the async compaction queue.
type AsyncCompactionConfig struct {
	WorkerCount int // Number of concurrent workers (default: 2)
	QueueSize   int // Max pending tasks (default: 100)
}

// NewAsyncCompactionQueue creates a new async compaction queue.
func NewAsyncCompactionQueue(compactor *Compactor, cfg AsyncCompactionConfig) *AsyncCompactionQueue {
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = 2
	}
	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := &AsyncCompactionQueue{
		compactor:   compactor,
		tasks:       make(chan CompactionTask, queueSize),
		ctx:         ctx,
		cancel:      cancel,
		workerCount: workerCount,
		queueSize:   queueSize,
		pending:     make(map[string]struct{}),
	}

	return q
}

// Start begins the worker goroutines.
func (q *AsyncCompactionQueue) Start() {
	log := logger.L()
	log.Info("Starting async compaction queue",
		zap.Int("workers", q.workerCount),
		zap.Int("queueSize", q.queueSize),
	)

	for i := 0; i < q.workerCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// Stop gracefully shuts down the queue and waits for pending tasks.
func (q *AsyncCompactionQueue) Stop() {
	log := logger.L()
	log.Info("Stopping async compaction queue")

	q.cancel()
	close(q.tasks)
	q.wg.Wait()

	log.Info("Async compaction queue stopped")
}

// Submit queues a compaction task. Non-blocking; returns false if queue is full or chat already pending.
func (q *AsyncCompactionQueue) Submit(chatID string, messages []agent.Message, previousSummary string) (chan CompactionResult, bool) {
	q.mu.Lock()
	// Check if already pending for this chatID
	if _, exists := q.pending[chatID]; exists {
		q.mu.Unlock()
		logger.L().Debug("Compaction already pending for chat", zap.String("chatID", chatID))
		return nil, false
	}
	q.pending[chatID] = struct{}{}
	q.mu.Unlock()

	resultChan := make(chan CompactionResult, 1)
	task := CompactionTask{
		ChatID:          chatID,
		Messages:        messages,
		PreviousSummary: previousSummary,
		ResultChan:      resultChan,
	}

	select {
	case q.tasks <- task:
		logger.L().Debug("Compaction task submitted", zap.String("chatID", chatID))
		return resultChan, true
	default:
		// Queue full
		q.mu.Lock()
		delete(q.pending, chatID)
		q.mu.Unlock()
		close(resultChan)
		logger.L().Warn("Compaction queue full, task dropped", zap.String("chatID", chatID))
		return nil, false
	}
}

// SubmitAndWait submits a task and waits for the result with a timeout.
func (q *AsyncCompactionQueue) SubmitAndWait(ctx context.Context, chatID string, messages []agent.Message, previousSummary string) (*CompactionResult, error) {
	resultChan, ok := q.Submit(chatID, messages, previousSummary)
	if !ok {
		return nil, nil // Already pending or queue full
	}

	select {
	case result := <-resultChan:
		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PendingCount returns the number of tasks currently pending.
func (q *AsyncCompactionQueue) PendingCount() int {
	return len(q.tasks)
}

// IsPending checks if a chat has a pending compaction task.
func (q *AsyncCompactionQueue) IsPending(chatID string) bool {
	q.mu.RLock()
	_, exists := q.pending[chatID]
	q.mu.RUnlock()
	return exists
}

func (q *AsyncCompactionQueue) worker(id int) {
	defer q.wg.Done()
	log := logger.L()

	for {
		select {
		case <-q.ctx.Done():
			return
		case task, ok := <-q.tasks:
			if !ok {
				return
			}
			q.processTask(log, id, task)
		}
	}
}

func (q *AsyncCompactionQueue) processTask(log *zap.Logger, workerID int, task CompactionTask) {
	start := time.Now()
	log.Debug("Processing compaction task",
		zap.Int("worker", workerID),
		zap.String("chatID", task.ChatID),
		zap.Int("messageCount", len(task.Messages)),
	)

	// Create a timeout context for the LLM call
	ctx, cancel := context.WithTimeout(q.ctx, 60*time.Second)
	defer cancel()

	summary, messages, err := q.compactor.CompactWithLLM(ctx, task.Messages, task.PreviousSummary)

	// Remove from pending set
	q.mu.Lock()
	delete(q.pending, task.ChatID)
	q.mu.Unlock()

	result := CompactionResult{
		ChatID:   task.ChatID,
		Summary:  summary,
		Messages: messages,
		Err:      err,
	}

	elapsed := time.Since(start)
	if err != nil {
		log.Warn("Compaction task failed",
			zap.Int("worker", workerID),
			zap.String("chatID", task.ChatID),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	} else {
		log.Debug("Compaction task completed",
			zap.Int("worker", workerID),
			zap.String("chatID", task.ChatID),
			zap.Duration("elapsed", elapsed),
			zap.Int("summaryLen", len(summary)),
			zap.Int("keptMessages", len(messages)),
		)
	}

	// Send result (non-blocking since channel has buffer of 1)
	select {
	case task.ResultChan <- result:
	default:
	}
	close(task.ResultChan)
}
