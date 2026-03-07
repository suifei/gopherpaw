package memory

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/sashabaranov/go-openai"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// EmbeddingClient calls external Embedding API (OpenAI-compatible).
type EmbeddingClient struct {
	client *openai.Client
	cfg    config.MemoryConfig
	cache  *embeddingCache
}

// embeddingCache is a simple LRU cache for embeddings.
type embeddingCache struct {
	mu    sync.RWMutex
	items map[string][]float32
	keys  []string
	max   int
}

func newEmbeddingCache(maxSize int) *embeddingCache {
	if maxSize <= 0 {
		maxSize = 2000
	}
	return &embeddingCache{
		items: make(map[string][]float32),
		keys:  make([]string, 0, maxSize),
		max:   maxSize,
	}
}

func (c *embeddingCache) get(text string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[text]
	return v, ok
}

func (c *embeddingCache) set(text string, vec []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[text]; ok {
		return
	}
	if len(c.keys) >= c.max {
		evict := c.keys[0]
		c.keys = c.keys[1:]
		delete(c.items, evict)
	}
	c.items[text] = vec
	c.keys = append(c.keys, text)
}

// NewEmbeddingClient creates an EmbeddingClient from config.
// Returns nil if API key is not configured.
func NewEmbeddingClient(cfg config.MemoryConfig) *EmbeddingClient {
	apiKey := cfg.EmbeddingAPIKey
	if apiKey == "" {
		return nil
	}
	clientCfg := openai.DefaultConfig(apiKey)
	if cfg.EmbeddingBaseURL != "" {
		clientCfg.BaseURL = cfg.EmbeddingBaseURL
	}
	maxCache := cfg.EmbeddingMaxCache
	if maxCache <= 0 {
		maxCache = 2000
	}
	return &EmbeddingClient{
		client: openai.NewClientWithConfig(clientCfg),
		cfg:    cfg,
		cache:  newEmbeddingCache(maxCache),
	}
}

// Embed returns the embedding vector for text. Uses cache when available.
func (e *EmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if e == nil || text == "" {
		return nil, fmt.Errorf("embedding client not configured or empty text")
	}
	if vec, ok := e.cache.get(text); ok {
		return vec, nil
	}
	model := openai.EmbeddingModel(e.cfg.EmbeddingModel)
	if model == "" {
		model = openai.AdaEmbeddingV2
	}
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: model,
	}
	if e.cfg.EmbeddingDimensions > 0 {
		req.Dimensions = e.cfg.EmbeddingDimensions
	}
	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create embeddings: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	vec := resp.Data[0].Embedding
	e.cache.set(text, vec)
	logger.L().Debug("embedding computed", zap.Int("dims", len(vec)))
	return vec, nil
}

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector length mismatch: %d vs %d", len(a), len(b))
	}
	if len(a) == 0 {
		return 0, nil
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0, nil
	}
	return dot / denom, nil
}
