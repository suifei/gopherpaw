// Package memory provides MemoryStore implementations for conversation history.
package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

// InMemoryStore implements agent.MemoryStore using an in-memory map.
// Safe for concurrent access.
type InMemoryStore struct {
	mu      sync.RWMutex
	history map[string][]storedMessage
	cfg     config.MemoryConfig
	maxHist int
}

// storedMessage is used for both in-memory and file persistence.
type storedMessage struct {
	Msg       agent.Message `json:"msg"`
	Timestamp int64         `json:"timestamp"`
}

// New creates a MemoryStore based on config. Backend "memory" uses in-memory storage.
func New(cfg config.MemoryConfig) agent.MemoryStore {
	maxHist := cfg.MaxHistory
	if maxHist <= 0 {
		maxHist = 50
	}
	return &InMemoryStore{
		history: make(map[string][]storedMessage),
		cfg:     cfg,
		maxHist: maxHist,
	}
}

// Save stores a message in the conversation history.
func (s *InMemoryStore) Save(ctx context.Context, chatID string, msg agent.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.history[chatID]
	ts := time.Now().Unix()
	list = append(list, storedMessage{Msg: msg, Timestamp: ts})
	if len(list) > s.maxHist {
		list = list[len(list)-s.maxHist:]
	}
	s.history[chatID] = list
	return nil
}

// Load retrieves recent conversation history.
func (s *InMemoryStore) Load(ctx context.Context, chatID string, limit int) ([]agent.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if chatID == "" {
		return nil, fmt.Errorf("chatID cannot be empty")
	}
	if limit <= 0 {
		limit = s.maxHist
	}
	s.mu.RLock()
	list := s.history[chatID]
	s.mu.RUnlock()
	if len(list) == 0 {
		return []agent.Message{}, nil
	}
	start := len(list) - limit
	if start < 0 {
		start = 0
	}
	out := make([]agent.Message, len(list)-start)
	for i, sm := range list[start:] {
		out[i] = sm.Msg
	}
	return out, nil
}

// Search performs keyword search across memory. Uses simple substring matching.
// Returns results sorted by score (keyword match count) and timestamp.
func (s *InMemoryStore) Search(ctx context.Context, chatID string, query string, topK int) ([]agent.MemoryResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if chatID == "" {
		return nil, fmt.Errorf("chatID cannot be empty")
	}
	if query == "" {
		return []agent.MemoryResult{}, nil
	}
	if topK <= 0 {
		topK = 5
	}
	s.mu.RLock()
	list := s.history[chatID]
	s.mu.RUnlock()
	if len(list) == 0 {
		return []agent.MemoryResult{}, nil
	}
	queryLower := strings.ToLower(query)
	keywords := strings.Fields(queryLower)
	var results []agent.MemoryResult
	for _, sm := range list {
		content := sm.Msg.Content
		if content == "" {
			continue
		}
		contentLower := strings.ToLower(content)
		score := 0.0
		for _, kw := range keywords {
			if strings.Contains(contentLower, kw) {
				score += 1.0
			}
		}
		if score > 0 {
			results = append(results, agent.MemoryResult{
				Content:   content,
				Score:     score / float64(len(keywords)),
				Timestamp: sm.Timestamp,
			})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Timestamp > results[j].Timestamp
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// Compact truncates old history to maxHistory. In-memory store simply keeps the last N messages.
func (s *InMemoryStore) Compact(ctx context.Context, chatID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.history[chatID]
	if len(list) <= s.maxHist {
		return nil
	}
	list = list[len(list)-s.maxHist:]
	s.history[chatID] = list
	return nil
}

// GetCompactSummary returns the current compact summary. In-memory store has no summary.
func (s *InMemoryStore) GetCompactSummary(ctx context.Context, chatID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if chatID == "" {
		return "", fmt.Errorf("chatID cannot be empty")
	}
	return "", nil
}

// SaveLongTerm persists content to long-term memory. In-memory store is a no-op.
func (s *InMemoryStore) SaveLongTerm(ctx context.Context, chatID string, content string, category string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}
	return nil
}

// LoadLongTerm loads long-term memory. In-memory store returns empty.
func (s *InMemoryStore) LoadLongTerm(ctx context.Context, chatID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if chatID == "" {
		return "", fmt.Errorf("chatID cannot be empty")
	}
	return "", nil
}

// SummaryMemory generates a simple text summary of messages (no LLM, just concatenation).
// Implements agent.MemorySummarizer interface.
func (s *InMemoryStore) SummaryMemory(ctx context.Context, messages []agent.Message) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(messages) == 0 {
		return "", nil
	}
	var sb strings.Builder
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "AI"
		} else if role == "user" {
			role = "User"
		}
		sb.WriteString(role)
		sb.WriteString(": ")
		content := m.Content
		if len(content) > 500 {
			content = content[:497] + "..."
		}
		sb.WriteString(content)
		sb.WriteString("\n")
	}
	text := sb.String()
	if len(text) > 2000 {
		return text[:1997] + "...", nil
	}
	return text, nil
}

// Ensure InMemoryStore implements MemorySummarizer.
var _ agent.MemorySummarizer = (*InMemoryStore)(nil)
