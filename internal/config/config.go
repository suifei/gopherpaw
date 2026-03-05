// Package config provides configuration loading, validation, and environment overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	Server     ServerConfig    `mapstructure:"server" yaml:"server"`
	Agent      AgentConfig     `mapstructure:"agent" yaml:"agent"`
	LLM        LLMConfig       `mapstructure:"llm" yaml:"llm"`
	Memory     MemoryConfig    `mapstructure:"memory" yaml:"memory"`
	Channels   ChannelsConfig  `mapstructure:"channels" yaml:"channels"`
	Scheduler  SchedulerConfig `mapstructure:"scheduler" yaml:"scheduler"`
	MCP        MCPConfig       `mapstructure:"mcp" yaml:"mcp"`
	Log        LogConfig       `mapstructure:"log" yaml:"log"`
	WorkingDir string          `mapstructure:"working_dir" yaml:"working_dir"`
	Skills     SkillsConfig    `mapstructure:"skills" yaml:"skills"`
	Runtime    RuntimeConfig   `mapstructure:"runtime" yaml:"runtime"`
}

// RuntimeConfig holds runtime environment settings for Python and Node.js.
type RuntimeConfig struct {
	Python PythonConfig `mapstructure:"python" yaml:"python"`
	Node   NodeConfig   `mapstructure:"node" yaml:"node"`
}

// PythonConfig holds Python environment settings.
type PythonConfig struct {
	VenvPath    string `mapstructure:"venv_path" yaml:"venv_path"`       // Path to virtual environment
	Interpreter string `mapstructure:"interpreter" yaml:"interpreter"`   // Explicit python interpreter path
	AutoInstall bool   `mapstructure:"auto_install" yaml:"auto_install"` // Auto pip install missing packages
}

// NodeConfig holds Node.js/Bun environment settings.
type NodeConfig struct {
	Runtime     string `mapstructure:"runtime" yaml:"runtime"`           // "bun" or "node"
	BunPath     string `mapstructure:"bun_path" yaml:"bun_path"`         // Custom bun executable path
	NodePath    string `mapstructure:"node_path" yaml:"node_path"`       // Custom node executable path
	AutoInstall bool   `mapstructure:"auto_install" yaml:"auto_install"` // Auto install missing packages
}

