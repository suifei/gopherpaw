// Package skills provides skill hub search and installation from remote sources.
package skills

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// HubSkillResult represents a skill found via hub search.
type HubSkillResult struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	SourceURL   string `json:"source_url"`
}

// HubInstallResult represents the result of installing a skill from hub.
type HubInstallResult struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	SourceURL string `json:"source_url"`
}

// HubConfig holds configurable parameters for hub HTTP operations.
type HubConfig struct {
	BaseURL     string
	SearchPath  string
	Timeout     time.Duration
	MaxRetries  int
	BackoffBase float64
	BackoffCap  float64
}

// DefaultHubConfig returns the default hub configuration.
func DefaultHubConfig() HubConfig {
	return HubConfig{
		BaseURL:     envOrDefault("GOPHERPAW_SKILLS_HUB_URL", "https://clawhub.ai"),
		SearchPath:  "/api/v1/search",
		Timeout:     envDurationOrDefault("GOPHERPAW_SKILLS_HUB_HTTP_TIMEOUT", 15*time.Second),
		MaxRetries:  envIntOrDefault("GOPHERPAW_SKILLS_HUB_HTTP_RETRIES", 3),
		BackoffBase: 0.8,
		BackoffCap:  6.0,
	}
}

var retryableStatus = map[int]bool{
	408: true, 409: true, 425: true, 429: true,
	500: true, 502: true, 503: true, 504: true,
}

// SearchHubSkills searches the skill hub for skills matching the query.
func SearchHubSkills(ctx context.Context, query string, limit int) ([]HubSkillResult, error) {
	cfg := DefaultHubConfig()
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", fmt.Sprintf("%d", limit))

	searchURL := joinURL(cfg.BaseURL, cfg.SearchPath) + "?" + params.Encode()
	body, err := hubHTTPGet(ctx, cfg, searchURL, "application/json")
	if err != nil {
		return nil, fmt.Errorf("hub search: %w", err)
	}

	return normSearchItems(body)
}

// InstallSkillFromHub downloads and installs a skill from a URL.
func (m *Manager) InstallSkillFromHub(ctx context.Context, bundleURL string, workingDir string, cfg SkillInstallConfig) (*HubInstallResult, error) {
	log := logger.L()

	name, content, err := fetchSkillBundle(ctx, bundleURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bundle: %w", err)
	}
	if content == "" {
		return nil, fmt.Errorf("empty skill content from %s", bundleURL)
	}

	targetDir := filepath.Join(workingDir, cfg.CustomizedDir, name)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	skillPath := filepath.Join(targetDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("write skill: %w", err)
	}

	m.mu.Lock()
	sk, err := loadSkill(skillPath)
	if err != nil {
		m.mu.Unlock()
		log.Warn("parse installed skill", zap.Error(err))
		return &HubInstallResult{Name: name, Enabled: true, SourceURL: bundleURL}, nil
	}
	sk.Enabled = true
	m.skills[sk.Name] = sk
	m.mu.Unlock()

	log.Info("Skill installed from hub", zap.String("name", sk.Name), zap.String("url", bundleURL))
	return &HubInstallResult{Name: sk.Name, Enabled: true, SourceURL: bundleURL}, nil
}

// SkillInstallConfig holds directory paths for skill installation.
type SkillInstallConfig struct {
	ActiveDir     string
	CustomizedDir string
}

// fetchSkillBundle fetches a skill bundle from a URL, dispatching by source type.
func fetchSkillBundle(ctx context.Context, bundleURL string) (name, content string, err error) {
	parsed, err := url.Parse(bundleURL)
	if err != nil {
		return "", "", fmt.Errorf("parse url: %w", err)
	}

	host := strings.ToLower(parsed.Host)

	switch {
	case strings.Contains(host, "github.com") || strings.Contains(host, "raw.githubusercontent.com"):
		return fetchFromGitHub(ctx, bundleURL)
	case strings.Contains(host, "clawhub.ai"):
		return fetchFromClawHub(ctx, bundleURL)
	case strings.Contains(host, "skills.sh"):
		return fetchFromSkillsSh(ctx, bundleURL)
	default:
		return fetchRawSkillMD(ctx, bundleURL)
	}
}

// fetchFromGitHub fetches a SKILL.md from a GitHub URL.
func fetchFromGitHub(ctx context.Context, rawURL string) (string, string, error) {
	cfg := DefaultHubConfig()

	ghURL := rawURL
	if strings.Contains(rawURL, "github.com") && !strings.Contains(rawURL, "raw.githubusercontent.com") {
		ghURL = strings.Replace(rawURL, "github.com", "raw.githubusercontent.com", 1)
		ghURL = strings.Replace(ghURL, "/blob/", "/", 1)
	}

	body, err := hubHTTPGet(ctx, cfg, ghURL, "text/plain")
	if err != nil {
		return "", "", err
	}

	name := deriveNameFromGitHubURL(rawURL)
	return name, body, nil
}

