package memory

import (
	"context"
	"testing"
	"time"
)

func BenchmarkVectorStore_Save_Single(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	chunk := &Chunk{
		ID:        "test-id",
		Content:   "test content",
		Timestamp: time.Now().Unix(),
		Vector:    []float32{0.1, 0.2, 0.3},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.Save("chat-1", chunk)
	}
}

func BenchmarkVectorStore_Save_Batch(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	chunks := make([]*Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.SaveBatch("chat-1", chunks)
	}
}

func BenchmarkVectorStore_Load_FromMemory(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	chunks := make([]*Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		}
	}
	vs.SaveBatch("chat-1", chunks)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.Load(ctx, "chat-1")
	}
}

func BenchmarkVectorStore_Load_FromDisk(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})

	chunks := make([]*Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		}
	}
	vs.SaveBatch("chat-1", chunks)
	vs.Flush(context.Background())
	vs.Close()

	vs2 := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs2.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs2.Load(ctx, "chat-1")
	}
}

func BenchmarkVectorStore_Flush_Small(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	for i := 0; i < 10; i++ {
		vs.Save("chat-1", &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		})
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.Flush(ctx)
	}
}

func BenchmarkVectorStore_Flush_Large(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	for i := 0; i < 1000; i++ {
		vs.Save("chat-1", &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		})
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.Flush(ctx)
	}
}

func BenchmarkVectorStore_Stats(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	for i := 0; i < 100; i++ {
		vs.Save("chat-1", &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.Stats()
	}
}

func BenchmarkVectorStore_Delete(b *testing.B) {
	tmp := b.TempDir()
	vs := NewVectorStore(VectorStoreConfig{WorkingDir: tmp})
	defer vs.Close()

	chunks := make([]*Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = &Chunk{
			ID:        "test-id",
			Content:   "test content",
			Timestamp: time.Now().Unix(),
			Vector:    []float32{0.1, 0.2, 0.3},
		}
	}
	vs.SaveBatch("chat-1", chunks)
	vs.Flush(context.Background())

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.Delete(ctx, "chat-1")
		vs.SaveBatch("chat-1", chunks)
		vs.Flush(ctx)
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	vec1 := make([]float32, 100)
	for i := range vec1 {
		vec1[i] = 0.1
	}
	vec2 := make([]float32, 100)
	for i := range vec2 {
		vec2[i] = 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

func BenchmarkCosineSimilarity_Small(b *testing.B) {
	vec1 := []float32{0.1, 0.2, 0.3}
	vec2 := []float32{0.4, 0.5, 0.6}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

func BenchmarkCosineSimilarity_Medium(b *testing.B) {
	vec1 := make([]float32, 500)
	for i := range vec1 {
		vec1[i] = 0.1
	}
	vec2 := make([]float32, 500)
	for i := range vec2 {
		vec2[i] = 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

func BenchmarkCosineSimilarity_Large(b *testing.B) {
	vec1 := make([]float32, 2000)
	for i := range vec1 {
		vec1[i] = 0.1
	}
	vec2 := make([]float32, 2000)
	for i := range vec2 {
		vec2[i] = 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}
