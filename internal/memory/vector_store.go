package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// VectorStore provides persistent storage for embedding vectors.
// Vectors are stored as JSON files, one per chat, with automatic periodic saving.
type VectorStore struct {
	mu         sync.RWMutex
	workingDir string
	dirty      map[string]bool // chatID -> needs save
	chunks     map[string]map[string]*Chunk
	saveTimer  *time.Timer
	savePeriod time.Duration
	closed     bool
}

// VectorStoreConfig configures the VectorStore.
type VectorStoreConfig struct {
	WorkingDir string
	SavePeriod time.Duration // How often to auto-save dirty chunks (default: 30s)
}

// NewVectorStore creates a persistent vector store.
func NewVectorStore(cfg VectorStoreConfig) *VectorStore {
	wd := cfg.WorkingDir
	if wd == "" {
		wd = "."
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		abs = wd
	}

	period := cfg.SavePeriod
	if period <= 0 {
		period = 30 * time.Second
	}

	vs := &VectorStore{
		workingDir: abs,
		dirty:      make(map[string]bool),
		chunks:     make(map[string]map[string]*Chunk),
		savePeriod: period,
	}

	// Start periodic save goroutine
	vs.startPeriodicSave()

	return vs
}

// vectorDir returns the directory for vector index files.
func (vs *VectorStore) vectorDir() string {
	return filepath.Join(vs.workingDir, "data", "vectors")
}

// vectorFile returns the path to the vector index file for a chat.
func (vs *VectorStore) vectorFile(chatID string) string {
	return filepath.Join(vs.vectorDir(), chatID+".json")
}

// storedVectorIndex is the JSON structure for persisted vectors.
type storedVectorIndex struct {
	ChatID    string         `json:"chat_id"`
	UpdatedAt int64          `json:"updated_at"`
	Chunks    []*storedChunk `json:"chunks"`
}

type storedChunk struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp int64     `json:"timestamp"`
	Vector    []float32 `json:"vector,omitempty"`
}

// Save persists a chunk to the vector store.
func (vs *VectorStore) Save(chatID string, chunk *Chunk) {
	if chatID == "" || chunk == nil {
		return
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return
	}

	if vs.chunks[chatID] == nil {
		vs.chunks[chatID] = make(map[string]*Chunk)
	}
	vs.chunks[chatID][chunk.ID] = chunk
	vs.dirty[chatID] = true
}

// SaveBatch persists multiple chunks.
func (vs *VectorStore) SaveBatch(chatID string, chunks []*Chunk) {
	if chatID == "" || len(chunks) == 0 {
		return
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.closed {
		return
	}

	if vs.chunks[chatID] == nil {
		vs.chunks[chatID] = make(map[string]*Chunk)
	}
	for _, c := range chunks {
		if c != nil {
			vs.chunks[chatID][c.ID] = c
		}
	}
	vs.dirty[chatID] = true
}

// Load retrieves all chunks for a chat.
func (vs *VectorStore) Load(ctx context.Context, chatID string) ([]*Chunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if chatID == "" {
		return nil, fmt.Errorf("chatID cannot be empty")
	}

	vs.mu.RLock()
	if chunks, ok := vs.chunks[chatID]; ok && len(chunks) > 0 {
		result := make([]*Chunk, 0, len(chunks))
		for _, c := range chunks {
			result = append(result, c)
		}
		vs.mu.RUnlock()
		return result, nil
	}
	vs.mu.RUnlock()

	// Try loading from disk
	return vs.loadFromDisk(ctx, chatID)
}

// loadFromDisk loads chunks from the vector index file.
func (vs *VectorStore) loadFromDisk(ctx context.Context, chatID string) ([]*Chunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path := vs.vectorFile(chatID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read vector index: %w", err)
	}

	var index storedVectorIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal vector index: %w", err)
	}

	// Convert to Chunk objects and cache in memory
	chunks := make([]*Chunk, len(index.Chunks))
	vs.mu.Lock()
	if vs.chunks[chatID] == nil {
		vs.chunks[chatID] = make(map[string]*Chunk)
	}
	for i, sc := range index.Chunks {
		chunk := &Chunk{
			ID:        sc.ID,
			Content:   sc.Content,
			Timestamp: sc.Timestamp,
			Vector:    sc.Vector,
		}
		chunks[i] = chunk
		vs.chunks[chatID][chunk.ID] = chunk
	}
	vs.mu.Unlock()

	return chunks, nil
}

