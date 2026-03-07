package agent

import (
	"context"
	"testing"
)

type testModelSwitcher struct {
	activeSlot string
}

func (m *testModelSwitcher) Switch(slotName string) error {
	m.activeSlot = slotName
	return nil
}

func (m *testModelSwitcher) ActiveSlot() string {
	return m.activeSlot
}

func (m *testModelSwitcher) SlotNames() []string {
	return []string{"default", "chat", "embedding"}
}

func (m *testModelSwitcher) HasCapability(cap string) bool {
	return cap == "chat"
}

func TestWithModelSwitcher(t *testing.T) {
	ctx := context.Background()
	ms := &testModelSwitcher{activeSlot: "default"}

	ctx = WithModelSwitcher(ctx, ms)

	got := GetModelSwitcher(ctx)
	if got == nil {
		t.Error("ModelSwitcher not set correctly")
	}
}

func TestGetModelSwitcher_Nil(t *testing.T) {
	ctx := context.Background()

	got := GetModelSwitcher(ctx)
	if got != nil {
		t.Error("expected nil ModelSwitcher")
	}
}

func TestWithChatID(t *testing.T) {
	ctx := context.Background()
	chatID := "test_chat_123"

	ctx = WithChatID(ctx, chatID)

	got := GetChatID(ctx)
	if got != chatID {
		t.Errorf("ChatID = %q, want %q", got, chatID)
	}
}
