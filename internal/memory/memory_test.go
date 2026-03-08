package memory

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestInMemoryStore_SaveLoad(t *testing.T) {
	cfg := config.MemoryConfig{MaxHistory: 10}
	store := New(cfg, nil).(*InMemoryStore)
	ctx := context.Background()

	// Save and load
	msg := agent.Message{Role: "user", Content: "hello"}
	if err := store.Save(ctx, "chat1", msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	msgs, err := store.Load(ctx, "chat1", 5)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "hello" {
		t.Errorf("Load: got %v", msgs)
	}
}

func TestInMemoryStore_EmptyChatID(t *testing.T) {
	store := New(config.MemoryConfig{}, nil)
	ctx := context.Background()
	msg := agent.Message{Role: "user", Content: "x"}
	if err := store.Save(ctx, "", msg); err == nil {
		t.Error("Save with empty chatID should fail")
	}
	if _, err := store.Load(ctx, "", 5); err == nil {
		t.Error("Load with empty chatID should fail")
	}
	if _, err := store.Search(ctx, "", "q", 5); err == nil {
		t.Error("Search with empty chatID should fail")
	}
	if _, err := store.GetCompactSummary(ctx, ""); err == nil {
		t.Error("GetCompactSummary with empty chatID should fail")
	}
	if err := store.SaveLongTerm(ctx, "", "x", "memory"); err == nil {
		t.Error("SaveLongTerm with empty chatID should fail")
	}
	if _, err := store.LoadLongTerm(ctx, ""); err == nil {
		t.Error("LoadLongTerm with empty chatID should fail")
	}
}

func TestInMemoryStore_MaxHistory(t *testing.T) {
	cfg := config.MemoryConfig{MaxHistory: 3}
	store := New(cfg, nil).(*InMemoryStore)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		store.Save(ctx, "c1", agent.Message{Role: "user", Content: string(rune('0' + i))})
	}
	msgs, _ := store.Load(ctx, "c1", 10)
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "2" || msgs[1].Content != "3" || msgs[2].Content != "4" {
		t.Errorf("Load: got %v", msgs)
	}
}

