// Package cache provides file-based caching for capability registries.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheDir 返回能力缓存目录的路径。
func CacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".gopherpaw", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}
	return cacheDir, nil
}

// CapabilityCachePath 返回能力缓存文件的路径。
func CapabilityCachePath() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "capabilities.json"), nil
}

// CapabilitySummaryPath 返回能力总结文件的路径。
func CapabilitySummaryPath() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "capability_summary.md"), nil
}

// SaveCapabilities 保存能力注册表到缓存文件。
func SaveCapabilities(data []byte) error {
	path, err := CapabilityCachePath()
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	// 原子写入：先写临时文件，再重命名
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	// 在 Windows 上需要先删除目标文件
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename cache file: %w", err)
	}

	return nil
}

// LoadCapabilities 从缓存文件加载能力注册表。
func LoadCapabilities() ([]byte, error) {
	path, err := CapabilityCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// SaveSummary 保存 AI 生成的能力总结到文件。
func SaveSummary(summary string) error {
	path, err := CapabilitySummaryPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(summary), 0644); err != nil {
		return fmt.Errorf("write summary file: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename summary file: %w", err)
	}

	return nil
}

// LoadSummary 从文件加载 AI 生成的能力总结。
func LoadSummary() (string, error) {
	path, err := CapabilitySummaryPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// CacheInfo 包含缓存元信息。
type CacheInfo struct {
	Exists    bool      `json:"exists"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      int64     `json:"size"`
	Version   string    `json:"version,omitempty"`
}

// GetCacheInfo 获取缓存文件的信息。
func GetCacheInfo() (*CacheInfo, error) {
	path, err := CapabilityCachePath()
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CacheInfo{Exists: false}, nil
		}
		return nil, err
	}

	// 读取缓存获取版本信息
	var version string
	data, err := os.ReadFile(path)
	if err == nil {
		var tmp map[string]any
		if json.Unmarshal(data, &tmp) == nil {
			if v, ok := tmp["version"].(string); ok {
				version = v
			}
		}
	}

	return &CacheInfo{
		Exists:    true,
		UpdatedAt: info.ModTime(),
		Size:      info.Size(),
		Version:   version,
	}, nil
}

// IsCacheExpired 检查缓存是否过期（超过指定小时数）。
func IsCacheExpired(ttlHours int) (bool, error) {
	info, err := GetCacheInfo()
	if err != nil {
		return true, err // 出错时认为已过期
	}
	if !info.Exists {
		return true, nil
	}

	expiryTime := info.UpdatedAt.Add(time.Duration(ttlHours) * time.Hour)
	return time.Now().After(expiryTime), nil
}

// ClearCache 清除所有缓存文件。
func ClearCache() error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}

	// 删除目录下的所有文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove cache file %s: %w", entry.Name(), err)
		}
	}

	return nil
}
