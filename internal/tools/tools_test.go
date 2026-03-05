package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTimeTool(t *testing.T) {
	tool := &TimeTool{}
	if tool.Name() != "get_current_time" {
		t.Errorf("Name: got %s", tool.Name())
	}
	out, err := tool.Execute(context.Background(), "{}")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestShellTool(t *testing.T) {
	tool := &ShellTool{}
	out, err := tool.Execute(context.Background(), `{"command":"echo hello"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "hello" && out != "Command executed successfully (no output)." {
		t.Errorf("Execute: got %q", out)
	}
}

func TestReadFileTool(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.txt")
	os.WriteFile(fp, []byte("line1\nline2\nline3"), 0644)
	tool := &ReadFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"test.txt"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "line1\nline2\nline3" {
		t.Errorf("Execute: got %q", out)
	}
}

func TestWriteFileTool(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &WriteFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"out.txt","content":"hello"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "Wrote 5 bytes to "+filepath.Join(dir, "out.txt")+"." {
		t.Errorf("Execute: got %q", out)
	}
}

func TestEditFileTool(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "edit.txt")
	os.WriteFile(fp, []byte("hello world\nfoo bar\nhello again"), 0644)

	tool := &EditFileTool{}
	if tool.Name() != "edit_file" {
		t.Errorf("Name: got %s", tool.Name())
	}
	// Replace first occurrence
	out, err := tool.Execute(context.Background(), `{"file_path":"edit.txt","old_text":"hello","new_text":"hi"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Successfully replaced") {
		t.Errorf("Execute: got %q", out)
	}
	data, _ := os.ReadFile(fp)
	content := string(data)
	if !strings.Contains(content, "hi world") {
		t.Errorf("Expected 'hi world' in file, got %q", content)
	}
	if !strings.Contains(content, "hi again") {
		t.Errorf("Expected 'hi again' in file, got %q", content)
	}
	// Old text not found
	out, err = tool.Execute(context.Background(), `{"file_path":"edit.txt","old_text":"nonexistent","new_text":"x"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "not found") {
		t.Errorf("Execute: expected error for not found, got %q", out)
	}
}

func TestAppendFileTool(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "append.txt")
	os.WriteFile(fp, []byte("line1\n"), 0644)

	tool := &AppendFileTool{}
	if tool.Name() != "append_file" {
		t.Errorf("Name: got %s", tool.Name())
	}
	out, err := tool.Execute(context.Background(), `{"file_path":"append.txt","content":"line2\n"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Appended") {
		t.Errorf("Execute: got %q", out)
	}
	data, _ := os.ReadFile(fp)
	if string(data) != "line1\nline2\n" {
		t.Errorf("Expected 'line1\\nline2\\n', got %q", string(data))
	}
	// Empty file_path
	out, err = tool.Execute(context.Background(), `{"file_path":"","content":"x"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") {
		t.Errorf("Execute: expected error for empty path, got %q", out)
	}
}

func TestGrepSearchTool(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("foo\nbar\nbaz"), 0644)
	tool := &GrepSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"bar","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out == "" || out == "No matches found" {
		t.Errorf("Execute: got %q", out)
	}
}

func TestGlobSearchTool(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x"), 0644)
	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"*.go","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out == "" || out == "No files matched" {
		t.Errorf("Execute: got %q", out)
	}
}

func TestRegisterBuiltin(t *testing.T) {
	tools := RegisterBuiltin()
	if len(tools) < 10 {
		t.Errorf("expected at least 10 tools (incl. edit_file, append_file, web_search, http_request), got %d", len(tools))
	}
	names := make(map[string]bool)
	for _, t := range tools {
		names[t.Name()] = true
	}
	if !names["edit_file"] {
		t.Error("edit_file tool not registered")
	}
	if !names["append_file"] {
		t.Error("append_file tool not registered")
	}
	if !names["web_search"] {
		t.Error("web_search tool not registered")
	}
	if !names["http_request"] {
		t.Error("http_request tool not registered")
	}
}

func TestWebSearchTool(t *testing.T) {
	ws, err := NewWebSearchTool()
	if err != nil {
		t.Fatalf("NewWebSearchTool: %v", err)
	}
	if ws.Name() != "web_search" {
		t.Errorf("Name: got %s", ws.Name())
	}
	if ws.Description() == "" {
		t.Error("Description empty")
	}
	if ws.Parameters() == nil {
		t.Error("Parameters nil")
	}
	// Empty query
	out, err := ws.Execute(context.Background(), `{"query":""}`)
	if err != nil {
		t.Fatalf("Execute empty query: %v", err)
	}
	if !strings.Contains(out, "Error") && !strings.Contains(out, "No") {
		t.Errorf("Execute empty query: got %q", out)
	}
	// Invalid JSON
	_, err = ws.Execute(context.Background(), `{invalid`)
	if err == nil {
		t.Error("Execute invalid JSON: expected error")
	}
}

func TestHTTPTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ht := NewHTTPTool()
	if ht.Name() != "http_request" {
		t.Errorf("Name: got %s", ht.Name())
	}
	if ht.Description() == "" {
		t.Error("Description empty")
	}
	if ht.Parameters() == nil {
		t.Error("Parameters nil")
	}
	// Empty URL
	out, err := ht.Execute(context.Background(), `{"url":""}`)
	if err != nil {
		t.Fatalf("Execute empty URL: %v", err)
	}
	if !strings.Contains(out, "Error") {
		t.Errorf("Execute empty URL: got %q", out)
	}
	// Valid GET
	out, err = ht.Execute(context.Background(), `{"url":"`+srv.URL+`","method":"GET"}`)
	if err != nil {
		t.Fatalf("Execute GET: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("Execute GET: got %q", out)
	}
	// Invalid JSON
	_, err = ht.Execute(context.Background(), `{invalid`)
	if err == nil {
		t.Error("Execute invalid JSON: expected error")
	}
}
