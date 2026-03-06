package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepSearchTool_EmptyPattern(t *testing.T) {
	tool := &GrepSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":""}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "No pattern provided") {
		t.Errorf("Expected no pattern error, got %q", out)
	}
}

func TestGrepSearchTool_PathNotExists(t *testing.T) {
	tool := &GrepSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"test","path":"/nonexistent/path"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "does not exist") {
		t.Errorf("Expected path not exists error, got %q", out)
	}
}

func TestGrepSearchTool_NoMatches(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello\nworld\n"), 0644)

	tool := &GrepSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"nonexistent","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "No matches found") {
		t.Errorf("Expected no matches message, got %q", out)
	}
}

func TestGrepSearchTool_CaseSensitive(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("Hello\nhello\nHELLO\n"), 0644)

	tool := &GrepSearchTool{}

	tests := []struct {
		name          string
		args          string
		wantCount     int
		caseSensitive bool
	}{
		{
			name:          "case insensitive (default)",
			args:          `{"pattern":"hello","path":".","case_sensitive":false}`,
			wantCount:     3,
			caseSensitive: false,
		},
		{
			name:          "case sensitive",
			args:          `{"pattern":"hello","path":".","case_sensitive":true}`,
			wantCount:     1,
			caseSensitive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tool.Execute(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			count := strings.Count(out, "test.txt:")
			if count != tt.wantCount {
				t.Errorf("Expected %d matches for case_sensitive=%t, got %d: %s", tt.wantCount, tt.caseSensitive, count, out)
			}
		})
	}
}

func TestGrepSearchTool_Regex(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("email@example.com\ntest@test.org\n"), 0644)

	tool := &GrepSearchTool{}

	tests := []struct {
		name     string
		args     string
		wantLine string
	}{
		{
			name:     "simple regex",
			args:     `{"pattern":"\\w+@\\w+\\.\\w+","path":".","is_regex":true}`,
			wantLine: "email@example.com",
		},
		{
			name:     "non-regex (literal)",
			args:     `{"pattern":"email@example.com","path":".","is_regex":false}`,
			wantLine: "email@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tool.Execute(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if !strings.Contains(out, tt.wantLine) {
				t.Errorf("Expected %q in output, got %q", tt.wantLine, out)
			}
		})
	}
}

func TestGrepSearchTool_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &GrepSearchTool{}

	out, err := tool.Execute(context.Background(), `{"pattern":"[invalid","path":".","is_regex":true}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "Invalid regex") {
		t.Errorf("Expected invalid regex error, got %q", out)
	}
}

func TestGrepSearchTool_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("hello there\n"), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("goodbye\n"), 0644)

	tool := &GrepSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"hello","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out, "a.txt:") {
		t.Errorf("Expected a.txt in results, got %q", out)
	}
	if !strings.Contains(out, "b.txt:") {
		t.Errorf("Expected b.txt in results, got %q", out)
	}
	if strings.Contains(out, "c.txt:") {
		t.Errorf("Did not expect c.txt in results, got %q", out)
	}
}

func TestGrepSearchTool_MultiLineMatch(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	content := "line1\nline2\nline3\nline4\nline5"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644)

	tool := &GrepSearchTool{}

	tests := []struct {
		name         string
		args         string
		wantMinLines int
	}{
		{
			name:         "match single line",
			args:         `{"pattern":"line3","path":"."}`,
			wantMinLines: 1,
		},
		{
			name:         "match all lines",
			args:         `{"pattern":"line","path":"."}`,
			wantMinLines: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tool.Execute(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			count := strings.Count(out, "test.txt:")
			if count < tt.wantMinLines {
				t.Errorf("Expected at least %d matches, got %d", tt.wantMinLines, count)
			}
		})
	}
}

func TestGlobSearchTool_EmptyPattern(t *testing.T) {
	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":""}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "No pattern provided") {
		t.Errorf("Expected no pattern error, got %q", out)
	}
}

func TestGlobSearchTool_PathNotExists(t *testing.T) {
	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"*.go","path":"/nonexistent/path"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "does not exist") {
		t.Errorf("Expected path not exists error, got %q", out)
	}
}

func TestGlobSearchTool_PathNotDirectory(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	fp := filepath.Join(dir, "file.txt")
	os.WriteFile(fp, []byte("test"), 0644)

	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"*.go","path":"file.txt"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "not a directory") {
		t.Errorf("Expected not a directory error, got %q", out)
	}
}

func TestGlobSearchTool_NoMatches(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	tool := &GlobSearchTool{}

	out, err := tool.Execute(context.Background(), `{"pattern":"*.nonexistent","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "No files matched") {
		t.Errorf("Expected no matches message, got %q", out)
	}
}

func TestGlobSearchTool_MultipleExtensions(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b"), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(dir, "d.go"), []byte("package d"), 0644)

	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"*.go","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out, "a.go") || !strings.Contains(out, "b.go") || !strings.Contains(out, "d.go") {
		t.Errorf("Expected all .go files in results, got %q", out)
	}
	if strings.Contains(out, "c.txt") {
		t.Errorf("Did not expect c.txt in results, got %q", out)
	}
}

func TestGlobSearchTool_RecursivePattern(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "subdir", "test.go"), []byte("package test"), 0644)

	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"test.go","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out, "test.go") {
		t.Errorf("Expected test.go in results, got %q", out)
	}
}

func TestGlobSearchTool_NestedDirectory(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)
	os.Mkdir(filepath.Join(dir, "a"), 0755)
	os.Mkdir(filepath.Join(dir, "a", "b"), 0755)
	os.WriteFile(filepath.Join(dir, "a", "b", "test.txt"), []byte("test"), 0644)

	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"test.txt","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out, "test.txt") {
		t.Errorf("Expected test.txt in results, got %q", out)
	}
}

func TestGlobSearchTool_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	SetWorkingDir(dir)

	tool := &GlobSearchTool{}
	out, err := tool.Execute(context.Background(), `{"pattern":"*","path":"."}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out, "No files matched") {
		t.Errorf("Expected no matches message, got %q", out)
	}
}

func TestIsTextFile(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  []byte
		wantText bool
	}{
		{
			name:     "text file",
			filename: "test.txt",
			content:  []byte("hello world"),
			wantText: true,
		},
		{
			name:     "go file",
			filename: "test.go",
			content:  []byte("package test"),
			wantText: true,
		},
		{
			name:     "binary png",
			filename: "test.png",
			content:  []byte("\x89PNG\r\n\x1a\n"),
			wantText: false,
		},
		{
			name:     "binary jpg",
			filename: "test.jpg",
			content:  []byte("\xff\xd8\xff\xe0"),
			wantText: false,
		},
		{
			name:     "empty file",
			filename: "empty.txt",
			content:  []byte(""),
			wantText: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := filepath.Join(dir, tt.filename)
			os.WriteFile(fp, tt.content, 0644)

			gotText := isTextFile(fp)
			if gotText != tt.wantText {
				t.Errorf("isTextFile(%q) = %v, want %v", tt.filename, gotText, tt.wantText)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAbs bool
	}{
		{
			name:    "relative path",
			input:   "test.txt",
			wantAbs: true,
		},
		{
			name:    "absolute path",
			input:   "/tmp/test.txt",
			wantAbs: true,
		},
		{
			name:    "path with .",
			input:   "./test.txt",
			wantAbs: true,
		},
		{
			name:    "path with ..",
			input:   "../test.txt",
			wantAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvePath(tt.input)
			if tt.wantAbs && !filepath.IsAbs(result) {
				t.Errorf("resolvePath(%q) = %q, want absolute path", tt.input, result)
			}
		})
	}
}
