package llm

import (
	"strings"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
)

func TestFileBlockSupportFormatter_NoChange(t *testing.T) {
	f := NewFileBlockFormatter()
	msgs := []agent.Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	result := f.Format(msgs)
	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}
	if result[1].Content != "hello" {
		t.Errorf("user message changed: %q", result[1].Content)
	}
}

func TestFileBlockSupportFormatter_FileBlock(t *testing.T) {
	f := NewFileBlockFormatter()
	msgs := []agent.Message{
		{Role: "assistant", ToolCalls: []agent.ToolCall{{ID: "tc1", Name: "send_file"}}},
		{Role: "tool", ToolCallID: "tc1", Content: `{"type":"file","file_path":"/tmp/test.png","mime_type":"image/png"}`},
	}
	result := f.Format(msgs)
	var toolMsg *agent.Message
	for i := range result {
		if result[i].Role == "tool" {
			toolMsg = &result[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("expected tool message in result")
	}
	if !strings.Contains(toolMsg.Content, "[File:") {
		t.Errorf("expected file block converted to text, got %q", toolMsg.Content)
	}
	if !strings.Contains(toolMsg.Content, "image/png") {
		t.Errorf("expected mime type in output, got %q", toolMsg.Content)
	}
}

func TestFileBlockSupportFormatter_ContentArray(t *testing.T) {
	f := NewFileBlockFormatter()
	msgs := []agent.Message{
		{Role: "assistant", ToolCalls: []agent.ToolCall{{ID: "tc1", Name: "multi_tool"}}},
		{Role: "tool", ToolCallID: "tc1", Content: `[{"type":"text","text":"hello"},{"type":"file","file_name":"doc.pdf","mime_type":"application/pdf"}]`},
	}
	result := f.Format(msgs)
	var toolMsg *agent.Message
	for i := range result {
		if result[i].Role == "tool" {
			toolMsg = &result[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("expected tool message in result")
	}
	if !strings.Contains(toolMsg.Content, "hello") {
		t.Errorf("expected text block preserved, got %q", toolMsg.Content)
	}
	if !strings.Contains(toolMsg.Content, "[File:") {
		t.Errorf("expected file block converted, got %q", toolMsg.Content)
	}
}

func TestFileBlockSupportFormatter_PlainText(t *testing.T) {
	f := NewFileBlockFormatter()
	msgs := []agent.Message{
		{Role: "assistant", ToolCalls: []agent.ToolCall{{ID: "tc1", Name: "test"}}},
		{Role: "tool", ToolCallID: "tc1", Content: "just plain text result"},
	}
	result := f.Format(msgs)
	var toolMsg *agent.Message
	for i := range result {
		if result[i].Role == "tool" {
			toolMsg = &result[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("expected tool message in result")
	}
	if toolMsg.Content != "just plain text result" {
		t.Errorf("plain text should be unchanged, got %q", toolMsg.Content)
	}
}

func TestFileBlockSupportFormatter_SanitizesToolMessages(t *testing.T) {
	f := NewFileBlockFormatter()
	msgs := []agent.Message{
		{Role: "assistant", ToolCalls: []agent.ToolCall{
			{ID: "tc1", Name: "test"},
			{ID: "", Name: ""},
		}},
		{Role: "tool", ToolCallID: "tc1", Content: "ok"},
		{Role: "tool", ToolCallID: "", Content: "orphan"},
	}
	result := f.Format(msgs)
	for _, m := range result {
		if m.Role == "tool" && m.ToolCallID == "" {
			t.Error("should have removed tool with empty ToolCallID")
		}
	}
}

func TestNewFileBlockFormatter(t *testing.T) {
	f := NewFileBlockFormatter()
	if f == nil {
		t.Fatal("expected non-nil formatter")
	}
	if !f.StripMessageName {
		t.Error("expected StripMessageName to be true by default")
	}
}
