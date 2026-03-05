package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
)

func jsonFileArg(path string) string {
	b, _ := json.Marshal(map[string]string{"file_path": path})
	return string(b)
}

func TestSendFileTool_TextFile(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "hello.txt")
	os.WriteFile(fp, []byte("hello world"), 0644)

	tool := &SendFileTool{}
	if tool.Name() != "send_file_to_user" {
		t.Errorf("Name: got %s", tool.Name())
	}

	result, err := tool.ExecuteRich(context.Background(), jsonFileArg(fp))
	if err != nil {
		t.Fatalf("ExecuteRich: %v", err)
	}
	if !strings.Contains(result.Text, "hello world") {
		t.Errorf("expected text content, got %q", result.Text)
	}
	if len(result.Attachments) != 0 {
		t.Errorf("text file should have no attachments, got %d", len(result.Attachments))
	}
}

func TestSendFileTool_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.png")
	os.WriteFile(fp, []byte{0x89, 0x50, 0x4E, 0x47}, 0644) // PNG magic bytes

	tool := &SendFileTool{}
	result, err := tool.ExecuteRich(context.Background(), jsonFileArg(fp))
	if err != nil {
		t.Fatalf("ExecuteRich: %v", err)
	}
	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}
	att := result.Attachments[0]
	if att.MimeType != "image/png" {
		t.Errorf("expected image/png, got %s", att.MimeType)
	}
	if att.FileName != "test.png" {
		t.Errorf("expected test.png, got %s", att.FileName)
	}
}

func TestSendFileTool_MissingFile(t *testing.T) {
	tool := &SendFileTool{}
	_, err := tool.ExecuteRich(context.Background(), jsonFileArg("/nonexistent/file.txt"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSendFileTool_Directory(t *testing.T) {
	dir := t.TempDir()
	tool := &SendFileTool{}
	_, err := tool.ExecuteRich(context.Background(), jsonFileArg(dir))
	if err == nil {
		t.Error("expected error for directory")
	}
}

func TestSendFileTool_EmptyPath(t *testing.T) {
	tool := &SendFileTool{}
	_, err := tool.ExecuteRich(context.Background(), `{"file_path":""}`)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestSendFileTool_Execute(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.md")
	os.WriteFile(fp, []byte("# Hello"), 0644)

	tool := &SendFileTool{}
	out, err := tool.Execute(context.Background(), jsonFileArg(fp))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected content, got %q", out)
	}
}

func TestScreenshotTool_Interface(t *testing.T) {
	tool := &ScreenshotTool{}
	if tool.Name() != "desktop_screenshot" {
		t.Errorf("Name: got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description empty")
	}
	if tool.Parameters() == nil {
		t.Error("Parameters nil")
	}
}

func TestBrowserTool_Interface(t *testing.T) {
	tool := &BrowserTool{}
	if tool.Name() != "browser_use" {
		t.Errorf("Name: got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description empty")
	}
	if tool.Parameters() == nil {
		t.Error("Parameters nil")
	}
}

func TestBrowserTool_UnknownAction(t *testing.T) {
	tool := &BrowserTool{}
	_, err := tool.Execute(context.Background(), `{"action":"nonexistent"}`)
	if err == nil {
		t.Error("expected error for unknown action")
	}
}

func TestBrowserTool_MissingPage(t *testing.T) {
	tool := &BrowserTool{}
	_, err := tool.Execute(context.Background(), `{"action":"click","selector":"#btn"}`)
	if err == nil {
		t.Error("expected error for missing page")
	}
}

func TestRichExecutorInterface(t *testing.T) {
	var _ agent.RichExecutor = (*BrowserTool)(nil)
	var _ agent.RichExecutor = (*ScreenshotTool)(nil)
	var _ agent.RichExecutor = (*SendFileTool)(nil)
}

func TestDetectMIME(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"file.png", "image/png"},
		{"file.jpg", "image/jpeg"},
		{"file.go", "text/plain"},
		{"file.md", "text/markdown"},
		{"file.yaml", "text/yaml"},
		{"file", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := detectMIME(tt.path)
		if got != tt.want {
			t.Errorf("detectMIME(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestIsTextMIME(t *testing.T) {
	if !isTextMIME("text/plain") {
		t.Error("text/plain should be text")
	}
	if !isTextMIME("application/json") {
		t.Error("application/json should be text")
	}
	if isTextMIME("image/png") {
		t.Error("image/png should not be text")
	}
}

func TestRegisterBuiltinNewTools(t *testing.T) {
	builtins = nil
	tools := RegisterBuiltin()
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}
	for _, name := range []string{"browser_use", "desktop_screenshot", "send_file_to_user"} {
		if !names[name] {
			t.Errorf("%s tool not registered", name)
		}
	}
}
