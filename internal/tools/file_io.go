package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/suifei/gopherpaw/internal/agent"
)

// workingDir is the base path for relative file operations.
var (
	workingDir   string
	workingDirMu sync.Mutex
)

// SetWorkingDir sets the working directory for file tools.
func SetWorkingDir(dir string) {
	workingDirMu.Lock()
	defer workingDirMu.Unlock()
	workingDir = dir
}

func getWorkingDir() string {
	workingDirMu.Lock()
	defer workingDirMu.Unlock()
	return workingDir
}

func resolvePath(p string) string {
	path := filepath.Clean(p)
	if !filepath.IsAbs(path) {
		path = filepath.Join(getWorkingDir(), path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// ReadFileTool reads file contents.
type ReadFileTool struct{}

// Name returns the tool identifier.
func (t *ReadFileTool) Name() string { return "read_file" }

// Description returns a human-readable description.
func (t *ReadFileTool) Description() string {
	return "Read a file. Relative paths resolve from working directory. Use start_line/end_line for line range (1-based)."
}

// Parameters returns the JSON Schema.
func (t *ReadFileTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path":  map[string]any{"type": "string", "description": "Path to file"},
			"start_line": map[string]any{"type": "integer", "description": "First line (1-based)"},
			"end_line":   map[string]any{"type": "integer", "description": "Last line (1-based)"},
		},
		"required": []string{"file_path"},
	}
}

type readFileArgs struct {
	FilePath  string `json:"file_path"`
	StartLine *int   `json:"start_line"`
	EndLine   *int   `json:"end_line"`
}

// Execute runs the tool.
func (t *ReadFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args readFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	fp := resolvePath(args.FilePath)
	if !pathExists(fp) {
		return fmt.Sprintf("Error: File %s does not exist.", fp), nil
	}
	if !isFile(fp) {
		return fmt.Sprintf("Error: %s is not a file.", fp), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: Read failed: %v", err), nil
	}
	lines := strings.Split(string(data), "\n")
	total := len(lines)
	s, e := 1, total
	if args.StartLine != nil {
		s = *args.StartLine
	}
	if args.EndLine != nil {
		e = *args.EndLine
	}
	if s < 1 {
		s = 1
	}
	if e > total {
		e = total
	}
	if s > e {
		return fmt.Sprintf("Error: start_line (%d) > end_line (%d)", s, e), nil
	}
	if s > total {
		return fmt.Sprintf("Error: start_line %d exceeds file length (%d)", s, total), nil
	}
	selected := lines[s-1 : e]
	content := strings.Join(selected, "\n")
	if args.StartLine != nil || args.EndLine != nil {
		return fmt.Sprintf("%s (lines %d-%d of %d)\n%s", fp, s, e, total, content), nil
	}
	return content, nil
}

// WriteFileTool writes file contents.
type WriteFileTool struct{}

// Name returns the tool identifier.
func (t *WriteFileTool) Name() string { return "write_file" }

// Description returns a human-readable description.
func (t *WriteFileTool) Description() string {
	return "Create or overwrite a file. Relative paths resolve from working directory."
}

// Parameters returns the JSON Schema.
func (t *WriteFileTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Path to file"},
			"content":   map[string]any{"type": "string", "description": "Content to write"},
		},
		"required": []string{"file_path", "content"},
	}
}

type writeFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// Execute runs the tool.
func (t *WriteFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args writeFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.FilePath == "" {
		return "Error: No file_path provided.", nil
	}
	fp := resolvePath(args.FilePath)
	if err := os.WriteFile(fp, []byte(args.Content), 0644); err != nil {
		return fmt.Sprintf("Error: Write failed: %v", err), nil
	}
	return fmt.Sprintf("Wrote %d bytes to %s.", len(args.Content), fp), nil
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// EditFileTool finds and replaces text in a file.
type EditFileTool struct{}

// Name returns the tool identifier.
func (t *EditFileTool) Name() string { return "edit_file" }

// Description returns a human-readable description.
func (t *EditFileTool) Description() string {
	return "Find-and-replace text in a file. All occurrences of old_text are replaced with new_text. Relative paths resolve from working directory."
}

// Parameters returns the JSON Schema.
func (t *EditFileTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Path to file"},
			"old_text":  map[string]any{"type": "string", "description": "Exact text to find"},
			"new_text":  map[string]any{"type": "string", "description": "Replacement text"},
		},
		"required": []string{"file_path", "old_text", "new_text"},
	}
}

