// Package agent provides token counting and message utility functions.
package agent

import (
	"encoding/json"
	"strings"
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides token counting using tiktoken for accurate token estimation.
// It lazily initializes the encoder on first use and caches it for performance.
type TokenCounter struct {
	mu      sync.Once
	encoder *tiktoken.Tiktoken
	model   string
}

// defaultEncoder is the package-level tiktoken encoder, initialized lazily.
var (
	defaultEncoderMu   sync.Once
	defaultEncoder     *tiktoken.Tiktoken
	defaultEncoderInit bool
)

// initDefaultEncoder initializes the default tiktoken encoder.
// Uses cl100k_base encoding (GPT-4, GPT-3.5-turbo, text-embedding-ada-002).
func initDefaultEncoder() {
	defaultEncoderMu.Do(func() {
		enc, err := tiktoken.GetEncoding("cl100k_base")
		if err == nil {
			defaultEncoder = enc
			defaultEncoderInit = true
		}
	})
}

// NewTokenCounter creates a TokenCounter for a specific model.
// Supported models include gpt-4, gpt-3.5-turbo, claude, etc.
// Falls back to cl100k_base encoding if model-specific encoding is not available.
func NewTokenCounter(model string) *TokenCounter {
	tc := &TokenCounter{model: model}
	return tc
}

// getEncoder returns the tiktoken encoder, initializing if needed.
func (tc *TokenCounter) getEncoder() *tiktoken.Tiktoken {
	tc.mu.Do(func() {
		if tc.model != "" {
			// Try model-specific encoding first
			enc, err := tiktoken.EncodingForModel(tc.model)
			if err == nil {
				tc.encoder = enc
				return
			}
		}
		// Fall back to default cl100k_base
		initDefaultEncoder()
		tc.encoder = defaultEncoder
	})
	return tc.encoder
}

// Count returns the token count for a string using tiktoken.
// Falls back to character-based estimation if encoder is unavailable.
func (tc *TokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}
	enc := tc.getEncoder()
	if enc != nil {
		return len(enc.Encode(text, nil, nil))
	}
	// Fallback: character-based estimation
	return estimateFallback(text)
}

// CountMessages returns the token count for a list of messages.
func (tc *TokenCounter) CountMessages(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += tc.Count(extractMessageText(m))
		total += 4 // per-message overhead (role, name, etc.)
	}
	return total
}

// estimateFallback provides character-based estimation when tiktoken is unavailable.
// Uses different ratios for CJK vs ASCII characters.
func estimateFallback(text string) int {
	cjkCount := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			cjkCount++
		}
	}
	// CJK characters are roughly 1.5-2 tokens each, ASCII roughly 4 chars per token
	asciiCount := len(text) - cjkCount*3 // rough byte adjustment for CJK
	if asciiCount < 0 {
		asciiCount = 0
	}
	return asciiCount/4 + cjkCount*2
}

// CountMessageTokens returns the token count for a list of messages using tiktoken.
// This is the primary function for accurate token counting.
func CountMessageTokens(messages []Message) int {
	initDefaultEncoder()
	total := 0
	for _, m := range messages {
		text := extractMessageText(m)
		if defaultEncoder != nil {
			total += len(defaultEncoder.Encode(text, nil, nil))
		} else {
			total += estimateFallback(text)
		}
		total += 4 // per-message overhead
	}
	return total
}

// CountStringTokens returns the token count for a string using tiktoken.
func CountStringTokens(text string) int {
	if text == "" {
		return 0
	}
	initDefaultEncoder()
	if defaultEncoder != nil {
		return len(defaultEncoder.Encode(text, nil, nil))
	}
	return estimateFallback(text)
}

// CountStringTokensForModel returns the token count using model-specific encoding.
func CountStringTokensForModel(text, model string) int {
	if text == "" {
		return 0
	}
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		// Fall back to cl100k_base
		initDefaultEncoder()
		if defaultEncoder != nil {
			return len(defaultEncoder.Encode(text, nil, nil))
		}
		return estimateFallback(text)
	}
	return len(enc.Encode(text, nil, nil))
}

// SafeCountMessageTokens is a safe wrapper that never panics.
func SafeCountMessageTokens(messages []Message) int {
	defer func() { recover() }()
	return CountMessageTokens(messages)
}

// extractMessageText extracts all text content from a message,
// including tool call arguments and tool results.
func extractMessageText(m Message) string {
	var sb strings.Builder
	sb.WriteString(m.Content)

	for _, tc := range m.ToolCalls {
		sb.WriteString(" ")
		sb.WriteString(tc.Arguments)
	}

	return sb.String()
}

