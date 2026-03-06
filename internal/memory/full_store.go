package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// FullMemoryStore implements the complete MemoryStore with file persistence,
// hybrid search, compaction, and long-term memory.
type FullMemoryStore struct {
	mu        sync.RWMutex
	cfg       config.MemoryConfig
	shortTerm map[string][]storedMessage
	summary   map[string]string
	maxHist   int
	fileStore *FileStore
	hybrid    *HybridSearcher
	compactor *Compactor
}

// NewFullMemoryStore creates a full-featured memory store.
// llm may be nil; compaction will fall back to truncation.
func NewFullMemoryStore(cfg config.MemoryConfig, llm agent.LLMProvider) agent.MemoryStore {
	maxHist := cfg.MaxHistory
	if maxHist <= 0 {
		maxHist = 50
	}
	embed := NewEmbeddingClient(cfg)
	hybrid := NewHybridSearcher(embed, &struct {
		VectorWeight  float64
		BM25Weight    float64
		CandidateMul  int
		MaxCandidates int
	}{
		VectorWeight:  0.7,
		BM25Weight:    0.3,
		CandidateMul:  3,
		MaxCandidates: 200,
	})
	return &FullMemoryStore{
		cfg:       cfg,
		shortTerm: make(map[string][]storedMessage),
		summary:   make(map[string]string),
		maxHist:   maxHist,
		fileStore: NewFileStore(cfg),
		hybrid:    hybrid,
		compactor: NewCompactor(llm, cfg),
	}
}

// Save stores a message and indexes it for search.
func (s *FullMemoryStore) Save(ctx context.Context, chatID string, msg agent.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}
	ts := time.Now().Unix()
	s.mu.Lock()
	list := s.shortTerm[chatID]
	list = append(list, storedMessage{Msg: msg, Timestamp: ts})
	if len(list) > s.maxHist {
		list = list[len(list)-s.maxHist:]
	}
	s.shortTerm[chatID] = list
	s.mu.Unlock()
	chunk := &Chunk{
		ID:        fmt.Sprintf("%s-%d-%d", chatID, ts, len(list)-1),
		Content:   msg.Content,
		Timestamp: ts,
	}
	if s.hybrid.embed != nil {
		vec, err := s.hybrid.embed.Embed(ctx, msg.Content)
		if err == nil {
			chunk.Vector = vec
		}
	}
	s.hybrid.IndexChunk(chatID, chunk)
	hist := make([]storedMessage, len(list))
	copy(hist, list)
	if err := s.fileStore.SaveHistory(ctx, chatID, hist); err != nil {
		logger.L().Warn("save history to file", zap.Error(err))
	}
	return nil
}

// Load retrieves recent conversation history.
func (s *FullMemoryStore) Load(ctx context.Context, chatID string, limit int) ([]agent.Message, error) {
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
	list := s.shortTerm[chatID]
	s.mu.RUnlock()
	if len(list) == 0 {
		hist, err := s.fileStore.LoadHistory(ctx, chatID)
		if err != nil {
			return nil, err
		}
		for _, h := range hist {
			list = append(list, storedMessage{Msg: h.Msg, Timestamp: h.Timestamp})
		}
		s.mu.Lock()
		s.shortTerm[chatID] = list
		s.mu.Unlock()
	}
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

// Search performs hybrid search.
func (s *FullMemoryStore) Search(ctx context.Context, chatID string, query string, topK int) ([]agent.MemoryResult, error) {
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
	return s.hybrid.Search(ctx, chatID, query, topK)
}

// Compact compresses history with LLM summary when available.
func (s *FullMemoryStore) Compact(ctx context.Context, chatID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.shortTerm[chatID]
	if len(list) <= s.maxHist {
		return nil
	}
	msgs := make([]agent.Message, len(list))
	for i, sm := range list {
		msgs[i] = sm.Msg
	}
	prevSummary := s.summary[chatID]
	summary, keep, err := s.compactor.CompactWithLLM(ctx, msgs, prevSummary)
	if err != nil {
		return err
	}
	if summary != "" {
		s.summary[chatID] = summary
	}
	var newList []storedMessage
	for _, m := range keep {
		ts := time.Now().Unix()
		newList = append(newList, storedMessage{Msg: m, Timestamp: ts})
	}
	s.shortTerm[chatID] = newList
	if err := s.fileStore.SaveHistory(ctx, chatID, newList); err != nil {
		logger.L().Warn("save compacted history", zap.Error(err))
	}
	return nil
}

// GetCompactSummary returns the current compact summary.
func (s *FullMemoryStore) GetCompactSummary(ctx context.Context, chatID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if chatID == "" {
		return "", fmt.Errorf("chatID cannot be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.summary[chatID], nil
}

// SaveLongTerm persists content to MEMORY.md or memory/YYYY-MM-DD.md.
func (s *FullMemoryStore) SaveLongTerm(ctx context.Context, chatID string, content string, category string) error {
	return s.fileStore.SaveLongTerm(ctx, chatID, content, category)
}

// LoadLongTerm loads long-term memory content.
func (s *FullMemoryStore) LoadLongTerm(ctx context.Context, chatID string) (string, error) {
	return s.fileStore.LoadLongTerm(ctx, chatID)
}

// SummaryMemory generates a standalone summary of the given messages.
// Implements agent.MemorySummarizer interface.
func (s *FullMemoryStore) SummaryMemory(ctx context.Context, messages []agent.Message) (string, error) {
	return s.compactor.SummaryMemory(ctx, messages)
}

// Ensure FullMemoryStore implements MemorySummarizer.
var _ agent.MemorySummarizer = (*FullMemoryStore)(nil)