// SkillsConfig holds skill directory settings.
type SkillsConfig struct {
	ActiveDir     string `mapstructure:"active_dir" yaml:"active_dir"`
	CustomizedDir string `mapstructure:"customized_dir" yaml:"customized_dir"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `mapstructure:"host" yaml:"host"`
	Port int    `mapstructure:"port" yaml:"port"`
}

// AgentConfig holds agent runtime settings.
type AgentConfig struct {
	SystemPrompt   string `mapstructure:"system_prompt" yaml:"system_prompt"`
	MaxTurns       int    `mapstructure:"max_turns" yaml:"max_turns"`
	MaxInputLength int    `mapstructure:"max_input_length" yaml:"max_input_length"`
	WorkingDir     string `mapstructure:"working_dir" yaml:"working_dir"`
}

// ModelSlot defines a named model with optional provider/credential overrides and capability tags.
type ModelSlot struct {
	Model        string   `mapstructure:"model" yaml:"model"`
	Provider     string   `mapstructure:"provider" yaml:"provider"`
	BaseURL      string   `mapstructure:"base_url" yaml:"base_url"`
	APIKey       string   `mapstructure:"api_key" yaml:"api_key"`
	Capabilities []string `mapstructure:"capabilities" yaml:"capabilities"`
}

// HasCapability returns true if this slot declares the given capability.
func (s ModelSlot) HasCapability(cap string) bool {
	for _, c := range s.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	Provider  string               `mapstructure:"provider" yaml:"provider"`
	Model     string               `mapstructure:"model" yaml:"model"`
	APIKey    string               `mapstructure:"api_key" yaml:"api_key"`
	BaseURL   string               `mapstructure:"base_url" yaml:"base_url"`
	OllamaURL string               `mapstructure:"ollama_url" yaml:"ollama_url"`
	Models    map[string]ModelSlot `mapstructure:"models" yaml:"models"`
}

// ResolveSlot returns a full LLMConfig for the given slot, inheriting unset fields from the parent.
func (c LLMConfig) ResolveSlot(name string) LLMConfig {
	slot, ok := c.Models[name]
	if !ok {
		return c
	}
	resolved := c
	if slot.Model != "" {
		resolved.Model = slot.Model
	}
	if slot.Provider != "" {
		resolved.Provider = slot.Provider
	}
	if slot.BaseURL != "" {
		resolved.BaseURL = slot.BaseURL
	}
	if slot.APIKey != "" {
		resolved.APIKey = slot.APIKey
	}
	return resolved
}

// MemoryConfig holds memory backend settings.
type MemoryConfig struct {
	Backend             string  `mapstructure:"backend" yaml:"backend"`
	DBPath              string  `mapstructure:"db_path" yaml:"db_path"`
	MaxHistory          int     `mapstructure:"max_history" yaml:"max_history"`
	WorkingDir          string  `mapstructure:"working_dir" yaml:"working_dir"`
	CompactThreshold    int     `mapstructure:"compact_threshold" yaml:"compact_threshold"`
	CompactKeepRecent   int     `mapstructure:"compact_keep_recent" yaml:"compact_keep_recent"`
	CompactRatio        float64 `mapstructure:"compact_ratio" yaml:"compact_ratio"`
	EmbeddingAPIKey     string  `mapstructure:"embedding_api_key" yaml:"embedding_api_key"`
	EmbeddingBaseURL    string  `mapstructure:"embedding_base_url" yaml:"embedding_base_url"`
	EmbeddingModel      string  `mapstructure:"embedding_model" yaml:"embedding_model"`
	EmbeddingDimensions int     `mapstructure:"embedding_dimensions" yaml:"embedding_dimensions"`
	EmbeddingMaxCache   int     `mapstructure:"embedding_max_cache" yaml:"embedding_max_cache"`
	FTSEnabled          bool    `mapstructure:"fts_enabled" yaml:"fts_enabled"`
}

// ChannelsConfig holds all channel configurations.
type ChannelsConfig struct {
	Console  ConsoleConfig  `mapstructure:"console" yaml:"console"`
	Telegram TelegramConfig `mapstructure:"telegram" yaml:"telegram"`
	Discord  DiscordConfig  `mapstructure:"discord" yaml:"discord"`
	DingTalk DingTalkConfig `mapstructure:"dingtalk" yaml:"dingtalk"`
	Feishu   FeishuConfig   `mapstructure:"feishu" yaml:"feishu"`
	QQ       QQConfig       `mapstructure:"qq" yaml:"qq"`
}

// ConsoleConfig holds console channel settings (stdin/stdout for dev).
type ConsoleConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

// TelegramConfig holds Telegram channel settings.
type TelegramConfig struct {
	Enabled       bool   `mapstructure:"enabled" yaml:"enabled"`
	BotPrefix     string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
	BotToken      string `mapstructure:"bot_token" yaml:"bot_token"`
	HTTPProxy     string `mapstructure:"http_proxy" yaml:"http_proxy"`
	HTTPProxyAuth string `mapstructure:"http_proxy_auth" yaml:"http_proxy_auth"`
}

// DiscordConfig holds Discord channel settings.
type DiscordConfig struct {
	Enabled       bool   `mapstructure:"enabled" yaml:"enabled"`
	BotPrefix     string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
	BotToken      string `mapstructure:"bot_token" yaml:"bot_token"`
	HTTPProxy     string `mapstructure:"http_proxy" yaml:"http_proxy"`
	HTTPProxyAuth string `mapstructure:"http_proxy_auth" yaml:"http_proxy_auth"`
}

// DingTalkConfig holds DingTalk channel settings.
type DingTalkConfig struct {
	Enabled      bool   `mapstructure:"enabled" yaml:"enabled"`
	BotPrefix    string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
	ClientID     string `mapstructure:"client_id" yaml:"client_id"`
	ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
}

// FeishuConfig holds Feishu channel settings.
type FeishuConfig struct {
	Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
	BotPrefix         string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
	AppID             string `mapstructure:"app_id" yaml:"app_id"`
	AppSecret         string `mapstructure:"app_secret" yaml:"app_secret"`
	EncryptKey        string `mapstructure:"encrypt_key" yaml:"encrypt_key"`
	VerificationToken string `mapstructure:"verification_token" yaml:"verification_token"`
}

// QQConfig holds QQ channel settings.
type QQConfig struct {
	Enabled      bool   `mapstructure:"enabled" yaml:"enabled"`
	BotPrefix    string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
	AppID        string `mapstructure:"app_id" yaml:"app_id"`
	ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
}

// SchedulerConfig holds scheduler settings.
type SchedulerConfig struct {
	Enabled   bool            `mapstructure:"enabled" yaml:"enabled"`
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat" yaml:"heartbeat"`
}

// HeartbeatConfig holds heartbeat job settings.
type HeartbeatConfig struct {
	Every       string       `mapstructure:"every" yaml:"every"`
	Target      string       `mapstructure:"target" yaml:"target"`
	ActiveHours *ActiveHours `mapstructure:"active_hours" yaml:"active_hours"`
}

