// Package skills provides skill loading and management from SKILL.md files.
package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
	"gopkg.in/yaml.v3"
)

// Skill represents a loaded SKILL.md with YAML front matter.
type Skill struct {
	Name        string
	Description string
	Keywords    []string   // Keywords for skill matching
	Content     string
	Enabled     bool
	Path        string
	Scripts     map[string]string // Script files from scripts/ directory
	References  map[string]string // Reference docs from references/ directory
	ExtraFiles  map[string][]byte // Extra files (binary or text)
}

// SkillAdapter 提供方法让 Skill 满足 agent 包中的接口要求。
// 这些方法使用 "Get" 前缀避免与字段名冲突。
func (s Skill) GetName() string        { return s.Name }
func (s Skill) GetDescription() string { return s.Description }
func (s Skill) GetKeywords() []string  { return s.Keywords }
func (s Skill) GetPath() string        { return s.Path }
func (s Skill) GetEnabled() bool       { return s.Enabled }

// Manager loads and manages skills from directories.
type Manager struct {
	mu         sync.RWMutex
	skills     map[string]*Skill
	currentQuery string // Current user query for dynamic skill selection
}

// NewManager creates a new skill manager.
func NewManager() *Manager {
	return &Manager{
		skills: make(map[string]*Skill),
	}
}

// LoadSkills loads SKILL.md files from active_dir and customized_dir under
// both workingDir (user data, e.g. ~/.gopherpaw/) and configDir (config file
// location, for built-in skills). Duplicate paths are skipped automatically.
func (m *Manager) LoadSkills(workingDir string, configDir string, cfg config.SkillsConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills = make(map[string]*Skill)

	candidates := []string{
		filepath.Join(workingDir, cfg.ActiveDir),
		filepath.Join(workingDir, cfg.CustomizedDir),
		filepath.Join(configDir, cfg.ActiveDir),
		filepath.Join(configDir, cfg.CustomizedDir),
	}

	seen := make(map[string]bool, len(candidates))
	for _, dir := range candidates {
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		if err := m.loadFromDir(dir); err != nil {
			return fmt.Errorf("load from %s: %w", dir, err)
		}
	}
	return nil
}

func (m *Manager) loadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				sk, err := loadSkill(skillPath)
				if err != nil {
					continue
				}
				sk.Enabled = true
				m.skills[sk.Name] = sk
			}
		}
	}
	return nil
}

type skillFrontMatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
}

func loadSkill(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var fm skillFrontMatter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 2 {
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
			fm.Name = filepath.Base(filepath.Dir(path))
		}
		if len(parts) == 3 {
			content = strings.TrimSpace(parts[2])
		}
	}
	if fm.Name == "" {
		fm.Name = filepath.Base(filepath.Dir(path))
	}

	skillDir := filepath.Dir(path)

	skill := &Skill{
		Name:        fm.Name,
		Description: fm.Description,
		Keywords:    fm.Keywords,
		Content:     content,
		Path:        path,
		Scripts:     loadDirectoryTextFiles(filepath.Join(skillDir, "scripts")),
		References:  loadDirectoryTextFiles(filepath.Join(skillDir, "references")),
		ExtraFiles:  loadDirectoryBinaryFiles(filepath.Join(skillDir, "extra_files")),
	}

	return skill, nil
}

// loadDirectoryTextFiles loads all text files from a directory into a map.
func loadDirectoryTextFiles(dir string) map[string]string {
	result := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		result[e.Name()] = string(data)
	}
	return result
}

// loadDirectoryBinaryFiles loads all files (binary or text) from a directory into a map.
func loadDirectoryBinaryFiles(dir string) map[string][]byte {
	result := make(map[string][]byte)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		result[e.Name()] = data
	}
	return result
}

// GetEnabledSkills returns all enabled skills.
func (m *Manager) GetEnabledSkills() []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Skill
	for _, s := range m.skills {
		if s.Enabled {
			out = append(out, *s)
		}
	}
	return out
}

