package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/suifei/gopherpaw/internal/agent"
)

const maxMatches = 200
const maxFileSize = 20 * 1024 * 1024 // 20MB

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".pdf": true,
	".zip": true, ".tar": true, ".gz": true, ".exe": true, ".dll": true,
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if binaryExts[ext] {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() <= maxFileSize
}

// GrepSearchTool searches file contents by pattern.
type GrepSearchTool struct{}

// Name returns the tool identifier.
func (t *GrepSearchTool) Name() string { return "grep_search" }

// Description returns a human-readable description.
func (t *GrepSearchTool) Description() string {
	return "Search file contents by pattern (or regex). Output format: path:line: content. Use path for directory or file."
}

// Parameters returns the JSON Schema.
func (t *GrepSearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern":     map[string]any{"type": "string", "description": "Search pattern"},
			"path":        map[string]any{"type": "string", "description": "Directory or file to search"},
			"is_regex":    map[string]any{"type": "boolean", "description": "Treat pattern as regex"},
			"case_sensitive": map[string]any{"type": "boolean", "description": "Case insensitive search"},
		},
		"required": []string{"pattern"},
	}
}

type grepArgs struct {
	Pattern       string `json:"pattern"`
	Path          string `json:"path"`
	IsRegex       bool   `json:"is_regex"`
	CaseSensitive bool   `json:"case_sensitive"`
}

// Execute runs the tool.
func (t *GrepSearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args grepArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Pattern == "" {
		return "Error: No pattern provided.", nil
	}
	searchRoot := workingDir
	if args.Path != "" {
		searchRoot = resolvePath(args.Path)
	}
	if !pathExists(searchRoot) {
		return fmt.Sprintf("Error: Path %s does not exist.", searchRoot), nil
	}
	regex := args.Pattern
	if !args.IsRegex {
		regex = regexp.QuoteMeta(args.Pattern)
	}
	var re *regexp.Regexp
	var err error
	if args.CaseSensitive {
		re, err = regexp.Compile(regex)
	} else {
		re, err = regexp.Compile("(?i)" + regex)
	}
	if err != nil {
		return fmt.Sprintf("Error: Invalid regex: %v", err), nil
	}

	var files []string
	info, _ := os.Stat(searchRoot)
	if info != nil && info.Mode().IsRegular() {
		files = []string{searchRoot}
	} else {
		filepath.Walk(searchRoot, func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			if isTextFile(p) {
				files = append(files, p)
			}
			return nil
		})
	}

	var matches []string
	truncated := false
	for _, fp := range files {
		if truncated {
			break
		}
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				if len(matches) >= maxMatches {
					truncated = true
					break
				}
				rel, _ := filepath.Rel(searchRoot, fp)
				if rel == "" {
					rel = filepath.Base(fp)
				}
				matches = append(matches, fmt.Sprintf("%s:%d: %s", rel, i+1, line))
			}
		}
	}
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for pattern: %s", args.Pattern), nil
	}
	result := strings.Join(matches, "\n")
	if truncated {
		result += fmt.Sprintf("\n\n(Results truncated at %d matches.)", maxMatches)
	}
	return result, nil
}

// GlobSearchTool finds files matching a glob pattern.
type GlobSearchTool struct{}

// Name returns the tool identifier.
func (t *GlobSearchTool) Name() string { return "glob_search" }

// Description returns a human-readable description.
func (t *GlobSearchTool) Description() string {
	return "Find files matching a glob pattern (e.g. *.go, **/*.json)."
}

// Parameters returns the JSON Schema.
func (t *GlobSearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string", "description": "Glob pattern"},
			"path":    map[string]any{"type": "string", "description": "Root directory"},
		},
		"required": []string{"pattern"},
	}
}

type globArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

// Execute runs the tool.
func (t *GlobSearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args globArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Pattern == "" {
		return "Error: No pattern provided.", nil
	}
	searchRoot := workingDir
	if args.Path != "" {
		searchRoot = resolvePath(args.Path)
	}
	if !pathExists(searchRoot) {
		return fmt.Sprintf("Error: Path %s does not exist.", searchRoot), nil
	}
	info, _ := os.Stat(searchRoot)
	if info != nil && !info.IsDir() {
		return "Error: Path is not a directory.", nil
	}
	var results []string
	filepath.Walk(searchRoot, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(searchRoot, p)
		if err != nil {
			return nil
		}
		matched, err := filepath.Match(args.Pattern, rel)
		if err != nil || !matched {
			matched, _ = filepath.Match(args.Pattern, filepath.Base(p))
		}
		if matched && len(results) < maxMatches {
			results = append(results, rel)
		}
		return nil
	})
	if len(results) == 0 {
		return fmt.Sprintf("No files matched pattern: %s", args.Pattern), nil
	}
	result := strings.Join(results, "\n")
	if len(results) >= maxMatches {
		result += fmt.Sprintf("\n\n(Results truncated at %d entries.)", maxMatches)
	}
	return result, nil
}

// Ensure interfaces.
var _ agent.Tool = (*GrepSearchTool)(nil)
var _ agent.Tool = (*GlobSearchTool)(nil)
