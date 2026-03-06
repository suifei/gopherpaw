package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	// Use a path that does not exist (works on Windows and Unix)
	missingPath := filepath.Join(t.TempDir(), "nonexistent.yaml")
	cfg, err := Load(missingPath)
	if err != nil {
		t.Fatalf("Load should not fail for missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load should return default config")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: 127.0.0.1
  port: 9000
agent:
  max_turns: 10
log:
  level: debug
  format: console
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Agent.MaxTurns != 10 {
		t.Errorf("expected max_turns 10, got %d", cfg.Agent.MaxTurns)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level debug, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "console" {
		t.Errorf("expected log format console, got %q", cfg.Log.Format)
	}
}

func TestLoad_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("invalid: yaml: [:"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load should fail for invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{"valid default", nil, false},
		{"invalid max_turns 0", func(c *Config) { c.Agent.MaxTurns = 0 }, true},
		{"invalid max_turns 101", func(c *Config) { c.Agent.MaxTurns = 101 }, true},
		{"invalid port 0", func(c *Config) { c.Server.Port = 0 }, true},
		{"invalid port 70000", func(c *Config) { c.Server.Port = 70000 }, true},
		{"invalid log level", func(c *Config) { c.Log.Level = "trace" }, true},
		{"invalid log format", func(c *Config) { c.Log.Format = "xml" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			if tt.modify != nil {
				tt.modify(cfg)
			}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("log:\n  level: info\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	os.Setenv("GOPHERPAW_LOG_LEVEL", "debug")
	os.Setenv("GOPHERPAW_LLM_API_KEY", "sk-test-key")
	defer os.Unsetenv("GOPHERPAW_LOG_LEVEL")
	defer os.Unsetenv("GOPHERPAW_LLM_API_KEY")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected env override log level debug, got %q", cfg.Log.Level)
	}
	if cfg.LLM.APIKey != "sk-test-key" {
		t.Errorf("expected env override api key, got %q", cfg.LLM.APIKey)
	}
}

func TestResolveDBPath(t *testing.T) {
	mc := MemoryConfig{DBPath: "./data/test.db"}
	resolved := mc.ResolveDBPath()
	if resolved == "" {
		t.Error("ResolveDBPath should not return empty")
	}
	if !filepath.IsAbs(resolved) {
		t.Errorf("expected absolute path, got %q", resolved)
	}
}

func TestResolveWorkingDir(t *testing.T) {
	// Empty config dir -> should resolve to ~/.gopherpaw (or .gopherpaw if no home)
	got := ResolveWorkingDir("")
	if got == "" {
		t.Error("ResolveWorkingDir should not return empty")
	}
	// Explicit dir
	got = ResolveWorkingDir("/tmp/test")
	if got == "" {
		t.Error("ResolveWorkingDir with path should not return empty")
	}
}

func TestModelSlot_HasCapability(t *testing.T) {
	tests := []struct {
		name string
		slot ModelSlot
		cap  string
		want bool
	}{
		{
			name: "capability exists",
			slot: ModelSlot{Capabilities: []string{"vision", "tool-use"}},
			cap:  "vision",
			want: true,
		},
		{
			name: "capability does not exist",
			slot: ModelSlot{Capabilities: []string{"vision", "tool-use"}},
			cap:  "streaming",
			want: false,
		},
		{
			name: "empty capabilities",
			slot: ModelSlot{Capabilities: []string{}},
			cap:  "vision",
			want: false,
		},
		{
			name: "nil capabilities",
			slot: ModelSlot{Capabilities: nil},
			cap:  "vision",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.slot.HasCapability(tt.cap)
			if got != tt.want {
				t.Errorf("HasCapability(%q) = %v, want %v", tt.cap, got, tt.want)
			}
		})
	}
}

func TestLLMConfig_ResolveSlot(t *testing.T) {
	tests := []struct {
		name   string
		config LLMConfig
		slot   string
		want   LLMConfig
	}{
		{
			name: "slot exists and overrides parent",
			config: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "key1",
				Models: map[string]ModelSlot{
					"vision": {Model: "gpt-4-vision", APIKey: "key2"},
				},
			},
			slot: "vision",
			want: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4-vision",
				APIKey:   "key2",
				Models: map[string]ModelSlot{
					"vision": {Model: "gpt-4-vision", APIKey: "key2"},
				},
			},
		},
		{
			name: "slot does not exist, returns parent",
			config: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "key1",
			},
			slot: "nonexistent",
			want: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "key1",
			},
		},
		{
			name: "slot exists with partial override",
			config: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "key1",
				Models: map[string]ModelSlot{
					"code": {Model: "gpt-4-code"},
				},
			},
			slot: "code",
			want: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4-code",
				APIKey:   "key1",
				Models: map[string]ModelSlot{
					"code": {Model: "gpt-4-code"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ResolveSlot(tt.slot)
			if got.Provider != tt.want.Provider || got.Model != tt.want.Model {
				t.Errorf("ResolveSlot(%q) = {Provider: %q, Model: %q}, want {Provider: %q, Model: %q}",
					tt.slot, got.Provider, got.Model, tt.want.Provider, tt.want.Model)
			}
			if got.APIKey != tt.want.APIKey {
				t.Errorf("ResolveSlot(%q) APIKey = %q, want %q", tt.slot, got.APIKey, tt.want.APIKey)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig should return non-nil config")
	}
	// Verify some basic defaults
	if cfg.Server.Port == 0 {
		t.Error("DefaultConfig should have non-zero port")
	}
	if cfg.Agent.MaxTurns == 0 {
		t.Error("DefaultConfig should have non-zero max_turns")
	}
}

func TestLLMConfig_ResolveSlot_EdgeCases(t *testing.T) {
	cfg := LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Models:   nil,
	}
	result := cfg.ResolveSlot("test")
	if result.Model != "gpt-4" {
		t.Errorf("ResolveSlot with nil Models should return parent config, got model %q", result.Model)
	}
}

func TestLoadWithWatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: 127.0.0.1
  port: 9000
log:
  level: debug
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	onChange := func(cfg *Config) {}

	cfg, err := LoadWithWatch(path, onChange)
	if err != nil {
		t.Fatalf("LoadWithWatch failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadWithWatch should return config")
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level debug, got %q", cfg.Log.Level)
	}
}

func TestLoadWithWatch_MissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "nonexistent.yaml")
	cfg, err := LoadWithWatch(missingPath, nil)
	if err != nil {
		t.Fatalf("LoadWithWatch should not fail for missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadWithWatch should return default config")
	}
}

func TestLoadProviders_MissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "providers.json")
	cfg, err := LoadProviders(missingPath)
	if err != nil {
		t.Fatalf("LoadProviders should not fail for missing file: %v", err)
	}
	if cfg != nil {
		t.Fatal("LoadProviders should return nil for missing file")
	}
}

func TestLoadProviders_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")
	content := `{"providers": {"openai": {"provider": "openai", "model": "gpt-4"}}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write providers: %v", err)
	}

	cfg, err := LoadProviders(path)
	if err != nil {
		t.Fatalf("LoadProviders failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadProviders should return config")
	}
	if cfg.Providers == nil {
		t.Fatal("LoadProviders should have providers map")
	}
	if len(cfg.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(cfg.Providers))
	}
}

func TestLoadProviders_EmptyProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("write providers: %v", err)
	}

	cfg, err := LoadProviders(path)
	if err != nil {
		t.Fatalf("LoadProviders failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadProviders should return config")
	}
	if cfg.Providers == nil {
		t.Fatal("LoadProviders should initialize providers map")
	}
}

func TestAgentConfig_ResolveWorkingDir(t *testing.T) {
	tests := []struct {
		name         string
		workingDir   string
		emptyHome    bool
		wantContains string
	}{
		{"empty working dir", "", false, ".gopherpaw"},
		{"relative path", "./data", false, "data"},
		{"absolute path", "/tmp/test", false, "/tmp/test"},
		{"tilde path", "~/test", false, "/test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AgentConfig{WorkingDir: tt.workingDir}
			result := cfg.ResolveWorkingDir()
			if result == "" {
				t.Error("ResolveWorkingDir should not return empty")
			}
		})
	}
}

func TestMemoryConfig_ResolveDBPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"relative path", "./data/test.db"},
		{"absolute path", "/tmp/test.db"},
		{"tilde path", "~/data/test.db"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := MemoryConfig{DBPath: tt.path}
			result := cfg.ResolveDBPath()
			if result == "" {
				t.Error("ResolveDBPath should not return empty")
			}
		})
	}
}

func TestValidate_AllEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("log:\n  level: info\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	os.Setenv("GOPHERPAW_LLM_API_KEY", "sk-test-key")
	os.Setenv("GOPHERPAW_LLM_BASE_URL", "https://test.com")
	os.Setenv("GOPHERPAW_LLM_MODEL", "gpt-4")
	os.Setenv("GOPHERPAW_LOG_LEVEL", "debug")
	os.Setenv("GOPHERPAW_LOG_FORMAT", "console")
	os.Setenv("GOPHERPAW_MEMORY_WORKING_DIR", "/tmp/mem")
	os.Setenv("GOPHERPAW_EMBEDDING_API_KEY", "sk-embed-key")
	os.Setenv("GOPHERPAW_EMBEDDING_BASE_URL", "https://embed.com")
	os.Setenv("GOPHERPAW_EMBEDDING_MODEL", "text-embed")
	os.Setenv("GOPHERPAW_WORKING_DIR", "/tmp/work")
	defer os.Unsetenv("GOPHERPAW_LLM_API_KEY")
	defer os.Unsetenv("GOPHERPAW_LLM_BASE_URL")
	defer os.Unsetenv("GOPHERPAW_LLM_MODEL")
	defer os.Unsetenv("GOPHERPAW_LOG_LEVEL")
	defer os.Unsetenv("GOPHERPAW_LOG_FORMAT")
	defer os.Unsetenv("GOPHERPAW_MEMORY_WORKING_DIR")
	defer os.Unsetenv("GOPHERPAW_EMBEDDING_API_KEY")
	defer os.Unsetenv("GOPHERPAW_EMBEDDING_BASE_URL")
	defer os.Unsetenv("GOPHERPAW_EMBEDDING_MODEL")
	defer os.Unsetenv("GOPHERPAW_WORKING_DIR")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.LLM.APIKey != "sk-test-key" {
		t.Errorf("expected API key from env, got %q", cfg.LLM.APIKey)
	}
	if cfg.LLM.BaseURL != "https://test.com" {
		t.Errorf("expected BaseURL from env, got %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "gpt-4" {
		t.Errorf("expected Model from env, got %q", cfg.LLM.Model)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level from env, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "console" {
		t.Errorf("expected log format from env, got %q", cfg.Log.Format)
	}
	if cfg.Memory.WorkingDir != "/tmp/mem" {
		t.Errorf("expected Memory.WorkingDir from env, got %q", cfg.Memory.WorkingDir)
	}
	if cfg.Memory.EmbeddingAPIKey != "sk-embed-key" {
		t.Errorf("expected EmbeddingAPIKey from env, got %q", cfg.Memory.EmbeddingAPIKey)
	}
	if cfg.Memory.EmbeddingBaseURL != "https://embed.com" {
		t.Errorf("expected EmbeddingBaseURL from env, got %q", cfg.Memory.EmbeddingBaseURL)
	}
	if cfg.Memory.EmbeddingModel != "text-embed" {
		t.Errorf("expected EmbeddingModel from env, got %q", cfg.Memory.EmbeddingModel)
	}
	if cfg.WorkingDir != "/tmp/work" {
		t.Errorf("expected WorkingDir from env, got %q", cfg.WorkingDir)
	}
}