type editFileArgs struct {
	FilePath string `json:"file_path"`
	OldText  string `json:"old_text"`
	NewText  string `json:"new_text"`
}

// Execute runs the tool.
func (t *EditFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args editFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	fp := resolvePath(args.FilePath)
	if !pathExists(fp) {
		return fmt.Sprintf("Error: File %s does not exist.", fp), nil
	}
	if !isFile(fp) {
		return fmt.Sprintf("Error: %s is not a file.", fp), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: Read failed: %v", err), nil
	}
	content := string(data)
	if !strings.Contains(content, args.OldText) {
		return fmt.Sprintf("Error: The text to replace was not found in %s.", fp), nil
	}
	newContent := strings.ReplaceAll(content, args.OldText, args.NewText)
	if err := os.WriteFile(fp, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: Write failed: %v", err), nil
	}
	return fmt.Sprintf("Successfully replaced text in %s.", fp), nil
}

// AppendFileTool appends content to the end of a file.
type AppendFileTool struct{}

// Name returns the tool identifier.
func (t *AppendFileTool) Name() string { return "append_file" }

// Description returns a human-readable description.
func (t *AppendFileTool) Description() string {
	return "Append content to the end of a file. Relative paths resolve from working directory."
}

// Parameters returns the JSON Schema.
func (t *AppendFileTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Path to file"},
			"content":   map[string]any{"type": "string", "description": "Content to append"},
		},
		"required": []string{"file_path", "content"},
	}
}

type appendFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// Execute runs the tool.
func (t *AppendFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args appendFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.FilePath == "" {
		return "Error: No file_path provided.", nil
	}
	fp := resolvePath(args.FilePath)
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Error: Append failed: %v", err), nil
	}
	defer f.Close()
	if _, err := f.WriteString(args.Content); err != nil {
		return fmt.Sprintf("Error: Append failed: %v", err), nil
	}
	return fmt.Sprintf("Appended %d bytes to %s.", len(args.Content), fp), nil
}

type ViewTextFileTool struct{}

func (t *ViewTextFileTool) Name() string { return "view_text_file" }

func (t *ViewTextFileTool) Description() string {
	return "View a text file safely. Checks if file is binary and rejects it. Relative paths resolve from working directory."
}

func (t *ViewTextFileTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Path to text file"},
		},
		"required": []string{"file_path"},
	}
}

type viewTextFileArgs struct {
	FilePath string `json:"file_path"`
}

func (t *ViewTextFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args viewTextFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	fp := resolvePath(args.FilePath)
	if !pathExists(fp) {
		return fmt.Sprintf("Error: File %s does not exist.", fp), nil
	}
	if !isFile(fp) {
		return fmt.Sprintf("Error: %s is not a file.", fp), nil
	}

	ext := strings.ToLower(filepath.Ext(fp))
	binaryExts := map[string]bool{
		".exe": true, ".bin": true, ".dll": true, ".so": true, ".dylib": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".ico": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".zip": true, ".tar": true, ".gz": true, ".rar": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".wav": true,
		".pyc": true, ".class": true, ".jar": true, ".war": true,
	}
	if binaryExts[ext] {
		return fmt.Sprintf("Error: %s is a binary file and cannot be viewed as text.", fp), nil
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: Failed to read file: %v", err), nil
	}

	return string(data), nil
}

var _ agent.Tool = (*ReadFileTool)(nil)
var _ agent.Tool = (*WriteFileTool)(nil)
var _ agent.Tool = (*EditFileTool)(nil)
var _ agent.Tool = (*AppendFileTool)(nil)
var _ agent.Tool = (*ViewTextFileTool)(nil)
