package agent

import (
	"strings"
	"testing"
)

func TestBuildEnvContext_AllFields(t *testing.T) {
	ctx := BuildEnvContext("session123", "user456", "telegram", "/home/work")

	if ctx == "" {
		t.Error("expected non-empty context")
	}
	if !strings.Contains(ctx, "session123") {
		t.Error("expected session_id in context")
	}
	if !strings.Contains(ctx, "user456") {
		t.Error("expected user_id in context")
	}
	if !strings.Contains(ctx, "telegram") {
		t.Error("expected channel in context")
	}
	if !strings.Contains(ctx, "/home/work") {
		t.Error("expected working_dir in context")
	}
	if !strings.Contains(ctx, "## 环境上下文") {
		t.Error("expected header in context")
	}
}

func TestBuildEnvContext_SomeFields(t *testing.T) {
	ctx := BuildEnvContext("session123", "", "console", "")

	if ctx == "" {
		t.Error("expected non-empty context")
	}
	if !strings.Contains(ctx, "session123") {
		t.Error("expected session_id in context")
	}
	if !strings.Contains(ctx, "console") {
		t.Error("expected channel in context")
	}
	if strings.Contains(ctx, "user_id") {
		t.Error("should not contain user_id when empty")
	}
}

func TestBuildEnvContext_NoFields(t *testing.T) {
	ctx := BuildEnvContext("", "", "", "")

	if ctx != "" {
		t.Errorf("expected empty context, got: %q", ctx)
	}
}

func TestBuildEnvContext_Format(t *testing.T) {
	ctx := BuildEnvContext("s1", "u1", "c1", "w1")

	// Check format
	lines := strings.Split(ctx, "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines, got %d", len(lines))
	}
	if lines[0] != "## 环境上下文" {
		t.Errorf("expected header, got: %q", lines[0])
	}
}
