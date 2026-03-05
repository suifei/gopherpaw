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
		home, _ := os.UserHomeDir()
		if home != "" {
			workingDir = filepath.Join(home, ".gopherpaw")
		} else {
			workingDir = "."
		}
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

// BuildSystemPrompt concatenates: SOUL + AGENTS + PROFILE + today MEMORY + skillsContent.
// Returns empty string if SOUL and AGENTS are missing (caller should use fallback).
func (p *PromptLoader) BuildSystemPrompt(skillsContent string) string {
	var parts []string

	soul, err := p.LoadSOUL()
	if err != nil || soul == "" {
		return ""
	}
	parts = append(parts, soul)

	agents, err := p.LoadAGENTS()
	if err != nil || agents == "" {
		return ""
	}
	parts = append(parts, agents)

	profile, _ := p.LoadPROFILE()
	if profile != "" {
		parts = append(parts, profile)
	}

	memMain, _ := p.readFile(fileMEMORY)
	if memMain != "" {
		parts = append(parts, memMain)
	}
	memToday, _ := p.readTodayMemory()
	if memToday != "" {
		parts = append(parts, memToday)
	}

	if skillsContent != "" {
		parts = append(parts, strings.TrimSpace(skillsContent))
	}

	return strings.Join(parts, "\n\n")
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
	return strings.TrimSpace(string(data)), nil
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
	return strings.TrimSpace(string(data)), nil
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
