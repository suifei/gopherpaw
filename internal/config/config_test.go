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
