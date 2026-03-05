package memory

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestInMemoryStore_SaveLoad(t *testing.T) {
	cfg := config.MemoryConfig{MaxHistory: 10}
	store := New(cfg).(*InMemoryStore)
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
	store := New(config.MemoryConfig{})
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
	store := New(cfg).(*InMemoryStore)
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
	store := New(config.MemoryConfig{MaxHistory: 50}).(*InMemoryStore)
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
	store := New(cfg).(*InMemoryStore)
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
	store := New(config.MemoryConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Load(ctx, "c1", 5)
	if err != context.Canceled {
		t.Errorf("Load with canceled ctx: got %v", err)
	}
}

func TestInMemoryStore_Concurrent(t *testing.T) {
	store := New(config.MemoryConfig{MaxHistory: 100})
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
	store := New(config.MemoryConfig{}).(*InMemoryStore)
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
	store := New(config.MemoryConfig{}).(*InMemoryStore)
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
