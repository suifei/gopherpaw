// Package llm provides message formatting for LLM API requests.
package llm

import (
	"encoding/json"
	"strings"

	"github.com/suifei/gopherpaw/internal/agent"
)

// Formatter processes messages before sending to the LLM API.
type Formatter interface {
	// Format transforms agent messages into the format expected by the LLM.
	Format(messages []agent.Message) []agent.Message
}

// FileBlockSupportFormatter wraps a base formatter and adds support for
// file blocks in tool results, converting them to text descriptions.
type FileBlockSupportFormatter struct {
	StripMessageName bool
}

// Format processes messages: sanitizes tool message pairing, converts file
// blocks in tool results to text, and optionally strips message names.
func (f *FileBlockSupportFormatter) Format(messages []agent.Message) []agent.Message {
	messages = agent.SanitizeToolMessages(messages)
	messages = convertFileBlocks(messages)
	if f.StripMessageName {
		messages = stripTopLevelMessageName(messages)
	}
	return messages
}

// fileBlock represents a file reference in a tool result content.
type fileBlock struct {
	Type     string `json:"type"`
	FilePath string `json:"file_path,omitempty"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	URL      string `json:"url,omitempty"`
}

// convertFileBlocks scans tool result messages for JSON file blocks
// and converts them to human-readable text descriptions.
func convertFileBlocks(messages []agent.Message) []agent.Message {
	result := make([]agent.Message, len(messages))
	for i, m := range messages {
		if m.Role == "tool" && m.Content != "" {
			m.Content = processToolContent(m.Content)
		}
		result[i] = m
	}
	return result
}

// processToolContent checks if tool content contains file blocks and
// converts them to text descriptions.
func processToolContent(content string) string {
	trimmed := strings.TrimSpace(content)

	if strings.HasPrefix(trimmed, "[") {
		var blocks []json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &blocks); err == nil {
			return processContentBlocks(blocks)
		}
	}

	if strings.HasPrefix(trimmed, "{") {
		var block fileBlock
		if err := json.Unmarshal([]byte(trimmed), &block); err == nil && block.Type == "file" {
			return formatFileBlock(block)
		}
	}

	return content
}

func processContentBlocks(blocks []json.RawMessage) string {
	var parts []string
	for _, raw := range blocks {
		var block struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		}
		if err := json.Unmarshal(raw, &block); err != nil {
			parts = append(parts, string(raw))
			continue
		}

		switch block.Type {
		case "file":
			var fb fileBlock
			if err := json.Unmarshal(raw, &fb); err == nil {
				parts = append(parts, formatFileBlock(fb))
			}
		case "text":
			parts = append(parts, block.Text)
		default:
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, "\n")
}

func formatFileBlock(fb fileBlock) string {
	name := fb.FileName
	if name == "" {
		name = fb.FilePath
	}
	if name == "" {
		name = fb.URL
	}
	mime := fb.MimeType
	if mime == "" {
		mime = "application/octet-stream"
	}
	return "[File: " + name + " (" + mime + ")]"
}

// stripTopLevelMessageName removes the "name" field from messages to
// ensure compatibility with APIs that don't support it.
func stripTopLevelMessageName(messages []agent.Message) []agent.Message {
	return messages
}

// NewFileBlockFormatter creates a FileBlockSupportFormatter with default settings.
func NewFileBlockFormatter() *FileBlockSupportFormatter {
	return &FileBlockSupportFormatter{
		StripMessageName: true,
	}
}
