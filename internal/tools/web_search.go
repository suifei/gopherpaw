// Package tools provides built-in tools for the agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kuhahalong/ddgsearch"
	"github.com/suifei/gopherpaw/internal/agent"
)

const defaultWebSearchMaxResults = 12

// WebSearchTool performs web search via DuckDuckGo (no API key required).
type WebSearchTool struct {
	client *ddgsearch.DDGS
}

// NewWebSearchTool creates a new WebSearchTool with default config.
func NewWebSearchTool() (*WebSearchTool, error) {
	cfg := &ddgsearch.Config{
		Timeout:    15 * time.Second,
		MaxRetries: 2,
		Cache:      true,
	}
	client, err := ddgsearch.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create ddgsearch client: %w", err)
	}
	return &WebSearchTool{client: client}, nil
}

// Name returns the tool identifier.
func (t *WebSearchTool) Name() string { return "web_search" }

// Description returns a human-readable description.
func (t *WebSearchTool) Description() string {
	return "Search the web for real-time information (weather, news, facts). Use this when you need current or up-to-date information that is not in your training data. No API key required."
}

// Parameters returns the JSON Schema for tool parameters.
func (t *WebSearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query (e.g. '长沙天气', 'Beijing weather')",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results (default 8)",
			},
		},
		"required": []string{"query"},
	}
}

type webSearchArgs struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

// Execute runs the tool.
func (t *WebSearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args webSearchArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "Error: No search query provided.", nil
	}
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = defaultWebSearchMaxResults
	}
	if maxResults > 15 {
		maxResults = 15
	}

	params := &ddgsearch.SearchParams{
		Query:      query,
		Region:     ddgsearch.RegionCN,
		SafeSearch: ddgsearch.SafeSearchModerate,
		MaxResults: maxResults,
	}

	resp, err := t.client.Search(ctx, params)
	if err != nil {
		if ddgsearch.IsNoResultsErr(err) {
			return fmt.Sprintf("No results found for: %s", query), nil
		}
		return "", fmt.Errorf("web search: %w", err)
	}

	if resp == nil || len(resp.Results) == 0 {
		return fmt.Sprintf("No results found for: %s", query), nil
	}

	var sb strings.Builder
	for i, r := range resp.Results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Description))
		}
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

// Ensure WebSearchTool implements agent.Tool.
var _ agent.Tool = (*WebSearchTool)(nil)
