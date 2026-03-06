package memory

import (
	"strings"
	"testing"
)

func BenchmarkBM25_Score_Simple(b *testing.B) {
	bm25 := NewBM25()
	query := "test"
	content := "test content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.Score(query, content)
	}
}

func BenchmarkBM25_Score_WithCorpus(b *testing.B) {
	bm25 := NewBM25()
	for i := 0; i < 100; i++ {
		bm25.AddDocument(strings.Repeat("document ", 50))
	}

	query := "test"
	content := "test content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.Score(query, content)
	}
}

func BenchmarkBM25_Score_LongQuery(b *testing.B) {
	bm25 := NewBM25()
	query := strings.Repeat("test query word ", 50)
	content := "test content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.Score(query, content)
	}
}

func BenchmarkBM25_Score_LongContent(b *testing.B) {
	bm25 := NewBM25()
	query := "test"
	content := strings.Repeat("test content ", 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.Score(query, content)
	}
}

func BenchmarkBM25_AddDocument_Small(b *testing.B) {
	bm25 := NewBM25()
	doc := strings.Repeat("test document ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.AddDocument(doc)
	}
}

func BenchmarkBM25_AddDocument_Large(b *testing.B) {
	bm25 := NewBM25()
	doc := strings.Repeat("test document ", 5000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.AddDocument(doc)
	}
}

func BenchmarkBM25_AddDocuments(b *testing.B) {
	docs := make([]string, 100)
	for i := 0; i < 100; i++ {
		docs[i] = strings.Repeat("document ", 50)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25 := NewBM25()
		bm25.AddDocuments(docs)
	}
}

func BenchmarkBM25_SetCorpus_Small(b *testing.B) {
	docs := make([]string, 100)
	for i := 0; i < 100; i++ {
		docs[i] = strings.Repeat("document ", 50)
	}

	bm25 := NewBM25()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.SetCorpus(docs)
	}
}

func BenchmarkBM25_SetCorpus_Large(b *testing.B) {
	docs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = strings.Repeat("document ", 50)
	}

	bm25 := NewBM25()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.SetCorpus(docs)
	}
}

func BenchmarkBM25_ScoreWithDetails(b *testing.B) {
	bm25 := NewBM25()
	query := "test"
	content := "test content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.ScoreWithDetails(query, content)
	}
}

func BenchmarkBM25_ScoreWithDetails_WithCorpus(b *testing.B) {
	bm25 := NewBM25()
	for i := 0; i < 100; i++ {
		bm25.AddDocument(strings.Repeat("document ", 50))
	}

	query := "test"
	content := "test content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.ScoreWithDetails(query, content)
	}
}

func BenchmarkBM25_ClearCorpus(b *testing.B) {
	bm25 := NewBM25()
	for i := 0; i < 1000; i++ {
		bm25.AddDocument(strings.Repeat("document ", 50))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm25.ClearCorpus()
	}
}

func BenchmarkTokenize_Small(b *testing.B) {
	text := strings.Repeat("test word ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenize(text)
	}
}

func BenchmarkTokenize_Medium(b *testing.B) {
	text := strings.Repeat("test word ", 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenize(text)
	}
}

func BenchmarkTokenize_Large(b *testing.B) {
	text := strings.Repeat("test word ", 5000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenize(text)
	}
}

func BenchmarkTokenizeWithDuplicates_Small(b *testing.B) {
	text := strings.Repeat("test word ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenizeWithDuplicates(text)
	}
}

func BenchmarkTokenizeWithDuplicates_Medium(b *testing.B) {
	text := strings.Repeat("test word ", 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenizeWithDuplicates(text)
	}
}

func BenchmarkTokenizeWithDuplicates_Large(b *testing.B) {
	text := strings.Repeat("test word ", 5000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenizeWithDuplicates(text)
	}
}
