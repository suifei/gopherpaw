package llm

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

type mockProvider struct {
	name  string
	calls atomic.Int64
}

func (m *mockProvider) Chat(_ context.Context, _ *agent.ChatRequest) (*agent.ChatResponse, error) {
	m.calls.Add(1)
	return &agent.ChatResponse{Content: "ok from " + m.name}, nil
}

func (m *mockProvider) ChatStream(_ context.Context, _ *agent.ChatRequest) (agent.ChatStream, error) {
	m.calls.Add(1)
	return &mockStream{name: m.name}, nil
}

func (m *mockProvider) Name() string { return m.name }

type mockStream struct{ name string }

func (s *mockStream) Recv() (*agent.ChatChunk, error) { return nil, io.EOF }
func (s *mockStream) Close() error                    { return nil }

func newTestRouter(t *testing.T) *ModelRouter {
	t.Helper()
	r := &ModelRouter{
		providers: map[string]agent.LLMProvider{
			"default": &mockProvider{name: "text-model"},
			"vision":  &mockProvider{name: "vision-model"},
			"code":    &mockProvider{name: "code-model"},
		},
		slots: map[string]config.ModelSlot{
			"default": {Model: "gpt-4o-mini", Capabilities: []string{"text", "tools"}},
			"vision":  {Model: "autoglm-phone", Capabilities: []string{"text", "vision", "tools"}},
			"code":    {Model: "deepseek-coder", Capabilities: []string{"text", "code", "tools"}},
		},
		active:  "default",
		baseCfg: config.LLMConfig{Model: "gpt-4o-mini"},
	}
	return r
}

func TestModelRouter_SingleSlot(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "openai",
		Model:    "test-model",
		APIKey:   "fake",
	}
	mock := &mockProvider{name: "single"}
	r := &ModelRouter{
		providers: map[string]agent.LLMProvider{defaultSlotName: mock},
		slots:     map[string]config.ModelSlot{defaultSlotName: {Model: "test-model"}},
		active:    defaultSlotName,
		baseCfg:   cfg,
	}

	if r.Name() != "single" {
		t.Errorf("single-slot Name() = %q, want %q", r.Name(), "single")
	}

	resp, err := r.Chat(context.Background(), &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok from single" {
		t.Errorf("unexpected response: %s", resp.Content)
	}
}

func TestModelRouter_MultiSlot_Name(t *testing.T) {
	r := newTestRouter(t)
	if r.Name() != "model-router" {
		t.Errorf("multi-slot Name() = %q, want %q", r.Name(), "model-router")
	}
}

func TestModelRouter_DefaultRouting(t *testing.T) {
	r := newTestRouter(t)
	resp, err := r.Chat(context.Background(), &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "what is 2+2?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok from text-model" {
		t.Errorf("expected default routing, got: %s", resp.Content)
	}
}

func TestModelRouter_VisionAutoRoute(t *testing.T) {
	r := newTestRouter(t)

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"png file", "Here is the screenshot: /tmp/screen.png", "ok from vision-model"},
		{"jpg file", "Check this image photo.jpg please", "ok from vision-model"},
		{"jpeg file", "image.jpeg content", "ok from vision-model"},
		{"webp file", "file.webp uploaded", "ok from vision-model"},
		{"base64 image", "data:image/png;base64,iVBOR...", "ok from vision-model"},
		{"no image", "just a text question", "ok from text-model"},
		{"code file", "check main.go for errors", "ok from text-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := r.Chat(context.Background(), &agent.ChatRequest{
				Messages: []agent.Message{{Role: "user", Content: tt.content}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if resp.Content != tt.want {
				t.Errorf("got %q, want %q", resp.Content, tt.want)
			}
		})
	}
}

func TestModelRouter_VisionNoFallback(t *testing.T) {
	r := &ModelRouter{
		providers: map[string]agent.LLMProvider{
			"default": &mockProvider{name: "text-only"},
		},
		slots: map[string]config.ModelSlot{
			"default": {Model: "text-model", Capabilities: []string{"text"}},
		},
		active:  "default",
		baseCfg: config.LLMConfig{Model: "text-model"},
	}

	resp, err := r.Chat(context.Background(), &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "look at screen.png"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok from text-only" {
		t.Errorf("should fallback to default when no vision slot: got %q", resp.Content)
	}
}

