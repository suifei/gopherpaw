package agent

import (
	"testing"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}
}

func TestSessionManager_GetOrCreate(t *testing.T) {
	sm := NewSessionManager()

	sess1 := sm.GetOrCreate("chat1")
	if sess1 == nil {
		t.Fatal("GetOrCreate returned nil")
	}
	if sess1.ChatID != "chat1" {
		t.Errorf("ChatID = %q, want %q", sess1.ChatID, "chat1")
	}

	sess2 := sm.GetOrCreate("chat1")
	if sess2 != sess1 {
		t.Error("expected same session instance")
	}

	sess3 := sm.GetOrCreate("chat2")
	if sess3 == sess1 {
		t.Error("expected different session for different chatID")
	}
}

func TestSessionManager_Remove(t *testing.T) {
	sm := NewSessionManager()

	_ = sm.GetOrCreate("chat1")
	sm.Remove("chat1")

	sess := sm.GetOrCreate("chat1")
	if sess == nil {
		t.Error("expected new session after remove")
	}
}

func TestSessionManager_RemoveNonExistent(t *testing.T) {
	sm := NewSessionManager()
	sm.Remove("nonexistent")
}
