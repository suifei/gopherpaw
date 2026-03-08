// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const (
	fileSOUL      = "SOUL.md"
	filePROFILE   = "PROFILE.md"
	fileBOOTSTRAP = "BOOTSTRAP.md"
	fileAGENTS    = "AGENTS.md"
	fileMEMORY    = "MEMORY.md"
	fileHEARTBEAT = "HEARTBEAT.md"
	memoryDir     = "memory"
)

// PromptFileEntry defines a file to load and whether it is required.
type PromptFileEntry struct {
	Filename string
	Required bool
}

// PromptConfig holds the prompt loading configuration.
type PromptConfig struct {
	FileOrder []PromptFileEntry
	Language  string
}

// DefaultPromptConfig returns the standard CoPaw-compatible prompt config.
func DefaultPromptConfig() PromptConfig {
	return PromptConfig{
		FileOrder: []PromptFileEntry{
			{Filename: fileAGENTS, Required: true},
			{Filename: fileSOUL, Required: true},
			{Filename: filePROFILE, Required: false},
		},
		Language: "zh",
	}
}

// PromptLoader loads the six-file prompt system from working directory.
type PromptLoader struct {
	workingDir string
	fallback   string
	config     PromptConfig
}

// WorkingDir returns the resolved working directory path.
func (p *PromptLoader) WorkingDir() string {
	return p.workingDir
}

// NewPromptLoader creates a PromptLoader for the given working directory.
// fallback is used when no six-file files exist (e.g. config.yaml system_prompt).
func NewPromptLoader(workingDir string, fallback string) *PromptLoader {
	if workingDir == "" {
		workingDir = "."
	}
	return &PromptLoader{
		workingDir: workingDir,
		fallback:   fallback,
		config:     DefaultPromptConfig(),
	}
}

// NewPromptLoaderWithConfig creates a PromptLoader with custom config.
func NewPromptLoaderWithConfig(workingDir string, fallback string, cfg PromptConfig) *PromptLoader {
	loader := NewPromptLoader(workingDir, fallback)
	loader.config = cfg
	return loader
}

// LoadSystemPrompt returns the full system prompt. Falls back to cfg.SystemPrompt if six files are missing.
func (p *PromptLoader) LoadSystemPrompt() (string, error) {
	s := p.BuildSystemPrompt("")
	if s != "" {
		return s, nil
	}
	if p.fallback != "" {
		return p.fallback, nil
	}
	return "You are a helpful AI assistant.", nil
}

// LoadSOUL reads SOUL.md (Agent values and behavior).
func (p *PromptLoader) LoadSOUL() (string, error) {
	return p.readFile(fileSOUL)
}

// LoadAGENTS reads AGENTS.md (workflow and rules).
func (p *PromptLoader) LoadAGENTS() (string, error) {
	return p.readFile(fileAGENTS)
}

// LoadPROFILE reads PROFILE.md (identity and user profile).
func (p *PromptLoader) LoadPROFILE() (string, error) {
	return p.readFile(filePROFILE)
}

// LoadMEMORY reads MEMORY.md (today's memory or main MEMORY.md).
func (p *PromptLoader) LoadMEMORY() (string, error) {
	s, err := p.readTodayMemory()
	if err != nil || s != "" {
		return s, err
	}
	return p.readFile(fileMEMORY)
}

// LoadHEARTBEAT reads HEARTBEAT.md (heartbeat checklist).
func (p *PromptLoader) LoadHEARTBEAT() (string, error) {
	return p.readFile(fileHEARTBEAT)
}

// HasBootstrap returns true if BOOTSTRAP.md exists.
func (p *PromptLoader) HasBootstrap() bool {
	path := filepath.Join(p.workingDir, fileBOOTSTRAP)
	_, err := os.Stat(path)
	return err == nil
}

// DeleteBootstrap removes BOOTSTRAP.md after bootstrap completes.
func (p *PromptLoader) DeleteBootstrap() error {
	path := filepath.Join(p.workingDir, fileBOOTSTRAP)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete BOOTSTRAP.md: %w", err)
	}
	logger.L().Info("Bootstrap completed, BOOTSTRAP.md removed", zap.String("path", path))
	return nil
}