func TestModelRouter_Switch(t *testing.T) {
	r := newTestRouter(t)

	if r.ActiveSlot() != "default" {
		t.Errorf("initial active = %q, want %q", r.ActiveSlot(), "default")
	}

	if err := r.Switch("code"); err != nil {
		t.Fatal(err)
	}
	if r.ActiveSlot() != "code" {
		t.Errorf("after switch active = %q, want %q", r.ActiveSlot(), "code")
	}

	resp, err := r.Chat(context.Background(), &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "plain text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok from code-model" {
		t.Errorf("expected code-model after switch, got: %s", resp.Content)
	}

	if err := r.Switch("nonexistent"); err == nil {
		t.Error("Switch to nonexistent slot should fail")
	}
}

func TestModelRouter_SlotNames(t *testing.T) {
	r := newTestRouter(t)
	names := r.SlotNames()
	if len(names) != 3 {
		t.Errorf("SlotNames() len = %d, want 3", len(names))
	}
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, want := range []string{"default", "vision", "code"} {
		if !nameSet[want] {
			t.Errorf("SlotNames() missing %q", want)
		}
	}
}

func TestModelRouter_HasCapability(t *testing.T) {
	r := newTestRouter(t)
	if !r.HasCapability("vision") {
		t.Error("HasCapability(vision) should be true")
	}
	if !r.HasCapability("code") {
		t.Error("HasCapability(code) should be true")
	}
	if r.HasCapability("audio") {
		t.Error("HasCapability(audio) should be false")
	}
}

func TestNeedsVision(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"hello world", false},
		{"/tmp/photo.png", true},
		{"image.JPG here", true},
		{"data:image/png;base64,abc", true},
		{"main.go is broken", false},
		{"file.txt and doc.pdf", false},
	}
	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			req := &agent.ChatRequest{
				Messages: []agent.Message{{Role: "user", Content: tt.content}},
			}
			if got := needsVision(req); got != tt.want {
				t.Errorf("needsVision(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestContainsImageRef(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"screenshot.png", true},
		{"photo.jpg", true},
		{"image.jpeg", true},
		{"pic.webp", true},
		{"icon.bmp", true},
		{"logo.svg", true},
		{"data:image/png;base64,iVBOR", true},
		{"readme.md", false},
		{"main.go", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := containsImageRef(tt.input); got != tt.want {
				t.Errorf("containsImageRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestModelSlot_HasCapability(t *testing.T) {
	slot := config.ModelSlot{Capabilities: []string{"text", "vision"}}
	if !slot.HasCapability("vision") {
		t.Error("should have vision")
	}
	if slot.HasCapability("code") {
		t.Error("should not have code")
	}
	empty := config.ModelSlot{}
	if empty.HasCapability("text") {
		t.Error("empty slot should have no capabilities")
	}
}

func TestLLMConfig_ResolveSlot(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "openai",
		Model:    "base-model",
		APIKey:   "base-key",
		BaseURL:  "https://api.example.com/v1",
		Models: map[string]config.ModelSlot{
			"vision": {
				Model: "vision-model",
			},
			"custom": {
				Model:   "custom-model",
				BaseURL: "https://custom.example.com/v1",
				APIKey:  "custom-key",
			},
		},
	}

	resolved := cfg.ResolveSlot("vision")
	if resolved.Model != "vision-model" {
		t.Errorf("vision model = %q, want %q", resolved.Model, "vision-model")
	}
	if resolved.APIKey != "base-key" {
		t.Errorf("vision should inherit base APIKey, got %q", resolved.APIKey)
	}
	if resolved.BaseURL != "https://api.example.com/v1" {
		t.Errorf("vision should inherit base BaseURL, got %q", resolved.BaseURL)
	}

	resolved2 := cfg.ResolveSlot("custom")
	if resolved2.BaseURL != "https://custom.example.com/v1" {
		t.Errorf("custom BaseURL = %q, want override", resolved2.BaseURL)
	}
	if resolved2.APIKey != "custom-key" {
		t.Errorf("custom APIKey = %q, want override", resolved2.APIKey)
	}

	resolved3 := cfg.ResolveSlot("nonexistent")
	if resolved3.Model != "base-model" {
		t.Errorf("nonexistent slot should return base config, got model=%q", resolved3.Model)
	}
}

func TestModelRouter_ChatStream(t *testing.T) {
	r := newTestRouter(t)
	stream, err := r.ChatStream(context.Background(), &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	_, err = stream.Recv()
	if err != io.EOF {
		t.Errorf("expected EOF from mock stream, got %v", err)
	}
}

func TestModelRouter_ConcurrentAccess(t *testing.T) {
	r := newTestRouter(t)
	ctx := context.Background()

	done := make(chan error, 20)
	for i := 0; i < 10; i++ {
		go func(i int) {
			_, err := r.Chat(ctx, &agent.ChatRequest{
				Messages: []agent.Message{{Role: "user", Content: fmt.Sprintf("msg %d", i)}},
			})
			done <- err
		}(i)
		go func(i int) {
			slots := []string{"default", "vision", "code"}
			done <- r.Switch(slots[i%3])
		}(i)
	}
	for i := 0; i < 20; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent op failed: %v", err)
		}
	}
}