// saveToDisk persists chunks for a chat to disk.
func (vs *VectorStore) saveToDisk(chatID string) error {
	vs.mu.RLock()
	chunks := vs.chunks[chatID]
	vs.mu.RUnlock()

	if len(chunks) == 0 {
		return nil
	}

	index := storedVectorIndex{
		ChatID:    chatID,
		UpdatedAt: time.Now().Unix(),
		Chunks:    make([]*storedChunk, 0, len(chunks)),
	}

	for _, c := range chunks {
		index.Chunks = append(index.Chunks, &storedChunk{
			ID:        c.ID,
			Content:   c.Content,
			Timestamp: c.Timestamp,
			Vector:    c.Vector,
		})
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal vector index: %w", err)
	}

	path := vs.vectorFile(chatID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write vector index: %w", err)
	}

	return nil
}

// Flush immediately saves all dirty chunks to disk.
func (vs *VectorStore) Flush(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	vs.mu.Lock()
	dirtyChatIDs := make([]string, 0, len(vs.dirty))
	for chatID := range vs.dirty {
		dirtyChatIDs = append(dirtyChatIDs, chatID)
	}
	for _, chatID := range dirtyChatIDs {
		delete(vs.dirty, chatID)
	}
	vs.mu.Unlock()

	var lastErr error
	for _, chatID := range dirtyChatIDs {
		if err := vs.saveToDisk(chatID); err != nil {
			logger.L().Warn("save vector index", zap.String("chatID", chatID), zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}

// startPeriodicSave starts the background save goroutine.
func (vs *VectorStore) startPeriodicSave() {
	vs.saveTimer = time.AfterFunc(vs.savePeriod, func() {
		vs.mu.Lock()
		if vs.closed {
			vs.mu.Unlock()
			return
		}
		vs.mu.Unlock()

		if err := vs.Flush(context.Background()); err != nil {
			logger.L().Warn("periodic vector flush", zap.Error(err))
		}

		vs.mu.Lock()
		if !vs.closed {
			vs.saveTimer.Reset(vs.savePeriod)
		}
		vs.mu.Unlock()
	})
}

// Close stops the periodic save and flushes all pending data.
func (vs *VectorStore) Close() error {
	vs.mu.Lock()
	vs.closed = true
	if vs.saveTimer != nil {
		vs.saveTimer.Stop()
	}
	vs.mu.Unlock()

	return vs.Flush(context.Background())
}

// Delete removes vector data for a chat.
func (vs *VectorStore) Delete(ctx context.Context, chatID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}

	vs.mu.Lock()
	delete(vs.chunks, chatID)
	delete(vs.dirty, chatID)
	vs.mu.Unlock()

	path := vs.vectorFile(chatID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove vector index: %w", err)
	}
	return nil
}

// Stats returns statistics about the vector store.
func (vs *VectorStore) Stats() VectorStoreStats {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	stats := VectorStoreStats{
		ChatCount: len(vs.chunks),
	}
	for _, chunks := range vs.chunks {
		stats.ChunkCount += len(chunks)
		for _, c := range chunks {
			if len(c.Vector) > 0 {
				stats.VectorCount++
			}
		}
	}
	return stats
}

// VectorStoreStats holds statistics about the vector store.
type VectorStoreStats struct {
	ChatCount   int
	ChunkCount  int
	VectorCount int
}