// BuildSystemPrompt concatenates: AGENTS + response format convention + capability index +
// skills content + SOUL + PROFILE + MEMORY.
// This aligns with CoPaw's loading order while adding AI collaboration features.
// Returns empty string if AGENTS and SOUL are missing (caller should use fallback).
func (p *PromptLoader) BuildSystemPrompt(skillsContent string) string {
	var parts []string

	// Load AGENTS first (workflow rules and how-to instructions)
	agents, err := p.LoadAGENTS()
	if err != nil || agents == "" {
		return ""
	}
	parts = append(parts, "# AGENTS.md\n\n"+agents)

	// IMPORTANT: Response format convention (AI collaboration)
	// This tells the AI how to structure responses for better collaboration
	parts = append(parts, "\n\n"+p.buildResponseFormatConvention())

	// Capability index - tells AI what capabilities are available
	if skillsContent != "" {
		parts = append(parts, "\n\n"+p.buildCapabilityIndex(skillsContent))
	}

	// Storage service guide - tells AI about the storage service
	parts = append(parts, "\n\n"+p.buildStorageServiceGuide())

	// Then SOUL (core values and identity)
	soul, err := p.LoadSOUL()
	if err != nil || soul == "" {
		return ""
	}
	parts = append(parts, "\n\n# SOUL.md\n\n"+soul)

	// Then PROFILE (optional identity and user profile)
	profile, _ := p.LoadPROFILE()
	if profile != "" {
		parts = append(parts, "# PROFILE.md\n\n"+profile)
	}

	memMain, _ := p.readFile(fileMEMORY)
	if memMain != "" {
		parts = append(parts, "# MEMORY.md\n\n"+memMain)
	}
	memToday, _ := p.readTodayMemory()
	if memToday != "" {
		parts = append(parts, "# memory/"+time.Now().Format("2006-01-02")+".md\n\n"+memToday)
	}

	return strings.Join(parts, "\n\n")
}

// buildResponseFormatConvention 构建响应格式约定说明。
func (p *PromptLoader) buildResponseFormatConvention() string {
	lang := p.config.Language
	if lang == "en" {
		return "# 🤝 AI Collaboration Framework\n\n" +
			"This framework works with you as an intelligent assistant. You can optionally structure your responses to help the framework assist you better.\n\n" +
			"## Optional Structured Response Format\n\n" +
			"You may include a JSON block in your responses (using markdown code blocks) to:\n\n" +
			"1. **Tell us what you're thinking** - We'll help you stay on track\n" +
			"2. **Request capabilities** - We'll remind you how to use them\n" +
			"3. **Store intermediate results** - We'll keep them for later retrieval\n" +
			"4. **Track progress** - We'll help you remember goals and milestones\n\n" +
			"### Example:\n\n" +
			"```json\n" +
			"{\n" +
			"  \"thought\": \"User wants a Word document. I need to use the docx skill.\",\n" +
			"  \"current_focus\": \"Preparing to generate Word document\",\n" +
			"  \"next_step\": \"Read the docx skill file first\",\n" +
			"  \"capabilities_needed\": [\"docx\"],\n" +
			"  \"progress_update\": \"Starting task\",\n" +
			"  \"storage_requests\": [\n" +
			"    {\n" +
			"      \"name\": \"macbook_prices\",\n" +
			"      \"description\": \"MacBook Air M5 price comparison data\",\n" +
			"      \"content\": \"US: $1099, China: ¥8999, ...\"\n" +
			"    }\n" +
			"  ],\n" +
			"  \"retrieval_requests\": [\"macbook_prices\"],\n" +
			"  \"final_answer\": \"\"\n" +
			"}\n" +
			"```\n\n" +
			"### Fields:\n" +
			"- `thought`: Your current thinking process\n" +
			"- `current_focus`: What you're working on now\n" +
			"- `next_step`: Your next planned action\n" +
			"- `capabilities_needed`: Capabilities you need (skills/tools/MCPs)\n" +
			"- `progress_update`: Progress status\n" +
			"- `storage_requests`: Content to store for later use\n" +
			"- `retrieval_requests`: Names of stored content to retrieve\n" +
			"- `final_answer`: Your final answer to the user (set when done)\n\n" +
			"*When you use `storage_requests`, the framework stores the content. When you use `retrieval_requests`, the framework injects the stored content into your context.\n\n" +
			"**Note**: This is OPTIONAL. You can respond normally. The structured format helps when working on complex tasks."
	}

	// Chinese (default)
	return "# 🤝 AI 智能协作框架\n\n" +
		"本框架作为智能助手与你协作。你可以选择性地使用结构化响应格式，让框架更好地协助你。\n\n" +
		"## 可选的结构化响应格式\n\n" +
		"你可以在响应中包含 JSON 代码块来：\n\n" +
		"1. **告诉我们你的思考** - 我们会帮你保持方向\n" +
		"2. **请求需要的能力** - 我们会提醒你如何使用\n" +
		"3. **存储中间结果** - 我们会保存供后续检索\n" +
		"4. **追踪进度** - 我们会帮你记住目标和里程碑\n\n" +
		"### 示例：\n\n" +
		"```json\n" +
		"{\n" +
		"  \"thought\": \"用户需要生成 Word 文档。我需要使用 docx 技能。\",\n" +
		"  \"current_focus\": \"准备生成 Word 文档\",\n" +
		"  \"next_step\": \"先读取 docx 技能文件\",\n" +
		"  \"capabilities_needed\": [\"docx\"],\n" +
		"  \"progress_update\": \"开始任务\",\n" +
		"  \"storage_requests\": [\n" +
		"    {\n" +
		"      \"name\": \"macbook_prices\",\n" +
		"      \"description\": \"MacBook Air M5 价格对比数据\",\n" +
		"      \"content\": \"美国: $1099, 中国: ¥8999, ...\"\n" +
		"    }\n" +
		"  ],\n" +
		"  \"retrieval_requests\": [\"macbook_prices\"],\n" +
		"  \"final_answer\": \"\"\n" +
		"}\n" +
		"```\n\n" +
		"### 字段说明：\n" +
		"- `thought`: 你当前的思考过程\n" +
		"- `current_focus`: 当前关注的部分\n" +
		"- `next_step`: 下一步计划\n" +
		"- `capabilities_needed`: 需要的能力（技能/工具/MCP）\n" +
		"- `progress_update`: 进度更新\n" +
		"- `storage_requests`: 需要存储的内容\n" +
		"- `retrieval_requests`: 需要检索的已存储内容名称\n" +
		"- `final_answer`: 给用户的最终回答（完成任务时设置）\n\n" +
		"*使用 `storage_requests` 时，框架会存储内容。使用 `retrieval_requests` 时，框架会注入已存储的内容到你的上下文中。\n\n" +
		"**注意**：这是可选的。你可以正常响应。结构化格式在处理复杂任务时很有帮助。"
}

