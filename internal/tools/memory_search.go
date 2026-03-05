package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/suifei/gopherpaw/internal/agent"
)

// MemorySearchTool searches memory semantically (hybrid: vector + BM25).
type MemorySearchTool struct{}

// Name returns the tool identifier.
func (t *MemorySearchTool) Name() string { return "memory_search" }

// Description returns a human-readable description.
func (t *MemorySearchTool) Description() string {
	return "Search MEMORY.md and memory/*.md files semantically. Use before answering questions about prior work, decisions, dates, people, preferences, or todos. Returns top relevant snippets."
}

// Parameters returns the JSON Schema.
func (t *MemorySearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":      map[string]any{"type": "string", "description": "Semantic search query"},
			"top_k":      map[string]any{"type": "integer", "description": "Max results (default 5)"},
			"min_score":  map[string]any{"type": "number", "description": "Minimum score threshold (default 0.1)"},
		},
		"required": []string{"query"},
	}
}

type memorySearchArgs struct {
	Query    string  `json:"query"`
	TopK     int     `json:"top_k"`
	MinScore float64 `json:"min_score"`
}

// Execute runs the tool.
func (t *MemorySearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	store := agent.GetMemoryStore(ctx)
	if store == nil {
		return "Error: Memory store is not available.", nil
	}
	chatID := agent.GetChatID(ctx)
	if chatID == "" {
		return "Error: Chat context is not available.", nil
	}
	var args memorySearchArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Query == "" {
		return "Error: query cannot be empty.", nil
	}
	topK := args.TopK
	if topK <= 0 {
		topK = 5
	}
	results, err := store.Search(ctx, chatID, args.Query, topK)
	if err != nil {
		return fmt.Sprintf("Error: Memory search failed: %v", err), nil
	}
	minScore := args.MinScore
	if minScore <= 0 {
		minScore = 0.1
	}
	var out strings.Builder
	for i, r := range results {
		if r.Score < minScore {
			continue
		}
		if i > 0 {
			out.WriteString("\n\n---\n\n")
		}
		out.WriteString(fmt.Sprintf("[Score: %.2f]\n%s", r.Score, r.Content))
	}
	if out.Len() == 0 {
		return "No relevant memory found for the query.", nil
	}
	return out.String(), nil
}
