package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool_NotExists(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &ReadFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"nonexistent.txt"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "does not exist") {
		t.Errorf("Expected file not found error, got %q", out)
	}
}

func TestReadFileTool_NotFile(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)

	tool := &ReadFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"subdir"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "not a file") {
		t.Errorf("Expected not a file error, got %q", out)
	}
}

func TestReadFileTool_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "empty.txt")
	os.WriteFile(fp, []byte(""), 0644)

	tool := &ReadFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"empty.txt"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "" {
		t.Errorf("Expected empty output, got %q", out)
	}
}

func TestReadFileTool_LineRange(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	content := "line1\nline2\nline3\nline4\nline5"
	fp := filepath.Join(dir, "lines.txt")
	os.WriteFile(fp, []byte(content), 0644)

	tool := &ReadFileTool{}
	tests := []struct {
		name      string
		args      string
		wantLines []string
	}{
		{
			name:      "start_line only",
			args:      `{"file_path":"lines.txt","start_line":2}`,
			wantLines: []string{"line2", "line3", "line4", "line5"},
		},
		{
			name:      "end_line only",
			args:      `{"file_path":"lines.txt","end_line":3}`,
			wantLines: []string{"line1", "line2", "line3"},
		},
		{
			name:      "both start and end",
			args:      `{"file_path":"lines.txt","start_line":2,"end_line":4}`,
			wantLines: []string{"line2", "line3", "line4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tool.Execute(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			for _, wantLine := range tt.wantLines {
				if !strings.Contains(out, wantLine) {
					t.Errorf("Expected %q in output, got %q", wantLine, out)
				}
			}
		})
	}
}

func TestReadFileTool_InvalidLineRange(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.txt")
	os.WriteFile(fp, []byte("line1\nline2\nline3"), 0644)

	tool := &ReadFileTool{}
	tests := []struct {
		name    string
		args    string
		wantErr string
	}{
		{
			name:    "start > end",
			args:    `{"file_path":"test.txt","start_line":3,"end_line":1}`,
			wantErr: "start_line (3) > end_line (1)",
		},
		{
			name:    "start exceeds length",
			args:    `{"file_path":"test.txt","start_line":10,"end_line":3}`,
			wantErr: "start_line (10) > end_line (3)",
		},
		{
			name:    "negative start",
			args:    `{"file_path":"test.txt","start_line":-1}`,
			wantErr: "",
		},
		{
			name:    "end exceeds length",
			args:    `{"file_path":"test.txt","end_line":10}`,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tool.Execute(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if tt.wantErr != "" && !strings.Contains(out, tt.wantErr) {
				t.Errorf("Expected error %q, got %q", tt.wantErr, out)
			}
		})
	}
}

func TestWriteFileTool_EmptyPath(t *testing.T) {
	tool := &WriteFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"","content":"hello"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") {
		t.Errorf("Expected error for empty path, got %q", out)
	}
}

func TestWriteFileTool_CreateSubdir(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)
	tool := &WriteFileTool{}

	fp := filepath.Join(subdir, "test.txt")
	out, err := tool.Execute(context.Background(), `{"file_path":"subdir/test.txt","content":"hello"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Wrote") {
		t.Errorf("Expected success message, got %q", out)
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("Expected 'hello', got %q", string(data))
	}
}

func TestWriteFileTool_Overwrite(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.txt")
	os.WriteFile(fp, []byte("old content"), 0644)

	tool := &WriteFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"test.txt","content":"new content"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Wrote") {
		t.Errorf("Expected success message, got %q", out)
	}

	data, _ := os.ReadFile(fp)
	if string(data) != "new content" {
		t.Errorf("Expected 'new content', got %q", string(data))
	}
}

func TestWriteFileTool_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &WriteFileTool{}

	out, err := tool.Execute(context.Background(), `{"file_path":"empty.txt","content":""}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Wrote 0 bytes") {
		t.Errorf("Expected 0 bytes message, got %q", out)
	}

	fp := filepath.Join(dir, "empty.txt")
	if _, err := os.Stat(fp); err != nil {
		t.Errorf("File should exist: %v", err)
	}
}

func TestEditFileTool_NotExists(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &EditFileTool{}

	out, err := tool.Execute(context.Background(), `{"file_path":"nonexistent.txt","old_text":"old","new_text":"new"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "does not exist") {
		t.Errorf("Expected file not found error, got %q", out)
	}
}

func TestEditFileTool_NotFile(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)

	tool := &EditFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"subdir","old_text":"old","new_text":"new"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "not a file") {
		t.Errorf("Expected not a file error, got %q", out)
	}
}

func TestEditFileTool_TextNotFound(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.txt")
	os.WriteFile(fp, []byte("hello world"), 0644)

	tool := &EditFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"test.txt","old_text":"nonexistent","new_text":"new"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "not found") {
		t.Errorf("Expected text not found error, got %q", out)
	}
}

func TestAppendFileTool_CreateNew(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &AppendFileTool{}

	fp := filepath.Join(dir, "new.txt")
	out, err := tool.Execute(context.Background(), `{"file_path":"new.txt","content":"hello"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Appended") {
		t.Errorf("Expected success message, got %q", out)
	}

	data, _ := os.ReadFile(fp)
	if string(data) != "hello" {
		t.Errorf("Expected 'hello', got %q", string(data))
	}
}

func TestAppendFileTool_MultipleAppends(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.txt")
	os.WriteFile(fp, []byte("line1\n"), 0644)

	tool := &AppendFileTool{}
	tool.Execute(context.Background(), `{"file_path":"test.txt","content":"line2\n"}`)
	tool.Execute(context.Background(), `{"file_path":"test.txt","content":"line3\n"}`)

	data, _ := os.ReadFile(fp)
	expected := "line1\nline2\nline3\n"
	if string(data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(data))
	}
}

func TestAppendFileTool_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "test.txt")
	os.WriteFile(fp, []byte("existing"), 0644)

	tool := &AppendFileTool{}
	out, err := tool.Execute(context.Background(), `{"file_path":"test.txt","content":""}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Appended 0 bytes") {
		t.Errorf("Expected 0 bytes message, got %q", out)
	}

	data, _ := os.ReadFile(fp)
	if string(data) != "existing" {
		t.Errorf("File content should be unchanged, got %q", string(data))
	}
}
