package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVectorStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	vs := NewVectorStore(VectorStoreConfig{
		WorkingDir: tmpDir,
		SavePeriod: 100 * time.Millisecond,
	})
	defer vs.Close()

	ctx := context.Background()
	chatID := "test-chat"

	// Save some chunks
	chunks := []*Chunk{
		{ID: "c1", Content: "hello world", Timestamp: 100, Vector: []float32{0.1, 0.2, 0.3}},
		{ID: "c2", Content: "foo bar", Timestamp: 200, Vector: []float32{0.4, 0.5, 0.6}},
		{ID: "c3", Content: "no vector", Timestamp: 300, Vector: nil},
	}

	for _, c := range chunks {
		vs.Save(chatID, c)
	}

	// Flush to disk
	if err := vs.Flush(ctx); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "data", "vectors", chatID+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("vector index file not created")
	}

	// Create new store and load
	vs2 := NewVectorStore(VectorStoreConfig{
		WorkingDir: tmpDir,
	})
	defer vs2.Close()

	loaded, err := vs2.Load(ctx, chatID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(loaded))
	}

	// Verify content
	chunkMap := make(map[string]*Chunk)
	for _, c := range loaded {
		chunkMap[c.ID] = c
	}

	if c, ok := chunkMap["c1"]; !ok {
		t.Error("c1 not found")
	} else {
		if c.Content != "hello world" {
			t.Errorf("c1 content mismatch: %q", c.Content)
		}
		if len(c.Vector) != 3 {
			t.Errorf("c1 vector length: %d", len(c.Vector))
		}
	}

	if c, ok := chunkMap["c3"]; !ok {
		t.Error("c3 not found")
	} else {
		if len(c.Vector) != 0 {
			t.Errorf("c3 should have no vector, got %d", len(c.Vector))
		}
	}
}

func TestVectorStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	vs := NewVectorStore(VectorStoreConfig{
		WorkingDir: tmpDir,
	})
	defer vs.Close()

	ctx := context.Background()
	chatID := "delete-test"

	vs.Save(chatID, &Chunk{ID: "c1", Content: "test"})
	if err := vs.Flush(ctx); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "data", "vectors", chatID+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should exist before delete")
	}

	// Delete
	if err := vs.Delete(ctx, chatID); err != nil {
		t.Fatal(err)
	}

	// Verify file deleted
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}

	// Load should return empty
	loaded, err := vs.Load(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 chunks after delete, got %d", len(loaded))
	}
}

func TestVectorStore_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	vs := NewVectorStore(VectorStoreConfig{
		WorkingDir: tmpDir,
	})
	defer vs.Close()

	// Initial stats
	stats := vs.Stats()
	if stats.ChatCount != 0 || stats.ChunkCount != 0 {
		t.Errorf("initial stats: %+v", stats)
	}

	// Add some chunks
	vs.Save("chat1", &Chunk{ID: "c1", Content: "a", Vector: []float32{0.1}})
	vs.Save("chat1", &Chunk{ID: "c2", Content: "b", Vector: nil})
	vs.Save("chat2", &Chunk{ID: "c3", Content: "c", Vector: []float32{0.2}})

	stats = vs.Stats()
	if stats.ChatCount != 2 {
		t.Errorf("ChatCount: %d", stats.ChatCount)
	}
	if stats.ChunkCount != 3 {
		t.Errorf("ChunkCount: %d", stats.ChunkCount)
	}
	if stats.VectorCount != 2 {
		t.Errorf("VectorCount: %d", stats.VectorCount)
	}
}

func TestVectorStore_PeriodicSave(t *testing.T) {
	tmpDir := t.TempDir()
	vs := NewVectorStore(VectorStoreConfig{
		WorkingDir: tmpDir,
		SavePeriod: 50 * time.Millisecond,
	})
	defer vs.Close()

	chatID := "periodic-test"
	vs.Save(chatID, &Chunk{ID: "c1", Content: "test"})

	// Wait for periodic save
	time.Sleep(150 * time.Millisecond)

	// Verify file exists (saved by periodic timer)
	path := filepath.Join(tmpDir, "data", "vectors", chatID+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should be created by periodic save")
	}
}

func TestVectorStore_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	vs := NewVectorStore(VectorStoreConfig{
		WorkingDir: tmpDir,
	})
	defer vs.Close()

	ctx := context.Background()
	chunks, err := vs.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for nonexistent chat, got %d", len(chunks))
	}
}

func TestHybridSearcher_WithPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create searcher with persistence
	h := NewHybridSearcherWithPersistence(nil, &HybridSearchConfig{
		WorkingDir: tmpDir,
	})

	// Index some chunks
	h.IndexChunk("chat1", &Chunk{ID: "c1", Content: "hello world", Timestamp: 100})
	h.IndexChunk("chat1", &Chunk{ID: "c2", Content: "foo bar baz", Timestamp: 200})

	// Flush
	if err := h.Flush(ctx); err != nil {
		t.Fatal(err)
	}

	// Close and create new searcher
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}

	h2 := NewHybridSearcherWithPersistence(nil, &HybridSearchConfig{
		WorkingDir: tmpDir,
	})
	defer h2.Close()

	// Load from persistence and search
	results, err := h2.Search(ctx, "chat1", "hello", 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Error("expected search results after loading from persistence")
	}

	found := false
	for _, r := range results {
		if r.Content == "hello world" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'hello world' chunk")
	}
}
