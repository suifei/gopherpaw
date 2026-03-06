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

func TestConsoleSend(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := NewConsole(&mockAgentForConsole{}, true, nil)
	err := c.Send(context.Background(), "user", "Hello", nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Send error = %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if !strings.Contains(buf.String(), "Hello") {
		t.Errorf("output missing text")
	}
}

func TestConsoleSendEmpty(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := NewConsole(&mockAgentForConsole{}, true, nil)
	err := c.Send(context.Background(), "user", "", nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Send error = %v", err)
	}
	r.Close()
}

func TestConsoleSendFile(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := NewConsole(&mockAgentForConsole{}, true, nil)
	err := c.SendFile(context.Background(), "user", "/tmp/file.pdf", "application/pdf", nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("SendFile error = %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if !strings.Contains(buf.String(), "file.pdf") {
		t.Errorf("output missing filename")
	}
}

func TestProgressReporter(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := &consoleProgressReporter{}
	reporter.OnThinking()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if !strings.Contains(buf.String(), "Thinking") {
		t.Errorf("missing Thinking output")
	}
}

func TestProgressReporterToolCall(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := &consoleProgressReporter{}
	reporter.OnToolCall("read_file", `{"path":"/tmp"}`)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if !strings.Contains(buf.String(), "read_file") {
		t.Errorf("missing tool name")
	}
}

func TestProgressReporterToolResult(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := &consoleProgressReporter{}
	reporter.OnToolResult("read_file", "content")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if !strings.Contains(buf.String(), "read_file") {
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
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

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

var _ Channel = (*ConsoleChannel)(nil)
var _ FileSender = (*ConsoleChannel)(nil)
