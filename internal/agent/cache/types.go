// Package cache provides capability registry management without circular dependencies.
package cache

import (
	"time"
)

// Capability 能力定义，描述系统能提供的各种能力（工具、技能、MCP）。
type Capability struct {
	ID          string            `json:"id"`          // 唯一标识 (如 "tool:read_file", "skill:docx")
	Type        string            `json:"type"`        // "tool" | "skill" | "mcp"
	Name        string            `json:"name"`        // 名称
	Description string            `json:"description"` // 描述
	Parameters  any               `json:"parameters,omitempty"` // 参数 Schema
	Examples    []string          `json:"examples,omitempty"`   // 使用示例
	Metadata    map[string]string `json:"metadata,omitempty"`   // 额外信息 (如 SKILL.md 路径)
}

// CapabilityRegistry 能力注册表，保存所有系统能力和 AI 生成的总结。
type CapabilityRegistry struct {
	Capabilities []Capability `json:"capabilities"` // 所有能力列表
	Summary      string       `json:"summary"`      // AI 生成的能力总结
	UpdatedAt    int64        `json:"updated_at"`   // 更新时间戳
	Version      string       `json:"version"`      // 版本号，用于检测变化
}

// CapabilityInfo 缓存信息。
type CapabilityInfo struct {
	Exists    bool      `json:"exists"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      int64     `json:"size"`
	Version   string    `json:"version,omitempty"`
}