// buildCapabilityIndex 构建能力索引说明。
// 引导 AI 主动探索可用技能，而不是依赖预设映射。
func (p *PromptLoader) buildCapabilityIndex(skillsContent string) string {
	lang := p.config.Language
	if lang == "en" {
		return "# 📋 Available Capabilities\n\n" +
			"## How to Find Skills\n\n" +
			"Skills are located in the `configs/active_skills/` directory. Each skill has a SKILL.md file with:\n" +
			"- A description of what the skill does\n" +
			"- Instructions on how to use it\n\n" +
			"## When You Need a Capability\n\n" +
			"1. **Think about what you need** - e.g., \"create a Word document\"\n" +
			"2. **Explore available skills** - Use `read_file` to list the `configs/active_skills/` directory\n" +
			"3. **Read relevant SKILL.md files** - Learn the proper method\n" +
			"4. **Execute using the skill's instructions** - Not generic tools\n\n" +
			"## Common File Type → Skill Pattern\n\n" +
			"When you see a file extension, check if there's a matching skill:\n\n" +
			"| File Extension | Likely Skill |\n" +
			"|----------------|-------------|\n" +
			"| .docx, .doc | `configs/active_skills/docx/SKILL.md` |\n" +
			"| .xlsx, .xls | `configs/active_skills/xlsx/SKILL.md` |\n" +
			"| .pptx, .ppt | `configs/active_skills/pptx/SKILL.md` |\n" +
			"| .pdf | `configs/active_skills/pdf/SKILL.md` |\n\n" +
			"## Important Principle\n\n" +
			"**Never use generic tools (like `write_file`) for specialized formats.**\n" +
			"Always check if there's a skill first. The framework will guide you if you forget.\n\n" +
			"---\n" + skillsContent
	}

	// Chinese (default)
	return "# 📋 可用能力\n\n" +
		"## 如何查找技能\n\n" +
		"技能位于 `configs/active_skills/` 目录。每个技能都有一个 SKILL.md 文件，包含：\n" +
		"- 技能描述（它能做什么）\n" +
		"- 使用说明（如何正确使用）\n\n" +
		"## 当你需要某个能力时\n\n" +
		"1. **思考你需要什么** - 例如：「创建 Word 文档」\n" +
		"2. **探索可用技能** - 用 `read_file` 列出 `configs/active_skills/` 目录\n" +
		"3. **阅读相关 SKILL.md** - 学习正确的使用方法\n" +
		"4. **按技能指导执行** - 不要用通用工具\n\n" +
		"## 常见文件类型 → 技能模式\n\n" +
		"看到文件扩展名时，检查是否有对应技能：\n\n" +
		"| 文件扩展名 | 可能的技能 |\n" +
		"|-----------|----------|\n" +
		"| .docx, .doc | `configs/active_skills/docx/SKILL.md` |\n" +
		"| .xlsx, .xls | `configs/active_skills/xlsx/SKILL.md` |\n" +
		"| .pptx, .ppt | `configs/active_skills/pptx/SKILL.md` |\n" +
		"| .pdf | `configs/active_skills/pdf/SKILL.md` |\n\n" +
		"## 重要原则\n\n" +
		"**永远不要用通用工具（如 `write_file`）处理专业格式。**\n" +
		"始终先检查是否有对应技能。框架会在你遗忘时提醒你。\n\n" +
		"---\n" + skillsContent
}

