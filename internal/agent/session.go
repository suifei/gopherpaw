// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import "sync"

// Session holds per-chat conversation state.
type Session struct {
	ChatID string
	mu     sync.RWMutex
}

// SessionManager manages sessions by chatID.
type SessionManager struct {
	mu      sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate returns the session for chatID, creating it if needed.
func (m *SessionManager) GetOrCreate(chatID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[chatID]; ok {
		return s
	}
	s := &Session{ChatID: chatID}
	m.sessions[chatID] = s
	return s
}

// Remove removes a session (e.g. on /new command).
func (m *SessionManager) Remove(chatID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, chatID)
}