// fetchFromClawHub fetches a skill from ClawHub API.
func fetchFromClawHub(ctx context.Context, bundleURL string) (string, string, error) {
	cfg := DefaultHubConfig()

	slug := extractClawHubSlug(bundleURL)
	if slug == "" {
		return fetchRawSkillMD(ctx, bundleURL)
	}

	fileURL := joinURL(cfg.BaseURL, fmt.Sprintf("/api/v1/skills/%s/file", slug))
	body, err := hubHTTPGet(ctx, cfg, fileURL, "application/json")
	if err != nil {
		return fetchRawSkillMD(ctx, bundleURL)
	}

	var resp struct {
		Name    string `json:"name"`
		Content string `json:"content"`
		Skill   string `json:"skill"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return slug, body, nil
	}

	content := resp.Content
	if content == "" {
		content = resp.Skill
	}
	name := resp.Name
	if name == "" {
		name = slug
	}
	return name, content, nil
}

// fetchFromSkillsSh fetches a skill from skills.sh.
func fetchFromSkillsSh(ctx context.Context, bundleURL string) (string, string, error) {
	cfg := DefaultHubConfig()

	body, err := hubHTTPGet(ctx, cfg, bundleURL, "application/json")
	if err != nil {
		return fetchRawSkillMD(ctx, bundleURL)
	}

	var resp struct {
		Name    string `json:"name"`
		Content string `json:"content"`
		Slug    string `json:"slug"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return "", "", fmt.Errorf("parse skills.sh response: %w", err)
	}

	name := resp.Name
	if name == "" {
		name = resp.Slug
	}
	if name == "" {
		name = "imported"
	}
	return name, resp.Content, nil
}

// fetchRawSkillMD fetches a raw SKILL.md file from any URL.
func fetchRawSkillMD(ctx context.Context, rawURL string) (string, string, error) {
	cfg := DefaultHubConfig()
	body, err := hubHTTPGet(ctx, cfg, rawURL, "")
	if err != nil {
		return "", "", err
	}
	name := deriveSkillNameFromURL(rawURL)
	return name, body, nil
}

// hubHTTPGet performs an HTTP GET with retries and exponential backoff.
func hubHTTPGet(ctx context.Context, cfg HubConfig, targetURL string, accept string) (string, error) {
	client := &http.Client{Timeout: cfg.Timeout}

	ghToken := os.Getenv("GITHUB_TOKEN")

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := computeBackoff(attempt, cfg.BackoffBase, cfg.BackoffCap)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		req.Header.Set("User-Agent", "GopherPaw/1.0")
		if ghToken != "" && (strings.Contains(targetURL, "github.com") || strings.Contains(targetURL, "githubusercontent.com")) {
			req.Header.Set("Authorization", "token "+ghToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			if attempt < cfg.MaxRetries {
				continue
			}
			return "", fmt.Errorf("http get: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if readErr != nil {
				return "", fmt.Errorf("read body: %w", readErr)
			}
			return string(body), nil
		}

		if retryableStatus[resp.StatusCode] && attempt < cfg.MaxRetries {
			continue
		}
		return "", fmt.Errorf("http %d from %s", resp.StatusCode, targetURL)
	}
	return "", fmt.Errorf("max retries exceeded for %s", targetURL)
}

// normSearchItems parses hub search response into HubSkillResult list.
func normSearchItems(body string) ([]HubSkillResult, error) {
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		var wrapper struct {
			Results []map[string]any `json:"results"`
			Data    []map[string]any `json:"data"`
			Skills  []map[string]any `json:"skills"`
			Items   []map[string]any `json:"items"`
		}
		if err2 := json.Unmarshal(raw, &wrapper); err2 != nil {
			return nil, fmt.Errorf("parse search items: %w", err)
		}
		for _, list := range [][]map[string]any{wrapper.Results, wrapper.Data, wrapper.Skills, wrapper.Items} {
			if len(list) > 0 {
				items = list
				break
			}
		}
	}

	results := make([]HubSkillResult, 0, len(items))
	for _, item := range items {
		r := HubSkillResult{
			Slug:        jsonStr(item, "slug"),
			Name:        jsonStr(item, "name"),
			Description: jsonStr(item, "description"),
			Version:     jsonStr(item, "version"),
			SourceURL:   jsonStr(item, "source_url"),
		}
		if r.Name == "" {
			r.Name = r.Slug
		}
		if r.Name != "" {
			results = append(results, r)
		}
	}
	return results, nil
}

// computeBackoff returns the backoff duration for the given attempt.
func computeBackoff(attempt int, base, cap float64) time.Duration {
	secs := math.Min(cap, base*math.Pow(2, float64(attempt-1)))
	return time.Duration(secs * float64(time.Second))
}

func joinURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func extractClawHubSlug(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "skills" {
		return parts[1]
	}
	if len(parts) >= 1 {
		return parts[len(parts)-1]
	}
	return ""
}

var safeNameRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func deriveNameFromGitHubURL(rawURL string) string {
	parts := strings.Split(strings.TrimSuffix(rawURL, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "SKILL.md" && i > 0 {
			return safeNameRe.ReplaceAllString(parts[i-1], "_")
		}
	}
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && parts[i] != "raw" && parts[i] != "blob" {
			return safeNameRe.ReplaceAllString(strings.TrimSuffix(parts[i], ".md"), "_")
		}
	}
	return "imported"
}

func jsonStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDurationOrDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// DecodeBase64Content decodes base64-encoded file content from GitHub API.
func DecodeBase64Content(encoded string) (string, error) {
	cleaned := strings.ReplaceAll(encoded, "\n", "")
	data, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
