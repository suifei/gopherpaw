package skills

import (
	"testing"
	"time"
)

func TestComputeBackoff(t *testing.T) {
	d := computeBackoff(1, 0.8, 6.0)
	if d < 700*time.Millisecond || d > 900*time.Millisecond {
		t.Errorf("attempt 1: expected ~800ms, got %v", d)
	}
	d = computeBackoff(2, 0.8, 6.0)
	if d < 1500*time.Millisecond || d > 1700*time.Millisecond {
		t.Errorf("attempt 2: expected ~1600ms, got %v", d)
	}
	d = computeBackoff(10, 0.8, 6.0)
	if d > 6100*time.Millisecond {
		t.Errorf("attempt 10: expected capped at 6s, got %v", d)
	}
}

func TestJoinURL(t *testing.T) {
	tests := []struct {
		base, path, want string
	}{
		{"https://example.com", "/api/v1", "https://example.com/api/v1"},
		{"https://example.com/", "/api/v1", "https://example.com/api/v1"},
		{"https://example.com", "api/v1", "https://example.com/api/v1"},
	}
	for _, tt := range tests {
		got := joinURL(tt.base, tt.path)
		if got != tt.want {
			t.Errorf("joinURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
		}
	}
}

func TestExtractClawHubSlug(t *testing.T) {
	tests := []struct {
		url, want string
	}{
		{"https://clawhub.ai/skills/my-skill", "my-skill"},
		{"https://clawhub.ai/skills/my-skill/", "my-skill"},
		{"https://clawhub.ai/other/path", "path"},
	}
	for _, tt := range tests {
		got := extractClawHubSlug(tt.url)
		if got != tt.want {
			t.Errorf("extractClawHubSlug(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestDeriveNameFromGitHubURL(t *testing.T) {
	tests := []struct {
		url, want string
	}{
		{"https://github.com/user/repo/blob/main/skills/my-skill/SKILL.md", "my-skill"},
		{"https://raw.githubusercontent.com/user/repo/main/skills/test/SKILL.md", "test"},
		{"https://github.com/user/repo", "repo"},
	}
	for _, tt := range tests {
		got := deriveNameFromGitHubURL(tt.url)
		if got != tt.want {
			t.Errorf("deriveNameFromGitHubURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestNormSearchItems_Array(t *testing.T) {
	body := `[{"slug":"s1","name":"Skill One","description":"desc","version":"1.0"}]`
	results, err := normSearchItems(body)
	if err != nil {
		t.Fatalf("normSearchItems: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Slug != "s1" {
		t.Errorf("slug: got %q", results[0].Slug)
	}
	if results[0].Name != "Skill One" {
		t.Errorf("name: got %q", results[0].Name)
	}
}

func TestNormSearchItems_Wrapper(t *testing.T) {
	body := `{"results":[{"slug":"s2","name":"Two"}]}`
	results, err := normSearchItems(body)
	if err != nil {
		t.Fatalf("normSearchItems: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Slug != "s2" {
		t.Errorf("slug: got %q", results[0].Slug)
	}
}

func TestNormSearchItems_Empty(t *testing.T) {
	body := `[]`
	results, err := normSearchItems(body)
	if err != nil {
		t.Fatalf("normSearchItems: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDecodeBase64Content(t *testing.T) {
	encoded := "SGVsbG8gV29ybGQ="
	decoded, err := DecodeBase64Content(encoded)
	if err != nil {
		t.Fatalf("DecodeBase64Content: %v", err)
	}
	if decoded != "Hello World" {
		t.Errorf("got %q, want %q", decoded, "Hello World")
	}
}

func TestEnvOrDefault(t *testing.T) {
	if envOrDefault("NONEXISTENT_VAR_12345", "default") != "default" {
		t.Error("expected default value")
	}
}

func TestJsonStr(t *testing.T) {
	m := map[string]any{"key": "value", "num": 42}
	if jsonStr(m, "key") != "value" {
		t.Error("expected 'value'")
	}
	if jsonStr(m, "num") != "" {
		t.Error("expected empty for non-string")
	}
	if jsonStr(m, "missing") != "" {
		t.Error("expected empty for missing key")
	}
}

func TestEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
		defVal int
		want   int
	}{
		{
			name:   "env not set",
			envKey: "NONEXISTENT_INT_VAR_12345",
			envVal: "",
			defVal: 42,
			want:   42,
		},
		{
			name:   "env set to valid int",
			envKey: "TEST_INT_VAR",
			envVal: "100",
			defVal: 42,
			want:   100,
		},
		{
			name:   "env set to invalid int",
			envKey: "TEST_INVALID_INT_VAR",
			envVal: "invalid",
			defVal: 42,
			want:   42,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := envIntOrDefault(tt.envKey, tt.defVal)
			if got != tt.want {
				t.Errorf("envIntOrDefault(%q, %d) = %d, want %d", tt.envKey, tt.defVal, got, tt.want)
			}
		})
	}
}

func TestEnvDurationOrDefault(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
		defVal time.Duration
		want   time.Duration
	}{
		{
			name:   "env not set",
			envKey: "NONEXISTENT_DURATION_VAR_12345",
			envVal: "",
			defVal: 10 * time.Second,
			want:   10 * time.Second,
		},
		{
			name:   "env set to valid duration",
			envKey: "TEST_DURATION_VAR",
			envVal: "5s",
			defVal: 10 * time.Second,
			want:   5 * time.Second,
		},
		{
			name:   "env set to invalid duration",
			envKey: "TEST_INVALID_DURATION_VAR",
			envVal: "invalid",
			defVal: 10 * time.Second,
			want:   10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := envDurationOrDefault(tt.envKey, tt.defVal)
			if got != tt.want {
				t.Errorf("envDurationOrDefault(%q, %v) = %v, want %v", tt.envKey, tt.defVal, got, tt.want)
			}
		})
	}
}
