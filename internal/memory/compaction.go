package memory

import (
	"context"
	"fmt"
	"strings"

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

// EstimateTokens uses tiktoken for accurate token counting.
// Falls back to character-based estimation if tiktoken is unavailable.
func EstimateTokens(s string) int {
	return agent.CountStringTokens(s)
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

// SummaryMemoryTemplate is the prompt template for generating a standalone summary.
const SummaryMemoryTemplate = `Please provide a concise summary of the following conversation. 
Focus on:
- Main topics discussed
- Key decisions or conclusions
- Important context for future reference

Keep the summary brief (under 500 words) but include all essential information.

Conversation:
%s`

// SummaryMemory generates a standalone summary of the given messages without modifying them.
// This is useful for /compact_str or when you need a summary for display without affecting history.
// If llm is nil, returns a simple text concatenation of messages.
func (c *Compactor) SummaryMemory(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Build conversation text
	var conv strings.Builder
	for _, m := range messages {
		// Skip system messages from summary
		if m.Role == "system" {
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "AI"
		} else if role == "user" {
			role = "User"
		}
		conv.WriteString(role)
		conv.WriteString(": ")
		content := m.Content
		// Truncate very long content
		if len(content) > 2000 {
			content = content[:1997] + "..."
		}
		conv.WriteString(content)
		conv.WriteString("\n\n")
	}

	if c.llm == nil {
		// Fallback: return truncated conversation text
		text := conv.String()
		if len(text) > 2000 {
			return text[:1997] + "...", nil
		}
		return text, nil
	}

	// Use LLM to generate summary
	prompt := fmt.Sprintf(SummaryMemoryTemplate, conv.String())
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "system", Content: "You are a helpful assistant that summarizes conversations accurately and concisely."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   800,
	}

	resp, err := c.llm.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm summary: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}
