package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/suifei/gopherpaw/internal/agent"
)

const (
	defaultVectorWeight  = 0.7
	defaultBM25Weight    = 0.3
	defaultCandidateMul  = 3
	defaultMaxCandidates = 200
)

// HybridSearcher combines vector search and BM25 for semantic + full-text search.
type HybridSearcher struct {
	mu          sync.RWMutex
	embed       *EmbeddingClient
	bm25        *BM25
	chunks      map[string]map[string]*Chunk // chatID -> chunkID -> Chunk
	vectorStore *VectorStore                 // Optional persistent vector storage
	vectorW     float64
	bm25W       float64
	candMul     int
	maxCand     int
}

// HybridSearchConfig configures the HybridSearcher.
type HybridSearchConfig struct {
	VectorWeight  float64
	BM25Weight    float64
	CandidateMul  int
	MaxCandidates int
	WorkingDir    string // For vector persistence
}

// NewHybridSearcher creates a HybridSearcher.
func NewHybridSearcher(embed *EmbeddingClient, cfg *struct {
	VectorWeight  float64
	BM25Weight    float64
	CandidateMul  int
	MaxCandidates int
}) *HybridSearcher {
	vw := defaultVectorWeight
	bw := defaultBM25Weight
	cm := defaultCandidateMul
	mc := defaultMaxCandidates
	if cfg != nil {
		if cfg.VectorWeight > 0 {
			vw = cfg.VectorWeight
		}
		if cfg.BM25Weight > 0 {
			bw = cfg.BM25Weight
		}
		if cfg.CandidateMul > 0 {
			cm = cfg.CandidateMul
		}
		if cfg.MaxCandidates > 0 {
			mc = cfg.MaxCandidates
		}
	}
	return &HybridSearcher{
		embed:   embed,
		bm25:    NewBM25(),
		chunks:  make(map[string]map[string]*Chunk),
		vectorW: vw,
		bm25W:   bw,
		candMul: cm,
		maxCand: mc,
	}
}

// NewHybridSearcherWithPersistence creates a HybridSearcher with persistent vector storage.
func NewHybridSearcherWithPersistence(embed *EmbeddingClient, cfg *HybridSearchConfig) *HybridSearcher {
	vw := defaultVectorWeight
	bw := defaultBM25Weight
	cm := defaultCandidateMul
	mc := defaultMaxCandidates
	wd := ""
	if cfg != nil {
		if cfg.VectorWeight > 0 {
			vw = cfg.VectorWeight
		}
		if cfg.BM25Weight > 0 {
			bw = cfg.BM25Weight
		}
		if cfg.CandidateMul > 0 {
			cm = cfg.CandidateMul
		}
		if cfg.MaxCandidates > 0 {
			mc = cfg.MaxCandidates
		}
		wd = cfg.WorkingDir
	}

	var vs *VectorStore
	if wd != "" {
		vs = NewVectorStore(VectorStoreConfig{
			WorkingDir: wd,
		})
	}

	return &HybridSearcher{
		embed:       embed,
		bm25:        NewBM25(),
		chunks:      make(map[string]map[string]*Chunk),
		vectorStore: vs,
		vectorW:     vw,
		bm25W:       bw,
		candMul:     cm,
		maxCand:     mc,
	}
}

// IndexChunk adds or updates a chunk for a chat.
func (h *HybridSearcher) IndexChunk(chatID string, c *Chunk) {
	if chatID == "" || c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.chunks[chatID] == nil {
		h.chunks[chatID] = make(map[string]*Chunk)
	}
	h.chunks[chatID][c.ID] = c
	// Update BM25 corpus for better IDF calculation
	h.bm25.AddDocument(c.Content)

	// Persist to vector store if available
	if h.vectorStore != nil {
		h.vectorStore.Save(chatID, c)
	}
}

// IndexChunks adds multiple chunks for a chat.
func (h *HybridSearcher) IndexChunks(chatID string, chunks []*Chunk) {
	for _, c := range chunks {
		h.IndexChunk(chatID, c)
	}
}

// LoadFromPersistence loads chunks from persistent storage for a chat.
// This should be called when initializing to restore previously saved vectors.
func (h *HybridSearcher) LoadFromPersistence(ctx context.Context, chatID string) error {
	if h.vectorStore == nil {
		return nil
	}

	chunks, err := h.vectorStore.Load(ctx, chatID)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.chunks[chatID] == nil {
		h.chunks[chatID] = make(map[string]*Chunk)
	}

	for _, c := range chunks {
		h.chunks[chatID][c.ID] = c
		h.bm25.AddDocument(c.Content)
	}

	return nil
}

// Flush persists all pending chunks to disk.
func (h *HybridSearcher) Flush(ctx context.Context) error {
	if h.vectorStore == nil {
		return nil
	}
	return h.vectorStore.Flush(ctx)
}

// Close cleans up resources and persists pending data.
func (h *HybridSearcher) Close() error {
	if h.vectorStore == nil {
		return nil
	}
	return h.vectorStore.Close()
}

// Search performs hybrid search: vector (if enabled) + BM25, weighted fusion.
func (h *HybridSearcher) Search(ctx context.Context, chatID string, query string, topK int) ([]agent.MemoryResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if chatID == "" || query == "" {
		return []agent.MemoryResult{}, nil
	}
	if topK <= 0 {
		topK = 5
	}

	h.mu.RLock()
	chatChunks := h.chunks[chatID]
	h.mu.RUnlock()

	// If no chunks in memory, try loading from persistence
	if len(chatChunks) == 0 && h.vectorStore != nil {
		if err := h.LoadFromPersistence(ctx, chatID); err == nil {
			h.mu.RLock()
			chatChunks = h.chunks[chatID]
			h.mu.RUnlock()
		}
	}

	if len(chatChunks) == 0 {
		return []agent.MemoryResult{}, nil
	}

	chunkList := make([]*Chunk, 0, len(chatChunks))
	for _, c := range chatChunks {
		chunkList = append(chunkList, c)
	}

	candLimit := topK * h.candMul
	if candLimit > h.maxCand {
		candLimit = h.maxCand
	}

	scores := make(map[string]float64)
	for _, c := range chunkList {
		bm25Score := h.bm25.Score(query, c.Content)
		var vecScore float64
		if h.embed != nil && len(c.Vector) > 0 {
			queryVec, err := h.embed.Embed(ctx, query)
			if err == nil {
				sim, err := CosineSimilarity(queryVec, c.Vector)
				if err == nil {
					vecScore = max(0, sim)
				}
			}
		}
		hybrid := h.vectorW*vecScore + h.bm25W*normalizeBM25(bm25Score)
		if h.embed == nil {
			hybrid = bm25Score
		}
		scores[c.ID] = hybrid
	}

	type scored struct {
		chunk *Chunk
		score float64
	}
	var scoredList []scored
	for _, c := range chunkList {
		scoredList = append(scoredList, scored{c, scores[c.ID]})
	}
	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].score > scoredList[j].score })
	if len(scoredList) > candLimit {
		scoredList = scoredList[:candLimit]
	}

	seen := make(map[string]bool)
	var results []agent.MemoryResult
	for _, s := range scoredList {
		if seen[s.chunk.ID] {
			continue
		}
		seen[s.chunk.ID] = true
		results = append(results, agent.MemoryResult{
			Content:   s.chunk.Content,
			Score:     s.score,
			Timestamp: s.chunk.Timestamp,
			ChunkID:   s.chunk.ID,
		})
		if len(results) >= topK {
			break
		}
	}
	return results, nil
}

func normalizeBM25(s float64) float64 {
	if s > 1 {
		return 1
	}
	return s
}
