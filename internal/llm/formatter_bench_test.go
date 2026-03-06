package llm

import (
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
)

func BenchmarkFormatter_Simple(b *testing.B) {
	formatter := NewFileBlockFormatter()
	messages := []agent.Message{
		{Role: "user", Content: "Hello, how are you?"},
		{Role: "assistant", Content: "I'm doing well, thank you!"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(messages)
	}
}

func BenchmarkFormatter_MultipleMessages(b *testing.B) {
	formatter := NewFileBlockFormatter()
	messages := make([]agent.Message, 100)
	for i := 0; i < 100; i++ {
		messages[i] = agent.Message{
			Role:    "user",
			Content: "This is a test message with some content.",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(messages)
	}
}

func BenchmarkFormatter_WithToolMessages(b *testing.B) {
	formatter := NewFileBlockFormatter()
	messages := []agent.Message{
		{Role: "user", Content: "Read the file"},
		{Role: "assistant", ToolCalls: []agent.ToolCall{{ID: "1", Name: "read_file", Arguments: `{"file_path": "test.txt"}`}}},
		{Role: "tool", ToolCallID: "1", Content: "File content here"},
		{Role: "assistant", Content: "The file contains..."},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(messages)
	}
}

func BenchmarkFormatter_WithFileBlocks(b *testing.B) {
	formatter := NewFileBlockFormatter()
	messages := []agent.Message{
		{Role: "user", Content: "Send a file"},
		{Role: "assistant", ToolCalls: []agent.ToolCall{{ID: "1", Name: "send_file", Arguments: `{"file_path": "test.txt"}`}}},
		{Role: "tool", ToolCallID: "1", Content: `[{"type": "file", "file_path": "test.txt", "file_name": "test.txt", "mime_type": "text/plain"}]`},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(messages)
	}
}

func BenchmarkFormatter_LongContent(b *testing.B) {
	formatter := NewFileBlockFormatter()
	longContent := string(make([]byte, 10000))
	for i := range longContent {
		longContent = longContent[:i] + "a" + longContent[i+1:]
	}

	messages := []agent.Message{
		{Role: "user", Content: longContent},
		{Role: "assistant", Content: "Response"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(messages)
	}
}

func BenchmarkFormatFileBlock(b *testing.B) {
	block := fileBlock{
		Type:     "file",
		FilePath: "/path/to/file.txt",
		FileName: "file.txt",
		MimeType: "text/plain",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatFileBlock(block)
	}
}

func BenchmarkFormatFileBlock_Multiple(b *testing.B) {
	blocks := []fileBlock{
		{Type: "file", FilePath: "/path/to/file1.txt", FileName: "file1.txt", MimeType: "text/plain"},
		{Type: "file", FilePath: "/path/to/file2.json", FileName: "file2.json", MimeType: "application/json"},
		{Type: "file", FilePath: "/path/to/file3.md", FileName: "file3.md", MimeType: "text/markdown"},
		{Type: "file", URL: "https://example.com/file.pdf", MimeType: "application/pdf"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, block := range blocks {
			formatFileBlock(block)
		}
	}
}

func BenchmarkProcessToolContent_Simple(b *testing.B) {
	content := "This is a simple tool result"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processToolContent(content)
	}
}

func BenchmarkProcessToolContent_WithFileBlock(b *testing.B) {
	content := `[{"type": "file", "file_path": "test.txt", "file_name": "test.txt", "mime_type": "text/plain"}]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processToolContent(content)
	}
}

func BenchmarkProcessToolContent_WithTextBlocks(b *testing.B) {
	content := `[{"type": "text", "text": "First block"}, {"type": "text", "text": "Second block"}]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processToolContent(content)
	}
}

func BenchmarkProcessToolContent_Mixed(b *testing.B) {
	content := `[{"type": "text", "text": "Text before"}, {"type": "file", "file_path": "test.txt", "file_name": "test.txt", "mime_type": "text/plain"}, {"type": "text", "text": "Text after"}]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processToolContent(content)
	}
}
