package channels

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockAgentForConsole struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgentForConsole) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "mock response", nil
}

func (m *mockAgentForConsole) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, chatID, text)
	}
	ch := make(chan string, 1)
	ch <- "mock stream response"
	close(ch)
	return ch, nil
}

func TestNewConsole(t *testing.T) {
	c := NewConsole(&mockAgentForConsole{}, true, nil)
	if c.Name() != "console" {
		t.Errorf("Name() = %v, want console", c.Name())
	}
	if !c.IsEnabled() {
		t.Errorf("IsEnabled() = false, want true")
	}
}

func TestConsoleDisabled(t *testing.T) {
	c := NewConsole(&mockAgentForConsole{}, false, nil)
	if c.IsEnabled() {
		t.Errorf("IsEnabled() = true, want false")
	}
}

func TestConsoleWithNilAgent(t *testing.T) {
	// This should not panic but handle gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred with nil agent: %v", r)
		}
	}()

	c := NewConsole(nil, true, nil)
	if c == nil {
		t.Error("NewConsole with nil agent should return a non-nil channel")
	}
	if c.Name() != "console" {
		t.Errorf("Name() = %v, want console", c.Name())
	}
}

func TestTruncateForConsole(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"empty", "", 10, ""},
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world this is long", 10, "hello worl..."},
		{"spaces", "   hello   ", 10, "hello"},
		{"zero_len", "hello", 0, "hello"},
		{"negative", "hello", -1, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForConsole(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// captureStdout safely captures stdout during test execution
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	// Atomically replace stdout
	os.Stdout = w

	// Make sure to restore stdout and capture output
	defer func() {
		os.Stdout = oldStdout
		w.Close()
	}()

	// Execute the test function
	fn()

	// Read all output
	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	r.Close()

	return buf.String()
}

// captureStderr safely captures stderr during test execution
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	// Atomically replace stderr
	os.Stderr = w

	// Make sure to restore stderr and capture output
	defer func() {
		os.Stderr = oldStderr
		w.Close()
	}()

	// Execute the test function
	fn()

	// Read all output
	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	r.Close()

	return buf.String()
}

func TestConsoleSend(t *testing.T) {
	var output string
	output = captureStdout(t, func() {
		c := NewConsole(&mockAgentForConsole{}, true, nil)
		err := c.Send(context.Background(), "user", "Hello", nil)
		if err != nil {
			t.Errorf("Send error = %v", err)
		}
	})

	if !strings.Contains(output, "Hello") {
		t.Errorf("output missing text")
	}
}

func TestConsoleSendEmpty(t *testing.T) {
	captureStdout(t, func() {
		c := NewConsole(&mockAgentForConsole{}, true, nil)
		err := c.Send(context.Background(), "user", "", nil)
		if err != nil {
			t.Errorf("Send error = %v", err)
		}
	})
}

func TestConsoleSendFile(t *testing.T) {
	output := captureStdout(t, func() {
		c := NewConsole(&mockAgentForConsole{}, true, nil)
		err := c.SendFile(context.Background(), "user", "/tmp/file.pdf", "application/pdf", nil)
		if err != nil {
			t.Errorf("SendFile error = %v", err)
		}
	})

	if !strings.Contains(output, "file.pdf") {
		t.Errorf("output missing filename")
	}
}

func TestProgressReporter(t *testing.T) {
	output := captureStderr(t, func() {
		reporter := &consoleProgressReporter{}
		reporter.OnThinking()
	})

	if !strings.Contains(output, "Thinking") {
		t.Errorf("missing Thinking output")
	}
}

func TestProgressReporterToolCall(t *testing.T) {
	output := captureStderr(t, func() {
		reporter := &consoleProgressReporter{}
		reporter.OnToolCall("read_file", `{"path":"/tmp"}`)
	})

	if !strings.Contains(output, "read_file") {
		t.Errorf("missing tool name")
	}
}

func TestProgressReporterToolResult(t *testing.T) {
	output := captureStderr(t, func() {
		reporter := &consoleProgressReporter{}
		reporter.OnToolResult("read_file", "content")
	})

	if !strings.Contains(output, "read_file") {
		t.Errorf("missing tool name in result")
	}
}

func TestProgressReporterFinalReply(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := &consoleProgressReporter{}
	reporter.OnFinalReply("reply")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	output := buf.String()
	if !strings.Contains(output, "───") {
		t.Errorf("missing separator in output: %q", output)
	}
}

func TestConsoleConcurrentSend(t *testing.T) {
	oldStdout := os.Stdout
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Failed to open /dev/null: %v", err)
	}
	defer func() {
		os.Stdout = oldStdout
		devNull.Close()
	}()
	os.Stdout = devNull

	c := NewConsole(&mockAgentForConsole{}, true, nil)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			text := fmt.Sprintf("msg %d", idx)
			c.Send(context.Background(), "user", text, nil)
		}(i)
	}
	wg.Wait()
}

func TestConsoleStopWithoutStart(t *testing.T) {
	c := NewConsole(&mockAgentForConsole{}, true, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("Stop without Start error = %v", err)
	}
}

// TestConsoleInitialTasks 测试初始任务功能
func TestConsoleInitialTasks(t *testing.T) {
	taskExecuted := false
	taskContent := ""

	mockAg := &mockAgentForConsole{
		runFunc: func(ctx context.Context, chatID, text string) (string, error) {
			taskExecuted = true
			taskContent = text
			return "response for " + text, nil
		},
	}

	c := NewConsole(mockAg, true, nil)
	c.SetInitialTasks([]string{"hello world"})
	c.SetRunOnce(true)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 启动 channel（会在 goroutine 中执行初始任务）
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start error = %v", err)
	}

	// 等待执行完成
	time.Sleep(500 * time.Millisecond)

	// 停止 channel
	if err := c.Stop(ctx); err != nil {
		t.Fatalf("Stop error = %v", err)
	}

	if !taskExecuted {
		t.Error("Initial task was not executed")
	}
	if taskContent != "hello world" {
		t.Errorf("Got task content %q, want %q", taskContent, "hello world")
	}
}

// TestConsoleSetInitialTasks 测试设置初始任务
func TestConsoleSetInitialTasks(t *testing.T) {
	c := NewConsole(&mockAgentForConsole{}, true, nil)

	tasks := []string{"task1", "task2"}
	c.SetInitialTasks(tasks)

	// 验证任务已设置（通过内部检查）
	c.mu.Lock()
	if len(c.initialTasks) != 2 {
		t.Errorf("Got %d tasks, want 2", len(c.initialTasks))
	}
	if c.initialTasks[0] != "task1" || c.initialTasks[1] != "task2" {
		t.Errorf("Tasks not set correctly: %v", c.initialTasks)
	}
	c.mu.Unlock()
}

// TestConsoleSetRunOnce 测试设置 runOnce 标志
func TestConsoleSetRunOnce(t *testing.T) {
	c := NewConsole(&mockAgentForConsole{}, true, nil)

	c.SetRunOnce(true)
	c.mu.Lock()
	if !c.runOnce {
		t.Error("runOnce not set to true")
	}
	c.mu.Unlock()

	c.SetRunOnce(false)
	c.mu.Lock()
	if c.runOnce {
		t.Error("runOnce not set to false")
	}
	c.mu.Unlock()
}

var _ Channel = (*ConsoleChannel)(nil)
var _ FileSender = (*ConsoleChannel)(nil)
