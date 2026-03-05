package memory

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

// CompactSummaryTemplate is the prompt template for LLM to generate compaction summary.
const CompactSummaryTemplate = `Summarize the following conversation into a structured summary. Include:
1. 目标 (Goals): What the user is trying to achieve
2. 约束与偏好 (Constraints/Preferences): User preferences, constraints
3. 进展 (Progress): What has been done so far
4. 关键决策 (Key Decisions): Important decisions made
5. 下一步 (Next Steps): Suggested next actions
6. 关键上下文 (Key Context): Other important context

Conversation:
%s

Previous summary (if any):
%s

Output the summary in the same structure (目标/约束与偏好/进展/关键决策/下一步/关键上下文).`

// Compactor handles context compaction with optional LLM summarization.
type Compactor struct {
	llm agent.LLMProvider
	cfg config.MemoryConfig
}

// NewCompactor creates a Compactor. llm may be nil for truncation-only mode.
func NewCompactor(llm agent.LLMProvider, cfg config.MemoryConfig) *Compactor {
	return &Compactor{llm: llm, cfg: cfg}
}

// CompactWithLLM compacts messages using LLM to generate a summary.
// Returns the summary and the messages to keep (recent N).
func (c *Compactor) CompactWithLLM(ctx context.Context, messages []agent.Message, previousSummary string) (string, []agent.Message, error) {
	keepRecent := c.cfg.CompactKeepRecent
	if keepRecent <= 0 {
		keepRecent = 3
	}
	if c.llm == nil {
		start := len(messages) - keepRecent
		if start < 0 {
			start = 0
		}
		return "", messages[start:], nil
	}
	var conv strings.Builder
	for _, m := range messages {
		conv.WriteString(m.Role)
		conv.WriteString(": ")
		conv.WriteString(m.Content)
		conv.WriteString("\n")
	}
	prompt := fmt.Sprintf(CompactSummaryTemplate, conv.String(), previousSummary)
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "system", Content: "You are a helpful assistant that summarizes conversations concisely."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   1024,
	}
	resp, err := c.llm.Chat(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("llm compact: %w", err)
	}
	summary := strings.TrimSpace(resp.Content)
	start := len(messages) - keepRecent
	if start < 0 {
		start = 0
	}
	return summary, messages[start:], nil
}

// EstimateTokens approximates token count (chars/4 for English, chars/2 for CJK).
func EstimateTokens(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			n += 2
		} else {
			n++
		}
	}
	return (utf8.RuneCountInString(s) + n) / 2
}

// ShouldCompact returns true if total token estimate exceeds threshold.
func ShouldCompact(messages []agent.Message, threshold int) bool {
	if threshold <= 0 {
		threshold = 100000
	}
	var total int
	for _, m := range messages {
		total += EstimateTokens(m.Content)
	}
	return total > threshold
}