// EnableSkill enables a skill by name.
func (m *Manager) EnableSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.skills[name]; ok {
		s.Enabled = true
		return nil
	}
	return fmt.Errorf("skill %q not found", name)
}

// DisableSkill disables a skill by name.
func (m *Manager) DisableSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.skills[name]; ok {
		s.Enabled = false
		return nil
	}
	return fmt.Errorf("skill %q not found", name)
}

// ImportFromURL downloads SKILL.md from the given URL and saves to customized_skills/{name}/SKILL.md.
// Supports raw.githubusercontent.com URLs. The name is derived from the URL path or can be provided.
func (m *Manager) ImportFromURL(ctx context.Context, url string, workingDir string, cfg config.SkillsConfig) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	content := string(body)
	if len(content) < 10 {
		return "", fmt.Errorf("content too short")
	}
	name := deriveSkillNameFromURL(url)
	customDir := filepath.Join(workingDir, cfg.CustomizedDir, name)
	if err := os.MkdirAll(customDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	skillPath := filepath.Join(customDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	m.mu.Lock()
	sk, err := loadSkill(skillPath)
	if err != nil {
		m.mu.Unlock()
		return name, nil
	}
	sk.Enabled = true
	m.skills[sk.Name] = sk
	m.mu.Unlock()
	return name, nil
}

func deriveSkillNameFromURL(url string) string {
	parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "SKILL.md" && i > 0 {
			return parts[i-1]
		}
		if parts[i] != "" && parts[i] != "raw" && parts[i] != "github.com" && parts[i] != "blob" {
			return strings.TrimSuffix(parts[i], ".md")
		}
	}
	return "imported"
}

// ListAllSkills returns all loaded skills (enabled and disabled).
func (m *Manager) ListAllSkills() []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Skill, 0, len(m.skills))
	for _, s := range m.skills {
		out = append(out, *s)
	}
	return out
}

// GetSkill returns a skill by name, or nil if not found.
func (m *Manager) GetSkill(name string) *Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.skills[name]; ok {
		cp := *s
		return &cp
	}
	return nil
}

// CreateSkill creates a new skill with the given name and content in the customized_skills directory.
func (m *Manager) CreateSkill(workingDir string, cfg config.SkillsConfig, name string, description string, content string) error {
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	skillDir := filepath.Join(workingDir, cfg.CustomizedDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", name))
	if description != "" {
		sb.WriteString(fmt.Sprintf("description: %s\n", description))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(content)

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	sk, err := loadSkill(skillPath)
	if err != nil {
		return fmt.Errorf("parse created skill: %w", err)
	}
	sk.Enabled = true
	m.skills[sk.Name] = sk
	return nil
}

// DeleteSkill removes a skill by name from the manager and optionally from disk.
func (m *Manager) DeleteSkill(name string, removeFromDisk bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sk, ok := m.skills[name]
	if !ok {
		return fmt.Errorf("skill %q not found", name)
	}

	if removeFromDisk && sk.Path != "" {
		dir := filepath.Dir(sk.Path)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove skill dir: %w", err)
		}
	}

	delete(m.skills, name)
	return nil
}

// SyncSkillsToWorkingDir copies built-in skills from configDir to workingDir
// if they don't already exist there.
func (m *Manager) SyncSkillsToWorkingDir(workingDir string, configDir string, cfg config.SkillsConfig) error {
	srcDir := filepath.Join(configDir, cfg.ActiveDir)
	dstDir := filepath.Join(workingDir, cfg.ActiveDir)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read source dir: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		srcSkill := filepath.Join(srcDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(srcSkill); os.IsNotExist(err) {
			continue
		}

		dstSkill := filepath.Join(dstDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(dstSkill); err == nil {
			continue // already exists
		}

		data, err := os.ReadFile(srcSkill)
		if err != nil {
			continue
		}

		dstSkillDir := filepath.Join(dstDir, e.Name())
		if err := os.MkdirAll(dstSkillDir, 0755); err != nil {
			continue
		}
		if err := os.WriteFile(dstSkill, data, 0644); err != nil {
			continue
		}
	}
	return nil
}

// ListAvailableSkills returns names of all skills found in the given directories
// (both enabled and not yet loaded).
func (m *Manager) ListAvailableSkills(workingDir string, configDir string, cfg config.SkillsConfig) []string {
	dirs := []string{
		filepath.Join(workingDir, cfg.ActiveDir),
		filepath.Join(workingDir, cfg.CustomizedDir),
		filepath.Join(configDir, cfg.ActiveDir),
		filepath.Join(configDir, cfg.CustomizedDir),
	}

	seen := make(map[string]bool)
	var names []string

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				continue
			}
			if !seen[e.Name()] {
				seen[e.Name()] = true
				names = append(names, e.Name())
			}
		}
	}
	return names
}

