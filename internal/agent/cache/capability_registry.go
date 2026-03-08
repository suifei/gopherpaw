// Package cache provides capability registry management.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RegistryManager 管理能力注册表的缓存和刷新。
type RegistryManager struct {
	mu            sync.RWMutex
	registry      *CapabilityRegistry
	cacheTTLHours int
	version       string // 当前版本号
}

// NewRegistryManager 创建一个新的能力注册表管理器。
func NewRegistryManager(cacheTTLHours int, version string) *RegistryManager {
	return &RegistryManager{
		cacheTTLHours: cacheTTLHours,
		version:       version,
	}
}

// Get 获取能力注册表，如果缓存未过期则返回缓存的版本。
func (m *RegistryManager) Get() (*CapabilityRegistry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 如果内存中有缓存且未过期，直接返回
	if m.registry != nil && !m.isExpired(m.registry) {
		return m.registry, nil
	}

	// 尝试从文件加载
	registry, err := m.loadFromFile()
	if err == nil && registry != nil && !m.isExpired(registry) {
		m.registry = registry
		return registry, nil
	}

	return nil, fmt.Errorf("cache expired or not found")
}

// Set 保存能力注册表到内存和文件。
func (m *RegistryManager) Set(registry *CapabilityRegistry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 设置版本号
	registry.Version = m.version
	registry.UpdatedAt = time.Now().Unix()

	m.registry = registry

	// 保存到文件
	if err := m.saveToFile(registry); err != nil {
		return fmt.Errorf("save to file: %w", err)
	}

	// 保存总结到单独的文件
	if registry.Summary != "" {
		if err := SaveSummary(registry.Summary); err != nil {
			return fmt.Errorf("save summary: %w", err)
		}
	}

	return nil
}

// Refresh 检查是否需要刷新缓存。
func (m *RegistryManager) Refresh() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.registry == nil {
		return true
	}

	return m.isExpired(m.registry)
}

// isExpired 检查注册表是否过期。
func (m *RegistryManager) isExpired(registry *CapabilityRegistry) bool {
	if m.cacheTTLHours <= 0 {
		return false // 永不过期
	}

	expiryTime := time.Unix(registry.UpdatedAt, 0).Add(time.Duration(m.cacheTTLHours) * time.Hour)
	return time.Now().After(expiryTime)
}

// loadFromFile 从文件加载能力注册表。
func (m *RegistryManager) loadFromFile() (*CapabilityRegistry, error) {
	data, err := LoadCapabilities()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在，不算错误
		}
		return nil, err
	}

	var registry CapabilityRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("unmarshal registry: %w", err)
	}

	// 如果版本不匹配，认为缓存无效
	if m.version != "" && registry.Version != m.version {
		return nil, fmt.Errorf("version mismatch: got %s, want %s", registry.Version, m.version)
	}

	return &registry, nil
}

// saveToFile 保存能力注册表到文件。
func (m *RegistryManager) saveToFile(registry *CapabilityRegistry) error {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	return SaveCapabilities(data)
}

// GetSummary 从文件加载缓存的总结（不检查过期时间，用于快速读取）。
func (m *RegistryManager) GetSummary() (string, error) {
	return LoadSummary()
}

// Clear 清除缓存。
func (m *RegistryManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.registry = nil
	return ClearCache()
}

// GetCacheInfo 获取缓存信息。
func (m *RegistryManager) GetCacheInfo() (*CacheInfo, error) {
	return GetCacheInfo()
}

// DefaultVersion 返回默认的缓存版本号。
// 当工具、技能、MCP 配置发生变化时，应该修改此版本号。
func DefaultVersion() string {
	// 使用编译时间作为版本的基础
	// 生产环境可以用配置文件版本或 Git commit hash
	return "v1.0.0"
}

// CapabilityVersionKey 返回用于检测能力变化的版本键。
// 这个键应该包含所有影响能力列表的因素：
// - 内置工具列表的变化
// - 技能目录内容的变化
// - MCP 服务器配置的变化
func CapabilityVersionKey(workingDir string, skillsActiveDir string, mcpConfig map[string]any) string {
	// 构建一个包含所有相关信息的版本键
	key := fmt.Sprintf("wd:%s|skills:%s|mcp:%v", workingDir, skillsActiveDir, mcpConfig)
	return key
}

// DirModTime 返回目录的最新修改时间。
func DirModTime(dir string) time.Time {
	var maxMod time.Time

	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(maxMod) {
			maxMod = info.ModTime()
		}
	}

	return maxMod
}

// FileExists 检查文件是否存在。
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureCacheDir 确保缓存目录存在。
func EnsureCacheDir() (string, error) {
	return CacheDir()
}

// GetCacheDirPath 返回缓存目录路径。
func GetCacheDirPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".gopherpaw", "cache")
	}
	return filepath.Join(homeDir, ".gopherpaw", "cache")
}
