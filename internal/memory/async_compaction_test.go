package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestAsyncCompactionQueue_StartStop(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 2,
		QueueSize:   10,
	})

	queue.Start()
	time.Sleep(10 * time.Millisecond) // Let workers start
	queue.Stop()
	// Should not hang or panic
}

func TestAsyncCompactionQueue_Submit(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 1,
		QueueSize:   10,
	})

	queue.Start()
	defer queue.Stop()

	msgs := []agent.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "how are you"},
		{Role: "assistant", Content: "I'm fine"},
	}

	resultChan, ok := queue.Submit("chat1", msgs, "")
	if !ok {
		t.Fatal("Submit should succeed")
	}

	select {
	case result := <-resultChan:
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.ChatID != "chat1" {
			t.Errorf("ChatID = %q, want chat1", result.ChatID)
		}
		if len(result.Messages) != 2 {
			t.Errorf("Messages count = %d, want 2", len(result.Messages))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestAsyncCompactionQueue_DuplicatePrevention(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 1,
		QueueSize:   10,
	})

	queue.Start()
	defer queue.Stop()

	msgs := []agent.Message{{Role: "user", Content: "test"}}

	// First submit should succeed
	_, ok1 := queue.Submit("chat1", msgs, "")
	if !ok1 {
		t.Fatal("First submit should succeed")
	}

	// Second submit for same chat should fail (duplicate)
	_, ok2 := queue.Submit("chat1", msgs, "")
	if ok2 {
		t.Error("Duplicate submit should fail")
	}

	// Wait for first task to complete
	time.Sleep(100 * time.Millisecond)

	// Now should be able to submit again
	_, ok3 := queue.Submit("chat1", msgs, "")
	if !ok3 {
		t.Error("Submit after completion should succeed")
	}
}

func TestAsyncCompactionQueue_QueueFull(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 0, // No workers - tasks will pile up
		QueueSize:   2,
	})

	// Don't start - tasks will accumulate
	msgs := []agent.Message{{Role: "user", Content: "test"}}

	// Fill queue
	queue.Submit("chat1", msgs, "")
	queue.Submit("chat2", msgs, "")

	// Queue should be full
	_, ok := queue.Submit("chat3", msgs, "")
	if ok {
		t.Error("Submit to full queue should fail")
	}
}

func TestAsyncCompactionQueue_Concurrent(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 4,
		QueueSize:   100,
	})

	queue.Start()
	defer queue.Stop()

	var completed int32
	var wg sync.WaitGroup
	msgs := []agent.Message{{Role: "user", Content: "test"}}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			chatID := string(rune('a' + id))
			resultChan, ok := queue.Submit(chatID, msgs, "")
			if !ok {
				return
			}
			select {
			case result := <-resultChan:
				if result.Err == nil {
					atomic.AddInt32(&completed, 1)
				}
			case <-time.After(5 * time.Second):
			}
		}(i)
	}

	wg.Wait()

	if completed < 15 {
		t.Errorf("Expected at least 15 completions, got %d", completed)
	}
}

func TestAsyncCompactionQueue_SubmitAndWait(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 1,
		QueueSize:   10,
	})

	queue.Start()
	defer queue.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgs := []agent.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	result, err := queue.SubmitAndWait(ctx, "chat1", msgs, "")
	if err != nil {
		t.Fatalf("SubmitAndWait error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Err != nil {
		t.Errorf("result error: %v", result.Err)
	}
}

func TestAsyncCompactionQueue_IsPending(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 0, // No workers
		QueueSize:   10,
	})

	msgs := []agent.Message{{Role: "user", Content: "test"}}

	if queue.IsPending("chat1") {
		t.Error("chat1 should not be pending initially")
	}

	queue.Submit("chat1", msgs, "")

	if !queue.IsPending("chat1") {
		t.Error("chat1 should be pending after submit")
	}

	if queue.IsPending("chat2") {
		t.Error("chat2 should not be pending")
	}
}

func TestAsyncCompactionQueue_PendingCount(t *testing.T) {
	compactor := NewCompactor(nil, config.MemoryConfig{CompactKeepRecent: 2})
	queue := NewAsyncCompactionQueue(compactor, AsyncCompactionConfig{
		WorkerCount: 0, // No workers
		QueueSize:   10,
	})

	msgs := []agent.Message{{Role: "user", Content: "test"}}

	if queue.PendingCount() != 0 {
		t.Errorf("PendingCount = %d, want 0", queue.PendingCount())
	}

	queue.Submit("chat1", msgs, "")
	queue.Submit("chat2", msgs, "")

	if queue.PendingCount() != 2 {
		t.Errorf("PendingCount = %d, want 2", queue.PendingCount())
	}
}
