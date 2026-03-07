package mcp

import (
	"testing"
	"time"
)

func TestCircuitBreaker_Closed(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	})

	if cb.GetState() != "closed" {
		t.Errorf("Expected initial state 'closed', got %q", cb.GetState())
	}

	if !cb.CanExecute() {
		t.Error("Expected CanExecute() to return true in closed state")
	}
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.GetState() != "open" {
		t.Errorf("Expected state 'open' after %d failures, got %q", 3, cb.GetState())
	}

	if cb.CanExecute() {
		t.Error("Expected CanExecute() to return false in open state")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != "open" {
		t.Fatalf("Expected state 'open', got %q", cb.GetState())
	}

	time.Sleep(150 * time.Millisecond)

	if !cb.CanExecute() {
		t.Error("Expected CanExecute() to return true after timeout")
	}
}

func TestCircuitBreaker_CloseAfterSuccesses(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != "open" {
		t.Fatalf("Expected state 'open', got %q", cb.GetState())
	}

	time.Sleep(150 * time.Millisecond)

	_ = cb.CanExecute()

	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.GetState() != "closed" {
		t.Errorf("Expected state 'closed' after %d successes, got %q", 2, cb.GetState())
	}
}

func TestCircuitBreaker_Disabled(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		Enabled:          false,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	})

	for i := 0; i < 10; i++ {
		cb.RecordFailure()
		if !cb.CanExecute() {
			t.Error("Expected CanExecute() to always return true when disabled")
		}
	}
}

func TestDefaultReconnectConfig(t *testing.T) {
	cfg := DefaultReconnectConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if cfg.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", cfg.MaxRetries)
	}

	if cfg.HealthCheckInterval == 0 {
		t.Error("Expected HealthCheckInterval to be set")
	}

	if cfg.CircuitBreaker == nil {
		t.Error("Expected CircuitBreaker config to be set")
	}

	if !cfg.CircuitBreaker.Enabled {
		t.Error("Expected CircuitBreaker.Enabled to be true")
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if cfg.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold to be 5, got %d", cfg.FailureThreshold)
	}

	if cfg.SuccessThreshold != 2 {
		t.Errorf("Expected SuccessThreshold to be 2, got %d", cfg.SuccessThreshold)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout to be 30s, got %v", cfg.Timeout)
	}
}