// buildStorageServiceGuide 构建存储服务说明。
func (p *PromptLoader) buildStorageServiceGuide() string {
	lang := p.config.Language
	if lang == "en" {
		return "# 💾 Framework Storage Service\n\n" +
			"You can store valuable information during your work and retrieve it later.\n\n" +
			"## Storing Content\n\n" +
			"Include in your structured response:\n" +
			"```json\n" +
			"{\n" +
			"  \"storage_requests\": [\n" +
			"    {\n" +
			"      \"name\": \"unique_name\",\n" +
			"      \"description\": \"Brief description\",\n" +
			"      \"content\": \"The content to store\"\n" +
			"    }\n" +
			"  ]\n" +
			"}\n" +
			"```\n\n" +
			"## Retrieving Content\n\n" +
			"Include in your structured response:\n" +
			"```json\n" +
			"{\n" +
			"  \"retrieval_requests\": [\"name1\", \"name2\"]\n" +
			"}\n" +
			"```\n\n" +
			"The framework will automatically inject the retrieved content into your context.\n\n" +
			"## Use Cases\n\n" +
			"- Store intermediate calculation results\n" +
			"- Save data collected from multiple sources\n" +
			"- Keep reference information for later use\n" +
			"- Maintain state across multiple turns\n\n" +
			"The storage is scoped per chat and persists during the conversation."
	}

	// Chinese (default)
	return "# 💾 框架存储服务\n\n" +
		"你可以在工作过程中存储有价值的信息，供后续使用。\n\n" +
		"## 存储内容\n\n" +
		"在结构化响应中包含：\n" +
		"```json\n" +
		"{\n" +
		"  \"storage_requests\": [\n" +
		"    {\n" +
		"      \"name\": \"唯一名称\",\n" +
		"      \"description\": \"简短描述\",\n" +
		"      \"content\": \"要存储的内容\"\n" +
		"    }\n" +
		"  ]\n" +
			"}\n" +
			"```\n\n" +
			"## 检索内容\n\n" +
			"在结构化响应中包含：\n" +
			"```json\n" +
			"{\n" +
			"  \"retrieval_requests\": [\"名称1\", \"名称2\"]\n" +
			"}\n" +
			"```\n\n" +
			"框架会自动将检索到的内容注入到你的上下文中。\n\n" +
			"## 使用场景\n\n" +
			"- 存储中间计算结果\n" +
			"- 保存从多个来源收集的数据\n" +
			"- 保存参考信息供后续使用\n" +
			"- 跨多个轮次维护状态\n\n" +
			"存储作用域为每个对话，在会话期间持续有效。"
}

// stripYAMLFrontmatter removes YAML frontmatter (---wrapped) from content.
// This matches CoPaw's behavior where YAML frontmatter like ---summary...---
// is stripped before including the content in the system prompt.
func stripYAMLFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	return content
}

func (p *PromptLoader) readFile(name string) (string, error) {
	path := filepath.Join(p.workingDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	content := strings.TrimSpace(string(data))
	content = stripYAMLFrontmatter(content)
	return content, nil
}

func (p *PromptLoader) readTodayMemory() (string, error) {
	today := time.Now().Format("2006-01-02")
	path := filepath.Join(p.workingDir, memoryDir, today+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read memory/%s.md: %w", today, err)
	}
	content := strings.TrimSpace(string(data))
	content = stripYAMLFrontmatter(content)
	return content, nil
}

// CopyMDFiles copies default md_files from srcDir to the working directory.
// Only copies files that don't already exist.
func (p *PromptLoader) CopyMDFiles(srcDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read md_files dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		dstPath := filepath.Join(p.workingDir, e.Name())
		if _, err := os.Stat(dstPath); err == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			continue
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			logger.L().Warn("copy md file", zap.String("file", e.Name()), zap.Error(err))
		}
	}
	return nil
}

// Language returns the configured language.
func (p *PromptLoader) Language() string {
	return p.config.Language
}

// Config returns the prompt configuration.
func (p *PromptLoader) Config() PromptConfig {
	return p.config
}