// SanitizeToolMessages validates and repairs tool message sequences.
// Ensures tool_call and tool_result messages are properly paired.
func SanitizeToolMessages(messages []Message) []Message {
	messages = removeInvalidToolBlocks(messages)
	messages = dedupToolCalls(messages)
	messages = reorderToolResults(messages)
	messages = removeUnpairedToolMessages(messages)
	return messages
}

// CheckValidMessages returns true if all tool_calls have matching tool results.
func CheckValidMessages(messages []Message) bool {
	useIDs := make(map[string]bool)
	resultIDs := make(map[string]bool)

	for _, m := range messages {
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				useIDs[tc.ID] = true
			}
		}
		if m.Role == "tool" && m.ToolCallID != "" {
			resultIDs[m.ToolCallID] = true
		}
	}

	if len(useIDs) != len(resultIDs) {
		return false
	}
	for id := range useIDs {
		if !resultIDs[id] {
			return false
		}
	}
	return true
}

// removeInvalidToolBlocks removes tool calls with empty ID or name.
func removeInvalidToolBlocks(messages []Message) []Message {
	result := make([]Message, 0, len(messages))
	for _, m := range messages {
		if len(m.ToolCalls) > 0 {
			valid := make([]ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				if tc.ID != "" && tc.Name != "" {
					valid = append(valid, tc)
				}
			}
			m.ToolCalls = valid
		}
		if m.Role == "tool" && m.ToolCallID == "" {
			continue
		}
		result = append(result, m)
	}
	return result
}

// dedupToolCalls removes duplicate tool calls with the same ID.
func dedupToolCalls(messages []Message) []Message {
	seen := make(map[string]bool)
	result := make([]Message, 0, len(messages))

	for _, m := range messages {
		if len(m.ToolCalls) > 0 {
			unique := make([]ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				if !seen[tc.ID] {
					seen[tc.ID] = true
					unique = append(unique, tc)
				}
			}
			m.ToolCalls = unique
		}
		result = append(result, m)
	}
	return result
}

// reorderToolResults ensures tool result messages immediately follow
// the assistant message containing the corresponding tool call.
func reorderToolResults(messages []Message) []Message {
	resultMap := make(map[string]Message)
	var nonResults []Message

	for _, m := range messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			resultMap[m.ToolCallID] = m
		} else {
			nonResults = append(nonResults, m)
		}
	}

	if len(resultMap) == 0 {
		return messages
	}

	result := make([]Message, 0, len(messages))
	for _, m := range nonResults {
		result = append(result, m)
		for _, tc := range m.ToolCalls {
			if toolResult, ok := resultMap[tc.ID]; ok {
				result = append(result, toolResult)
				delete(resultMap, tc.ID)
			}
		}
	}

	for _, m := range resultMap {
		result = append(result, m)
	}
	return result
}

// removeUnpairedToolMessages removes tool calls without results and vice versa.
func removeUnpairedToolMessages(messages []Message) []Message {
	useIDs := make(map[string]bool)
	resultIDs := make(map[string]bool)

	for _, m := range messages {
		for _, tc := range m.ToolCalls {
			useIDs[tc.ID] = true
		}
		if m.Role == "tool" && m.ToolCallID != "" {
			resultIDs[m.ToolCallID] = true
		}
	}

	result := make([]Message, 0, len(messages))
	for _, m := range messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			if !useIDs[m.ToolCallID] {
				continue
			}
		}
		if len(m.ToolCalls) > 0 {
			paired := make([]ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				if resultIDs[tc.ID] {
					paired = append(paired, tc)
				}
			}
			m.ToolCalls = paired
		}
		result = append(result, m)
	}
	return result
}

// TruncateText truncates text keeping the beginning and end.
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	if maxLength <= 20 {
		return text[:maxLength]
	}
	keep := (maxLength - 10) / 2
	return text[:keep] + "\n...[truncated]...\n" + text[len(text)-keep:]
}

// RepairEmptyToolInputs attempts to parse raw_input JSON when input is empty.
func RepairEmptyToolInputs(messages []Message) []Message {
	for i := range messages {
		for j := range messages[i].ToolCalls {
			tc := &messages[i].ToolCalls[j]
			if tc.Arguments == "" || tc.Arguments == "{}" {
				continue
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(tc.Arguments), &parsed); err != nil {
				tc.Arguments = "{}"
			}
		}
	}
	return messages
}