// ActiveHours restricts heartbeat to a time window (HH:MM 24h).
type ActiveHours struct {
	Start string `mapstructure:"start" yaml:"start"`
	End   string `mapstructure:"end" yaml:"end"`
}

// MCPConfig holds MCP server configurations.
type MCPConfig struct {
	Servers map[string]MCPServerConfig `mapstructure:"servers" yaml:"servers"`
}

// MCPServerConfig holds a single MCP server's configuration.
// Supports three transport types: stdio, streamable_http, sse.
type MCPServerConfig struct {
	Name        string            `mapstructure:"name" yaml:"name"`
	Description string            `mapstructure:"description" yaml:"description"`
	Enabled     *bool             `mapstructure:"enabled" yaml:"enabled"` // nil = true (default), false = disabled
	Transport   string            `mapstructure:"transport" yaml:"transport"` // "stdio", "streamable_http", "sse" (default: "stdio")
	URL         string            `mapstructure:"url" yaml:"url"`             // HTTP/SSE endpoint URL
	Headers     map[string]string `mapstructure:"headers" yaml:"headers"`     // HTTP headers for remote transports
	Command     string            `mapstructure:"command" yaml:"command"`     // Executable command for stdio transport
	Args        []string          `mapstructure:"args" yaml:"args"`           // Command-line arguments
	Env         map[string]string `mapstructure:"env" yaml:"env"`             // Environment variables
	Cwd         string            `mapstructure:"cwd" yaml:"cwd"`             // Working directory for stdio transport
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level" yaml:"level"`
	Format string `mapstructure:"format" yaml:"format"`
}

const envPrefix = "GOPHERPAW_"

// Load reads configuration from the given path. Returns default config if file is missing.
// Environment variables with GOPHERPAW_ prefix override config values (e.g. GOPHERPAW_LLM_API_KEY).
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return defaultConfig(), nil
		}
		// Also treat "file not found" as missing config (e.g. on Windows)
		if errors.Is(err, os.ErrNotExist) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Apply env overrides for sensitive fields
	applyEnvOverrides(&cfg)

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// OnConfigChange is called when config file changes (for hot reload).
type OnConfigChange func(*Config)

