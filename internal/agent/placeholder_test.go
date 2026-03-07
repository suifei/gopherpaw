package agent

import (
	"context"
	"testing"
)

func TestPlaceholderAgent_Run(t *testing.T) {
	a := NewPlaceholder()
	ctx := context.Background()
	resp, err := a.Run(ctx, "chat1", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp == "" {
		t.Error("Run returned empty response")
	}
	if resp != "[Placeholder] Agent not yet implemented. Message received: hello" {
		t.Errorf("unexpected response: %q", resp)
	}
}

func TestPlaceholderAgent_Run_Canceled(t *testing.T) {
	a := NewPlaceholder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := a.Run(ctx, "chat1", "hello")
	if err != context.Canceled {
		t.Errorf("Run with canceled ctx: got err %v, want context.Canceled", err)
	}
}

func TestPlaceholderAgent_RunStream(t *testing.T) {
	a := NewPlaceholder()
	ctx := context.Background()
	ch, err := a.RunStream(ctx, "chat1", "hi")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	msg := <-ch
	if msg == "" {
		t.Error("RunStream returned empty message")
	}
	if msg != "[Placeholder] Agent not yet implemented. Message received: hi" {
		t.Errorf("unexpected stream message: %q", msg)
	}
}