// GetSystemPromptAddition returns concatenated content of all enabled skills.
func (m *Manager) GetSystemPromptAddition() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var sb strings.Builder
	for _, s := range m.skills {
		if s.Enabled && s.Content != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString("--- ")
			sb.WriteString(s.Name)
			if s.Description != "" {
				sb.WriteString(": ")
				sb.WriteString(s.Description)
			}
			sb.WriteString(" ---\n")
			sb.WriteString(s.Content)
		}
	}
	return sb.String()
}

// GetSkillIndex returns AgentScope-style skill index with name, description, and path.
// This is a lazy-loading approach: the index tells the LLM what skills are available,
// but the LLM must use the read_file tool to get the full SKILL.md content.
// This reduces system prompt size and ensures skills are loaded on-demand.
// The workingDir parameter is used to convert absolute paths to relative paths.
func (m *Manager) GetSkillIndex(workingDir string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	// Use a prominent warning box at the top
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  ⚠️  CRITICAL: CHECK AVAILABLE SKILLS BEFORE USING ANY TOOLS                ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")

	sb.WriteString("**Before starting any task, you MUST check if there's a relevant skill below.**\n\n")
	sb.WriteString("**To use a skill, call the `read_file` tool with the skill's path.**\n\n")

	// Provide a clear example
	sb.WriteString("📖 **EXAMPLE**: To create a Word document:\n")
	sb.WriteString("   1. Check the skills list below for `docx` skill\n")
	sb.WriteString("   2. Call: `read_file` with `file_path=\"configs/active_skills/docx/SKILL.md\"`\n")
	sb.WriteString("   3. Follow the instructions in the skill file\n\n")

	sb.WriteString("─────────────────────────────────────────────────────────────────────────────────\n\n")

	for _, s := range m.skills {
		if !s.Enabled {
			continue
		}

		// Use emoji for better visibility
		sb.WriteString(fmt.Sprintf("🔹 **%s**\n\n", s.Name))

		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("   *%s*\n\n", s.Description))
		}

		// Convert absolute path to relative path for LLM's read_file tool
		relPath := s.Path
		if workingDir != "" {
			if rp, err := filepath.Rel(workingDir, s.Path); err == nil {
				relPath = rp
			}
		}
		sb.WriteString(fmt.Sprintf("   📂 Path: `%s`\n\n", relPath))
		sb.WriteString(fmt.Sprintf("   💡 To use: `read_file` with `file_path=\"%s\"`\n\n", relPath))

		// Show keywords if available
		if len(s.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf("   🏷️  Keywords: %s\n\n", strings.Join(s.Keywords, ", ")))
		}

		sb.WriteString("─────────────────────────────────────────────────────────────────────────────────\n\n")
	}

	return sb.String()
}