// LoadWithWatch loads config and starts watching for file changes.
// When the config file changes, it reloads and calls onChange with the new config.
func LoadWithWatch(path string, onChange OnConfigChange) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return defaultConfig(), nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	applyEnvOverrides(&cfg)
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	if onChange != nil {
		v.OnConfigChange(func(_ fsnotify.Event) {
			if err := v.ReadInConfig(); err != nil {
				return
			}
			var newCfg Config
			if err := v.Unmarshal(&newCfg); err != nil {
				return
			}
			applyEnvOverrides(&newCfg)
			if Validate(&newCfg) == nil {
				onChange(&newCfg)
			}
		})
		v.WatchConfig()
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("agent.system_prompt", "You are a helpful AI assistant.")
	v.SetDefault("agent.max_turns", 20)
	v.SetDefault("agent.max_input_length", 131072)
	v.SetDefault("agent.working_dir", "")
	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.model", "gpt-4o-mini")
	v.SetDefault("llm.ollama_url", "http://localhost:11434")
	v.SetDefault("memory.backend", "sqlite")
	v.SetDefault("memory.db_path", "./data/gopherpaw.db")
	v.SetDefault("memory.max_history", 50)
	v.SetDefault("memory.working_dir", ".")
	v.SetDefault("memory.compact_threshold", 100000)
	v.SetDefault("memory.compact_keep_recent", 3)
	v.SetDefault("memory.compact_ratio", 0.7)
	v.SetDefault("memory.embedding_dimensions", 1024)
	v.SetDefault("memory.embedding_max_cache", 2000)
	v.SetDefault("memory.fts_enabled", true)
	v.SetDefault("channels.console.enabled", true)
	v.SetDefault("scheduler.enabled", false)
	v.SetDefault("scheduler.heartbeat.every", "30m")
	v.SetDefault("scheduler.heartbeat.target", "main")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	// Runtime defaults
	v.SetDefault("runtime.python.venv_path", "")
	v.SetDefault("runtime.python.interpreter", "")
	v.SetDefault("runtime.python.auto_install", true)
	v.SetDefault("runtime.node.runtime", "bun")
	v.SetDefault("runtime.node.bun_path", "")
	v.SetDefault("runtime.node.node_path", "")
	v.SetDefault("runtime.node.auto_install", true)
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Agent: AgentConfig{
			SystemPrompt:   "You are a helpful AI assistant.",
			MaxTurns:       20,
			MaxInputLength: 131072,
		},
		LLM: LLMConfig{
			Provider:  "openai",
			Model:     "gpt-4o-mini",
			OllamaURL: "http://localhost:11434",
		},
		Memory: MemoryConfig{
			Backend:             "sqlite",
			DBPath:              "./data/gopherpaw.db",
			MaxHistory:          50,
			WorkingDir:          ".",
			CompactThreshold:    100000,
			CompactKeepRecent:   3,
			CompactRatio:        0.7,
			EmbeddingDimensions: 1024,
			EmbeddingMaxCache:   2000,
			FTSEnabled:          true,
		},
		Scheduler: SchedulerConfig{
			Enabled: false,
			Heartbeat: HeartbeatConfig{
				Every:  "30m",
				Target: "main",
			},
		},
		MCP: MCPConfig{Servers: map[string]MCPServerConfig{}},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Runtime: RuntimeConfig{
			Python: PythonConfig{
				AutoInstall: true,
			},
			Node: NodeConfig{
				Runtime:     "bun",
				AutoInstall: true,
			},
		},
	}
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	return defaultConfig()
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv(envPrefix + "LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv(envPrefix + "LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv(envPrefix + "LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv(envPrefix + "LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv(envPrefix + "LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv(envPrefix + "MEMORY_WORKING_DIR"); v != "" {
		cfg.Memory.WorkingDir = v
	}
	if v := os.Getenv(envPrefix + "EMBEDDING_API_KEY"); v != "" {
		cfg.Memory.EmbeddingAPIKey = v
	}
	if v := os.Getenv(envPrefix + "EMBEDDING_BASE_URL"); v != "" {
		cfg.Memory.EmbeddingBaseURL = v
	}
	if v := os.Getenv(envPrefix + "EMBEDDING_MODEL"); v != "" {
		cfg.Memory.EmbeddingModel = v
	}
	if v := os.Getenv(envPrefix + "WORKING_DIR"); v != "" {
		cfg.WorkingDir = v
	}
}

// Validate checks configuration for required fields and valid ranges.
func Validate(cfg *Config) error {
	if cfg.Agent.MaxTurns < 1 || cfg.Agent.MaxTurns > 100 {
		return fmt.Errorf("agent.max_turns must be 1-100, got %d", cfg.Agent.MaxTurns)
	}
	if cfg.Agent.MaxInputLength > 0 && cfg.Agent.MaxInputLength < 1000 {
		return fmt.Errorf("agent.max_input_length must be >= 1000 when set, got %d", cfg.Agent.MaxInputLength)
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be 1-65535, got %d", cfg.Server.Port)
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(cfg.Log.Level)] {
		return fmt.Errorf("log.level must be debug/info/warn/error, got %q", cfg.Log.Level)
	}
	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[strings.ToLower(cfg.Log.Format)] {
		return fmt.Errorf("log.format must be json/console, got %q", cfg.Log.Format)
	}
	return nil
}

// ResolveWorkingDir returns the effective working directory (package-level).
// Priority: GOPHERPAW_WORKING_DIR env > config working_dir > ~/.gopherpaw/
func ResolveWorkingDir(cfgDir string) string {
	if v := os.Getenv(envPrefix + "WORKING_DIR"); v != "" {
		return expandPath(v)
	}
	if cfgDir != "" {
		return expandPath(cfgDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gopherpaw"
	}
	return filepath.Join(home, ".gopherpaw")
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// ProvidersConfig holds multiple LLM provider configs (for providers.json).
type ProvidersConfig struct {
	Providers map[string]LLMConfig `json:"providers" mapstructure:"providers"`
}

// LoadProviders loads providers from a JSON file at path. Returns nil if file not found.
func LoadProviders(path string) (*ProvidersConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read providers: %w", err)
	}
	var cfg ProvidersConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal providers: %w", err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]LLMConfig)
	}
	return &cfg, nil
}

// ResolveAgentWorkingDir returns the agent working directory. Empty means ~/.gopherpaw.
func (c *AgentConfig) ResolveWorkingDir() string {
	wd := c.WorkingDir
	if wd == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, ".gopherpaw")
		}
		return "."
	}
	if strings.HasPrefix(wd, "~/") {
		home, _ := os.UserHomeDir()
		wd = filepath.Join(home, wd[2:])
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return wd
	}
	return abs
}

// ResolveDBPath expands ~ and relative paths in DBPath.
func (c *MemoryConfig) ResolveDBPath() string {
	p := c.DBPath
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, p[2:])
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