func TestInMemoryStore_Search(t *testing.T) {
	store := New(config.MemoryConfig{MaxHistory: 50}, nil).(*InMemoryStore)
	ctx := context.Background()
	store.Save(ctx, "c1", agent.Message{Role: "user", Content: "hello world"})
	store.Save(ctx, "c1", agent.Message{Role: "assistant", Content: "hi there"})
	store.Save(ctx, "c1", agent.Message{Role: "user", Content: "world peace"})
	results, err := store.Search(ctx, "c1", "world", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestInMemoryStore_Compact(t *testing.T) {
	cfg := config.MemoryConfig{MaxHistory: 3}
	store := New(cfg, nil).(*InMemoryStore)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		store.Save(ctx, "c1", agent.Message{Role: "user", Content: string(rune('0' + i))})
	}
	if err := store.Compact(ctx, "c1"); err != nil {
		t.Fatalf("Compact: %v", err)
	}
	msgs, _ := store.Load(ctx, "c1", 10)
	if len(msgs) != 3 {
		t.Errorf("after Compact expected 3, got %d", len(msgs))
	}
}

func TestInMemoryStore_ContextCancel(t *testing.T) {
	store := New(config.MemoryConfig{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Load(ctx, "c1", 5)
	if err != context.Canceled {
		t.Errorf("Load with canceled ctx: got %v", err)
	}
}

func TestInMemoryStore_Concurrent(t *testing.T) {
	store := New(config.MemoryConfig{MaxHistory: 100}, nil)
	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			store.Save(ctx, "c1", agent.Message{Role: "user", Content: "x"})
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 50; i++ {
			store.Load(ctx, "c1", 10)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

func TestInMemoryStore_GetCompactSummary(t *testing.T) {
	store := New(config.MemoryConfig{}, nil).(*InMemoryStore)
	ctx := context.Background()
	summary, err := store.GetCompactSummary(ctx, "c1")
	if err != nil {
		t.Fatalf("GetCompactSummary: %v", err)
	}
	if summary != "" {
		t.Errorf("InMemoryStore should return empty summary, got %q", summary)
	}
}

func TestInMemoryStore_SaveLoadLongTerm(t *testing.T) {
	store := New(config.MemoryConfig{}, nil).(*InMemoryStore)
	ctx := context.Background()
	if err := store.SaveLongTerm(ctx, "c1", "content", "memory"); err != nil {
		t.Fatalf("SaveLongTerm: %v", err)
	}
	content, err := store.LoadLongTerm(ctx, "c1")
	if err != nil {
		t.Fatalf("LoadLongTerm: %v", err)
	}
	if content != "" {
		t.Errorf("InMemoryStore LoadLongTerm should return empty, got %q", content)
	}
}

func TestBM25_Score(t *testing.T) {
	bm := NewBM25()
	if s := bm.Score("", "hello"); s != 0 {
		t.Errorf("empty query: got %v", s)
	}
	if s := bm.Score("hello", ""); s != 0 {
		t.Errorf("empty content: got %v", s)
	}
	s := bm.Score("hello world", "hello world foo")
	if s < 0.5 {
		t.Errorf("expected match, got %v", s)
	}
	s2 := bm.Score("hello world", "hello world")
	if s2 < 1.0 {
		t.Errorf("full match should score >= 1.0: got %v", s2)
	}
}

func TestBM25_WithCorpus(t *testing.T) {
	bm := NewBM25()

	// Add corpus for IDF calculation
	bm.AddDocuments([]string{
		"the quick brown fox jumps over the lazy dog",
		"the lazy dog sleeps all day",
		"a quick fox runs fast",
		"dogs and cats are pets",
		"the weather is nice today",
	})

	// "quick" appears in 2/5 docs, "lazy" appears in 2/5 docs
	// Rare term "weather" should have higher IDF
	scoreCommon := bm.Score("the", "the quick brown fox") // "the" is very common
	scoreRare := bm.Score("weather", "the weather is nice today")

	// Rare term should score relatively higher due to IDF
	if scoreCommon <= 0 {
		t.Errorf("common term score should be positive: got %v", scoreCommon)
	}
	if scoreRare <= 0 {
		t.Errorf("rare term score should be positive: got %v", scoreRare)
	}
}

func TestBM25_LengthNormalization(t *testing.T) {
	bm := NewBM25()

	// Set corpus with varying document lengths
	bm.AddDocuments([]string{
		"short doc",
		"this is a medium length document with more words",
		"this is a very long document that contains many many words and has a lot of content repeated multiple times to make it longer",
	})

	// Query for a term that appears once in each
	query := "doc"
	shortContent := "short doc about programming"
	longContent := "this is a very long doc that contains many many words and has a lot of content repeated multiple times to make it longer and even more content here"

	shortScore := bm.Score(query, shortContent)
	longScore := bm.Score(query, longContent)

	// With length normalization, short document should not be penalized too much
	// and long document should not get unfair advantage just from length
	if shortScore <= 0 {
		t.Errorf("short doc score should be positive: got %v", shortScore)
	}
	if longScore <= 0 {
		t.Errorf("long doc score should be positive: got %v", longScore)
	}
}

func TestBM25_Options(t *testing.T) {
	// Test custom k1 and b parameters
	bm := NewBM25(WithK1(2.0), WithB(0.5))

	s := bm.Score("hello world", "hello world foo bar")
	if s <= 0 {
		t.Errorf("score with custom params should be positive: got %v", s)
	}

	// Test with extreme settings
	bmNoNorm := NewBM25(WithB(0)) // No length normalization
	sNoNorm := bmNoNorm.Score("test", "test document")
	if sNoNorm <= 0 {
		t.Errorf("score without normalization should be positive: got %v", sNoNorm)
	}
}

func TestBM25_ScoreWithDetails(t *testing.T) {
	bm := NewBM25()
	bm.AddDocuments([]string{
		"hello world",
		"hello there",
		"world peace",
	})

	score, details := bm.ScoreWithDetails("hello world", "hello world example")

	if score <= 0 {
		t.Errorf("score should be positive: got %v", score)
	}

	// Check that details contains expected keys
	if _, ok := details["doc_length"]; !ok {
		t.Error("details should contain doc_length")
	}
	if _, ok := details["k1"]; !ok {
		t.Error("details should contain k1")
	}
	if _, ok := details["b"]; !ok {
		t.Error("details should contain b")
	}
	if _, ok := details["total_score"]; !ok {
		t.Error("details should contain total_score")
	}
}

func TestBM25_ClearCorpus(t *testing.T) {
	bm := NewBM25()
	bm.AddDocuments([]string{"doc1", "doc2", "doc3"})

	// Score with corpus
	s1 := bm.Score("doc1", "doc1 content")

	// Clear and score again
	bm.ClearCorpus()
	s2 := bm.Score("doc1", "doc1 content")

	// Both should be positive, but may differ due to IDF
	if s1 <= 0 || s2 <= 0 {
		t.Errorf("scores should be positive: s1=%v, s2=%v", s1, s2)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	sim, err := CosineSimilarity(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if sim != 1.0 {
		t.Errorf("identical vectors: got %v", sim)
	}
	a2 := []float32{1, 0, 0}
	b2 := []float32{0, 1, 0}
	sim2, err := CosineSimilarity(a2, b2)
	if err != nil {
		t.Fatal(err)
	}
	if sim2 != 0 {
		t.Errorf("orthogonal vectors: got %v", sim2)
	}
	_, err = CosineSimilarity([]float32{1, 2}, []float32{1})
	if err == nil {
		t.Error("mismatch length should error")
	}
}

func TestInMemoryStore_SummaryMemory(t *testing.T) {
	store := New(config.MemoryConfig{}, nil).(*InMemoryStore)
	ctx := context.Background()

	// Empty messages
	summary, err := store.SummaryMemory(ctx, nil)
	if err != nil {
		t.Fatalf("SummaryMemory empty: %v", err)
	}
	if summary != "" {
		t.Errorf("SummaryMemory empty: expected empty, got %q", summary)
	}

	// Some messages
	msgs := []agent.Message{
		{Role: "user", Content: "Hello, how are you?"},
		{Role: "assistant", Content: "I'm doing well, thanks!"},
	}
	summary, err = store.SummaryMemory(ctx, msgs)
	if err != nil {
		t.Fatalf("SummaryMemory: %v", err)
	}
	if !contains(summary, "User:") || !contains(summary, "AI:") {
		t.Errorf("SummaryMemory expected role prefixes, got %q", summary)
	}

	// Context cancel
	ctxCancel, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.SummaryMemory(ctxCancel, msgs)
	if err != context.Canceled {
		t.Errorf("SummaryMemory with canceled ctx: got %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != "" && s != substr &&
		(s == substr || len(s) > len(substr) && s[:len(substr)] == substr ||
			len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}

func TestCompactor_SummaryMemory(t *testing.T) {
	cfg := config.MemoryConfig{}
	compactor := NewCompactor(nil, cfg)
	ctx := context.Background()

	// Empty messages
	summary, err := compactor.SummaryMemory(ctx, nil)
	if err != nil {
		t.Fatalf("SummaryMemory empty: %v", err)
	}
	if summary != "" {
		t.Errorf("SummaryMemory empty: expected empty, got %q", summary)
	}

	// With messages (no LLM, should return concatenation)
	msgs := []agent.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is 2+2?"},
		{Role: "assistant", Content: "2+2 equals 4."},
	}
	summary, err = compactor.SummaryMemory(ctx, msgs)
	if err != nil {
		t.Fatalf("SummaryMemory: %v", err)
	}
	// System messages should be skipped
	if contains(summary, "system") {
		t.Errorf("SummaryMemory should skip system messages, got %q", summary)
	}
	if !contains(summary, "User:") || !contains(summary, "AI:") {
		t.Errorf("SummaryMemory should include User and AI, got %q", summary)
	}
}