// GetSkillIndexCompact returns a compact AgentScope-style skill index.
// This version dynamically lists all enabled skills (not hardcoded).
func (m *Manager) GetSkillIndexCompact(workingDir string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	// Use a more compact format
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  ⚠️  CHECK SKILLS BEFORE CREATING DOCUMENTS                                  ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")

	// Count enabled skills
	var enabledSkills []string
	for name, s := range m.skills {
		if s.Enabled {
			enabledSkills = append(enabledSkills, name)
		}
	}

	if len(enabledSkills) == 0 {
		sb.WriteString("No skills currently enabled.\n")
		return sb.String()
	}

	// Build table of all enabled skills
	sb.WriteString("┌─────────────────┬──────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│ Skill           │ How to Load                                                  │\n")
	sb.WriteString("├─────────────────┼──────────────────────────────────────────────────────────────┤\n")

	for _, name := range enabledSkills {
		s := m.skills[name]
		relPath := s.Path
		if workingDir != "" {
			if rp, err := filepath.Rel(workingDir, s.Path); err == nil {
				relPath = rp
			}
		}

		// Truncate description if too long
		desc := s.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		sb.WriteString(fmt.Sprintf("│ %-15s │ %-60s │\n", name, desc))
		sb.WriteString(fmt.Sprintf("│                 │ read_file \"%s\"%*s│\n",
			relPath, 60-len(relPath), ""))
		sb.WriteString("├─────────────────┼──────────────────────────────────────────────────────────────┤\n")
	}

	sb.WriteString("└─────────────────┴──────────────────────────────────────────────────────────────┘\n\n")

	sb.WriteString("**Before creating documents, read the relevant SKILL.md file first!**\n")

	return sb.String()
}

// SelectSkills returns skills that match the given query based on
// name, description, and keywords. Uses case-insensitive matching.
func (m *Manager) SelectSkills(query string) []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if query == "" {
		// Empty query: return all enabled skills
		return m.GetEnabledSkills()
	}

	query = strings.ToLower(query)
	var matched []Skill

	for _, s := range m.skills {
		if !s.Enabled {
			continue
		}

		// Check name match
		if strings.Contains(query, strings.ToLower(s.Name)) {
			matched = append(matched, *s)
			continue
		}

		// Check description match
		if s.Description != "" {
			desc := strings.ToLower(s.Description)
			// Split description into words and check each
			descWords := strings.Fields(desc)
			for _, word := range descWords {
				cleanWord := strings.Trim(word, ".,!?;:'\"()[]{}")
				if cleanWord != "" && strings.Contains(query, cleanWord) {
					matched = append(matched, *s)
					goto nextSkill
				}
			}
		}

		// Check keywords match
		if len(s.Keywords) > 0 {
			for _, keyword := range s.Keywords {
				lowerKeyword := strings.ToLower(keyword)
				if strings.Contains(query, lowerKeyword) {
					matched = append(matched, *s)
					goto nextSkill
				}
			}
		}

	nextSkill:
	}

	return matched
}

// GetRelevantSkillsContent returns the content of skills that match the query.
// If no skills match, falls back to returning all enabled skills content.
func (m *Manager) GetRelevantSkillsContent(query string) string {
	matched := m.SelectSkills(query)
	if len(matched) == 0 {
		// Fallback: return all enabled skills
		return m.GetSystemPromptAddition()
	}

	var sb strings.Builder
	for _, skill := range matched {
		if skill.Content == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("--- ")
		sb.WriteString(skill.Name)
		if skill.Description != "" {
			sb.WriteString(": ")
			sb.WriteString(skill.Description)
		}
		sb.WriteString(" ---\n")
		sb.WriteString(skill.Content)
	}
	return sb.String()
}

// SetQuery sets the current user query for dynamic skill selection.
// This allows GetDynamicSystemPromptAddition to return relevant skills.
func (m *Manager) SetQuery(query string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentQuery = strings.ToLower(query)
}

// GetDynamicSystemPromptAddition returns the content of skills that match
// the current query set by SetQuery. If no query is set or no skills match,
// returns all enabled skills content.
func (m *Manager) GetDynamicSystemPromptAddition() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentQuery == "" {
		return m.GetSystemPromptAddition()
	}

	return m.GetRelevantSkillsContent(m.currentQuery)
}

// ClearQuery clears the current query.
func (m *Manager) ClearQuery() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentQuery = ""
}
