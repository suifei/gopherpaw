// Package tools provides built-in tools for the agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
)

const (
	defaultWebSearchMaxResults = 10
	baiduSearchURL            = "https://www.baidu.com/s"
	baiduUserAgent            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// baiduResult represents a single search result from Baidu.
type baiduResult struct {
	Title       string
	Description string
	URL         string
}

// WebSearchTool performs web search via Baidu (no API key required).
type WebSearchTool struct {
	client *http.Client
}

// NewWebSearchTool creates a new WebSearchTool with default config.
func NewWebSearchTool() (*WebSearchTool, error) {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

// Name returns the tool identifier.
func (t *WebSearchTool) Name() string { return "web_search" }

// Description returns a human-readable description.
func (t *WebSearchTool) Description() string {
	return "使用百度搜索引擎搜索实时信息（天气、新闻、事实等）。当你需要最新的、不在训练数据中的信息时使用此工具。无需 API Key。"
}

// Parameters returns the JSON Schema for tool parameters.
func (t *WebSearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "搜索查询词（例如：'长沙天气', '最新科技新闻'）",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "最大结果数量（默认 10，最多 15）",
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
		return "错误: 未提供搜索查询词。", nil
	}
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = defaultWebSearchMaxResults
	}
	if maxResults > 15 {
		maxResults = 15
	}

	// 构建 Baidu 搜索 URL
	searchURL := fmt.Sprintf("%s?wd=%s&rn=%d", baiduSearchURL, url.QueryEscape(query), maxResults)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，模拟浏览器访问
	req.Header.Set("User-Agent", baiduUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	// 发送请求
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("搜索返回状态码: %d", resp.StatusCode)
	}

	// 读取响应内容（处理 GBK 编码）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 尝试检测并转换编码（百度通常使用 GBK 或 GB18030）
	decodedBody, err := decodeGBK(body)
	if err != nil {
		// 如果 GBK 解码失败，尝试 UTF-8
		decodedBody = string(body)
	}

	// 解析搜索结果
	results := parseBaiduResults(decodedBody, maxResults)

	if len(results) == 0 {
		return fmt.Sprintf("未找到相关结果: %s", query), nil
	}

	// 格式化输出
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("百度搜索结果 - \"%s\"\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if r.Description != "" {
			// 清理描述中的 HTML 标签
			desc := cleanHTML(r.Description)
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", desc))
		}
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

// decodeGBK 尝试将 GBK/GB18030 编码的字节转换为 UTF-8 字符串
func decodeGBK(data []byte) (string, error) {
	// 现代百度搜索结果通常使用 UTF-8
	// 如果需要处理 GBK，可以使用 golang.org/x/text/encoding/simplifiedchinese
	// 这里简化处理，直接转换为字符串
	return string(data), nil
}

// parseBaiduResults 从 HTML 中解析搜索结果
func parseBaiduResults(htmlContent string, maxResults int) []baiduResult {
	var results []baiduResult

	// 百度搜索结果的正则表达式模式
	// 匹配格式: <div class="result ...">...<a href="...">标题</a>...<div class="c-abstract">描述</div>...</div>

	// 提取结果块（多种可能的 class 名称）
	// 注意：Go regexp 不支持 lookahead，使用更简单的模式
	resultPatterns := []string{
		`<div[^>]*class="result[^"]*"[^>]*>(.+?)</div>\s*</div>`,
		`<div[^>]*class="c-container[^"]*"[^>]*>(.+?)</div>\s*</div>`,
		`<div[^>]*data-tools="[^"]*"[^>]*>(.+?)</div>\s*</div>`,
	}

	for _, pattern := range resultPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(htmlContent, -1)

		for _, match := range matches {
			if len(match) > 1 {
				block := match[1]
				result := extractResultFromBlock(block)
				if result.Title != "" && len(results) < maxResults {
					results = append(results, result)
				}
			}
		}
		if len(results) >= maxResults {
			break
		}
	}

	// 如果正则解析失败，尝试更简单的提取方式
	if len(results) == 0 {
		results = extractResultsSimple(htmlContent, maxResults)
	}

	return results
}

// extractResultFromBlock 从单个结果块中提取信息
func extractResultFromBlock(block string) baiduResult {
	var result baiduResult

	// 提取标题和链接
	titleRe := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	titleMatches := titleRe.FindStringSubmatch(block)
	if len(titleMatches) >= 3 {
		result.URL = titleMatches[1]
		result.Title = cleanHTML(titleMatches[2])
	}

	// 提取描述（多种可能的 class 名称）
	descPatterns := []string{
		`<div[^>]*class="c-abstract[^"]*"[^>]*>(.*?)</div>`,
		`<div[^>]*class="c-span-last[^"]*"[^>]*>(.*?)</div>`,
		`<span[^>]*class="content-right_8Zs40[^"]*"[^>]*>(.*?)</span>`,
	}

	for _, pattern := range descPatterns {
		descRe := regexp.MustCompile(pattern)
		descMatches := descRe.FindStringSubmatch(block)
		if len(descMatches) >= 2 {
			result.Description = cleanHTML(descMatches[1])
			break
		}
	}

	return result
}

// extractResultsSimple 简单方式提取结果（备用方案）
func extractResultsSimple(htmlContent string, maxResults int) []baiduResult {
	var results []baiduResult

	// 查找所有链接
	linkRe := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	matches := linkRe.FindAllStringSubmatch(htmlContent, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 3 {
			href := match[1]
			title := cleanHTML(match[2])

			// 过滤非百度搜索结果链接
			if !strings.Contains(href, "baidu.com") &&
				!strings.Contains(href, "#") &&
				strings.HasPrefix(href, "http") &&
				title != "" &&
				len(title) > 3 &&
				!seen[href] {
				seen[href] = true
				results = append(results, baiduResult{
					Title: title,
					URL:   href,
				})
				if len(results) >= maxResults {
					break
				}
			}
		}
	}

	return results
}

// Ensure WebSearchTool implements agent.Tool.
var _ agent.Tool = (*WebSearchTool)(nil)
