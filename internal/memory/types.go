// Package memory provides MemoryStore implementations for conversation history.
package memory

// CompactConfig holds compaction parameters.
type CompactConfig struct {
	Threshold  int     // Token threshold to trigger auto-compact (default 100000)
	KeepRecent int     // Messages to keep after compact (default 3)
	Ratio      float64 // Target compression ratio 0.0-1.0 (default 0.7)
}

// Chunk represents an indexed memory chunk for hybrid search.
type Chunk struct {
	ID        string
	Content   string
	Timestamp int64
	Vector    []float32 // Embedding vector (nil if not embedded)
}
