package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/suifei/gopherpaw/internal/agent"
)

type sendFileArgs struct {
	FilePath string `json:"file_path"`
}

// SendFileTool sends a local file to the user via the current channel.
type SendFileTool struct{}

func (t *SendFileTool) Name() string { return "send_file_to_user" }

func (t *SendFileTool) Description() string {
	return "Send a local file to the user. For text files, returns content directly. For images/audio/video/binary files, delivers the file through the current messaging channel."
}

func (t *SendFileTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Path to the local file to send"},
		},
		"required": []string{"file_path"},
	}
}

func (t *SendFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	result, err := t.ExecuteRich(ctx, arguments)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func (t *SendFileTool) ExecuteRich(ctx context.Context, arguments string) (*agent.ToolResult, error) {
	var args sendFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}
	if args.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	resolvedPath := args.FilePath
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(workingDir, resolvedPath)
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", resolvedPath)
	}

	mimeType := detectMIME(resolvedPath)

	if isTextMIME(mimeType) {
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		const maxText = 32000
		content := string(data)
		if len(content) > maxText {
			content = content[:maxText] + "\n... [truncated]"
		}
		return &agent.ToolResult{
			Text: fmt.Sprintf("File content of %s:\n\n%s", filepath.Base(resolvedPath), content),
		}, nil
	}

	return &agent.ToolResult{
		Text: fmt.Sprintf("Sent file: %s (%s, %d bytes)", filepath.Base(resolvedPath), mimeType, info.Size()),
		Attachments: []agent.Attachment{{
			FilePath: resolvedPath,
			MimeType: mimeType,
			FileName: filepath.Base(resolvedPath),
		}},
	}, nil
}

func detectMIME(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}
	ext = strings.ToLower(ext)
	switch ext {
	case ".md", ".markdown":
		return "text/markdown"
	case ".go", ".py", ".js", ".ts", ".rs", ".c", ".cpp", ".h", ".java":
		return "text/plain"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".toml":
		return "text/toml"
	default:
		mt := mime.TypeByExtension(ext)
		if mt != "" {
			return mt
		}
		return "application/octet-stream"
	}
}

func isTextMIME(mt string) bool {
	return strings.HasPrefix(mt, "text/") ||
		mt == "application/json" ||
		mt == "application/xml" ||
		mt == "application/javascript"
}

var (
	_ agent.Tool         = (*SendFileTool)(nil)
	_ agent.RichExecutor = (*SendFileTool)(nil)
)
