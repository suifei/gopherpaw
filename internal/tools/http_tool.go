// Package tools provides built-in tools for the agent.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
)

const defaultHTTPTimeout = 6 * time.Minute

// HTTPTool performs HTTP requests (GET, POST, etc.).
type HTTPTool struct {
	client *http.Client
}

// NewHTTPTool creates a new HTTPTool with default config.
func NewHTTPTool() *HTTPTool {
	return &HTTPTool{
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

// Name returns the tool identifier.
func (t *HTTPTool) Name() string { return "http_request" }

// Description returns a human-readable description.
func (t *HTTPTool) Description() string {
	return "Make HTTP requests (GET, POST, PUT, DELETE) to any URL. Use for calling REST APIs, fetching web pages, or accessing external services. Supports JSON body for POST/PUT."
}

// Parameters returns the JSON Schema for tool parameters.
func (t *HTTPTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Full URL (e.g. https://api.example.com/data)",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method: GET, POST, PUT, DELETE (default GET)",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Request body for POST/PUT (JSON string or plain text)",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Optional HTTP headers as key-value pairs",
			},
		},
		"required": []string{"url"},
	}
}

type httpArgs struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

// Execute runs the tool.
func (t *HTTPTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args httpArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}
	url := strings.TrimSpace(args.URL)
	if url == "" {
		return "Error: No URL provided.", nil
	}
	method := strings.ToUpper(strings.TrimSpace(args.Method))
	if method == "" {
		method = http.MethodGet
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		// OK
	default:
		return fmt.Sprintf("Error: Unsupported HTTP method: %s", method), nil
	}

	var bodyReader io.Reader
	if args.Body != "" && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
		bodyReader = bytes.NewReader([]byte(args.Body))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// 设置默认中文 HTTP headers（操作系统级别的区域设置）
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}

	// 处理用户自定义 headers（会覆盖默认值）
	for k, v := range args.Headers {
		req.Header.Set(k, v)
	}
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Sprintf("Error: HTTP %d\n%s", resp.StatusCode, string(data)), nil
	}

	// 检查 Content-Type 并处理内容
	contentType := resp.Header.Get("Content-Type")
	body := string(data)

	// 如果是 HTML，自动提取文本内容
	if shouldExtractContent(contentType) {
		extracted := cleanHTML(body)
		// 如果提取后的内容非空且明显不同，使用提取结果
		if extracted != "" && extracted != body {
			body = fmt.Sprintf("[Extracted from %s]\n\n%s", contentType, extracted)
			// 如果原始内容太长，添加提示
			if len(extracted) < len(data) {
				body += fmt.Sprintf("\n\n[Note: Original content was %d bytes, extracted to %d characters]",
					len(data), len(extracted))
			}
		}
	}

	// 对 HTML 响应，如果提取后内容为空或太短，说明可能是动态页面
	if isHTMLContentType(contentType) && len(body) < 200 {
		body += "\n\n[Note: This appears to be a dynamic page. Consider using browser_use tool for JavaScript-heavy sites.]"
	}

	// Truncate very large responses
	const maxResponseSize = 64 * 1024
	if len(body) > maxResponseSize {
		return fmt.Sprintf("%s\n\n(Response truncated. Total %d bytes.)", body[:maxResponseSize], len(body)), nil
	}
	return body, nil
}

// Ensure HTTPTool implements agent.Tool.
var _ agent.Tool = (*HTTPTool)(nil)
