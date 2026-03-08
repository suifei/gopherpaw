// Package tools provides utility functions for built-in tools.
package tools

import (
	"regexp"
	"strings"
)

// cleanHTML 清理 HTML 标签和实体，提取纯文本内容。
func cleanHTML(s string) string {
	// 移除 script 和 style 标签及其内容
	scriptRe := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	s = scriptRe.ReplaceAllString(s, "")

	styleRe := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	s = styleRe.ReplaceAllString(s, "")

	// 移除 HTML 注释
	commentRe := regexp.MustCompile(`<!--.*?-->`)
	s = commentRe.ReplaceAllString(s, "")

	// 移除 HTML 标签
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, " ")

	// 替换 HTML 实体
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "\u003c", "<")
	s = strings.ReplaceAll(s, "\u003e", ">")
	s = strings.ReplaceAll(s, "&mdash;", "—")
	s = strings.ReplaceAll(s, "&ndash;", "-")
	s = strings.ReplaceAll(s, "&hellip;", "...")
	s = strings.ReplaceAll(s, "&rsquo;", "'")
	s = strings.ReplaceAll(s, "&lsquo;", "'")
	s = strings.ReplaceAll(s, "&rdquo;", "\"")
	s = strings.ReplaceAll(s, "&ldquo;", "\"")

	// 清理多余空格和换行
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	return s
}

// isHTMLContentType 检查是否是 HTML 内容类型。
func isHTMLContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") ||
		strings.Contains(ct, "application/xhtml")
}

// shouldExtractContent 判断是否应该提取内容（而非返回原始数据）。
func shouldExtractContent(contentType string) bool {
	ct := strings.ToLower(contentType)
	// HTML 需要提取文本
	if isHTMLContentType(ct) {
		return true
	}
	// XML 可能也需要提取
	if strings.Contains(ct, "text/xml") || strings.Contains(ct, "application/xml") {
		return true
	}
	// JSON 保持原样
	if strings.Contains(ct, "application/json") {
		return false
	}
	// 其他二进制或文本格式，根据前缀判断
	return strings.HasPrefix(ct, "text/")
}
