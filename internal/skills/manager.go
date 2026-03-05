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
	Content     string
	Enabled     bool
	Path        string
}

// Manager loads and manages skills from directories.
type Manager struct {
	mu     sync.RWMutex
	skills map[string]*Skill
}

// NewManager creates a new skill manager.
func NewManager() *Manager {
	return &Manager{
		skills: make(map[string]*Skill),
	}
}

// LoadSkills loads SKILL.md files from active_dir and customized_dir under workingDir.
func (m *Manager) LoadSkills(workingDir string, cfg config.SkillsConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills = make(map[string]*Skill)

	dirs := []string{
		filepath.Join(workingDir, cfg.ActiveDir),
		filepath.Join(workingDir, cfg.CustomizedDir),
	}
	for _, dir := range dirs {
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
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
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
	return &Skill{
		Name:        fm.Name,
		Description: fm.Description,
		Content:     content,
		Path:        path,
	}, nil
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
