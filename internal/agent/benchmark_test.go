package agent

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

// mockLLMBench implements LLMProvider for benchmarking
type mockLLMBench struct {
	response string
}

func (m *mockLLMBench) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: m.response}, nil
}

func (m *mockLLMBench) ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return &mockBenchStream{response: m.response}, nil
}

func (m *mockLLMBench) Name() string { return "mock-bench" }

type mockBenchStream struct {
	response string
	sent     bool
}

func (s *mockBenchStream) Recv() (*ChatChunk, error) {
	if s.sent {
		return nil, io.EOF
	}
	s.sent = true
	return &ChatChunk{Content: s.response}, nil
}

func (s *mockBenchStream) Close() error { return nil }

// mockMemBench implements MemoryStore for benchmarking
type mockMemBench struct{}

func (m *mockMemBench) Save(ctx context.Context, chatID string, msg Message) error {
	return nil
}

func (m *mockMemBench) Load(ctx context.Context, chatID string, limit int) ([]Message, error) {
	return nil, nil
}

func (m *mockMemBench) Search(ctx context.Context, chatID string, query string, topK int) ([]MemoryResult, error) {
	return nil, nil
}

func (m *mockMemBench) Compact(ctx context.Context, chatID string) error {
	return nil
}

func (m *mockMemBench) GetCompactSummary(ctx context.Context, chatID string) (string, error) {
	return "", nil
}

func (m *mockMemBench) SaveLongTerm(ctx context.Context, chatID string, content string, category string) error {
	return nil
}

func (m *mockMemBench) LoadLongTerm(ctx context.Context, chatID string) (string, error) {
	return "", nil
}

// BenchmarkReactAgent_Run benchmarks the ReactAgent Run method
func BenchmarkReactAgent_Run(b *testing.B) {
	mockLLM := &mockLLMBench{response: "Test response"}
	mockMemory := &mockMemBench{}
	tools := []Tool{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns: 10,
		},
	}

	ag := NewReact(mockLLM, mockMemory, tools, cfg)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chatID := fmt.Sprintf("bench-chat-%d", i)
		_, _ = ag.Run(ctx, chatID, "test message")
	}
}

// BenchmarkReactAgent_Parallel benchmarks parallel execution
func BenchmarkReactAgent_Parallel(b *testing.B) {
	mockLLM := &mockLLMBench{response: "Test response"}
	mockMemory := &mockMemBench{}
	tools := []Tool{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns: 10,
		},
	}

	ag := NewReact(mockLLM, mockMemory, tools, cfg)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			chatID := fmt.Sprintf("bench-chat-%d", i)
			_, _ = ag.Run(ctx, chatID, "test message")
			i++
		}
	})
}

// TestMemoryUsage tests memory usage stays within limits
func TestMemoryUsage(t *testing.T) {
	// Skip in race detection mode as memory stats can be unreliable
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	mockLLM := &mockLLMBench{response: "Test response"}
	mockMemory := &mockMemBench{}
	tools := []Tool{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns: 10,
		},
	}

	ag := NewReact(mockLLM, mockMemory, tools, cfg)
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		chatID := fmt.Sprintf("memory-test-%d", i)
		_, _ = ag.Run(ctx, chatID, "test message")
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	usedMB := int64(m2.Alloc) - int64(m1.Alloc)
	if usedMB < 0 {
		usedMB = 0
	}
	usedMB = usedMB / 1024 / 1024

	t.Logf("Memory used: %d MB", usedMB)

	if usedMB > 500 {
		t.Errorf("Memory usage too high: %d MB (limit: 500 MB)", usedMB)
	}
}

// TestResponseTime tests response time for simple queries
func TestResponseTime(t *testing.T) {
	mockLLM := &mockLLMBench{response: "Test response"}
	mockMemory := &mockMemBench{}
	tools := []Tool{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns: 10,
		},
	}

	ag := NewReact(mockLLM, mockMemory, tools, cfg)
	ctx := context.Background()

	iterations := 10
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		chatID := fmt.Sprintf("response-test-%d", i)
		start := time.Now()
		_, err := ag.Run(ctx, chatID, "test message")
		duration := time.Since(start)
		totalDuration += duration

		if err != nil {
			t.Errorf("Run failed: %v", err)
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average response time: %v", avgDuration)

	if avgDuration > 5*time.Second {
		t.Errorf("Response time too slow: %v (limit: 5s)", avgDuration)
	}
}

// TestConcurrentSessions tests concurrent session handling
func TestConcurrentSessions(t *testing.T) {
	mockLLM := &mockLLMBench{response: "Test response"}
	mockMemory := &mockMemBench{}
	tools := []Tool{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns: 10,
		},
	}

	ag := NewReact(mockLLM, mockMemory, tools, cfg)
	ctx := context.Background()

	numGoroutines := 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			chatID := fmt.Sprintf("concurrent-test-%d", id)
			_, err := ag.Run(ctx, chatID, "test message")
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent run failed: %v", err)
	}

	t.Logf("Successfully handled %d concurrent sessions", numGoroutines)
}

// TestMemoryLeak tests for memory leaks
func TestMemoryLeak(t *testing.T) {
	mockLLM := &mockLLMBench{response: "Test response"}
	mockMemory := &mockMemBench{}
	tools := []Tool{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns: 10,
		},
	}

	ag := NewReact(mockLLM, mockMemory, tools, cfg)
	ctx := context.Background()

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 1000; i++ {
		chatID := fmt.Sprintf("leak-test-%d", i%10) // Reuse chat IDs
		_, _ = ag.Run(ctx, chatID, "test message")
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	leakedMB := int64(m2.Alloc) - int64(m1.Alloc)
	if leakedMB < 0 {
		leakedMB = 0
	}
	leakedMB = leakedMB / 1024 / 1024

	t.Logf("Potential memory leak: %d MB", leakedMB)

	// Allow some variance, but not excessive growth
	if leakedMB > 50 {
		t.Errorf("Possible memory leak: %d MB growth", leakedMB)
	}
}

// BenchmarkToolExecution benchmarks tool execution performance
func BenchmarkToolExecution(b *testing.B) {
	tool := &benchTool{name: "bench_tool", desc: "Benchmark tool"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(context.Background(), `{"arg":"value"}`)
	}
}

type benchTool struct {
	name string
	desc string
}

func (t *benchTool) Name() string        { return t.name }
func (t *benchTool) Description() string { return t.desc }
func (t *benchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"arg": map[string]any{"type": "string"},
		},
	}
}
func (t *benchTool) Execute(ctx context.Context, arguments string) (string, error) {
	return "mock result", nil
}
