package channels

import (
	"context"
	"strings"
	"testing"
)

func BenchmarkConsole_Send_Small(b *testing.B) {
	console := NewConsole(nil, true, nil)

	content := strings.Repeat("test line\n", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		console.Send(context.Background(), "user", content, nil)
	}
}

func BenchmarkConsole_Send_Medium(b *testing.B) {
	console := NewConsole(nil, true, nil)

	content := strings.Repeat("test line\n", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		console.Send(context.Background(), "user", content, nil)
	}
}

func BenchmarkConsole_Send_Large(b *testing.B) {
	console := NewConsole(nil, true, nil)

	content := strings.Repeat("test line\n", 100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		console.Send(context.Background(), "user", content, nil)
	}
}

func BenchmarkConsole_SendFile(b *testing.B) {
	console := NewConsole(nil, true, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		console.SendFile(context.Background(), "user", "/path/to/file.txt", "text/plain", nil)
	}
}

func BenchmarkTruncateForConsole_Short(b *testing.B) {
	text := "short text"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateForConsole(text, 100)
	}
}

func BenchmarkTruncateForConsole_Medium(b *testing.B) {
	text := strings.Repeat("test line ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateForConsole(text, 100)
	}
}

func BenchmarkTruncateForConsole_Long(b *testing.B) {
	text := strings.Repeat("test line ", 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateForConsole(text, 100)
	}
}

func BenchmarkTruncateForConsole_NoTruncate(b *testing.B) {
	text := strings.Repeat("test line ", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateForConsole(text, 500)
	}
}

func BenchmarkConsoleProgressReporter_OnThinking(b *testing.B) {
	reporter := &consoleProgressReporter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reporter.OnThinking()
	}
}

func BenchmarkConsoleProgressReporter_OnToolCall(b *testing.B) {
	reporter := &consoleProgressReporter{}
	args := strings.Repeat("arg ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reporter.OnToolCall("test_tool", args)
	}
}

func BenchmarkConsoleProgressReporter_OnToolResult(b *testing.B) {
	reporter := &consoleProgressReporter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reporter.OnToolResult("test_tool", "result")
	}
}

func BenchmarkConsoleProgressReporter_OnFinalReply(b *testing.B) {
	reporter := &consoleProgressReporter{}
	content := strings.Repeat("reply line\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reporter.OnFinalReply(content)
	}
}
