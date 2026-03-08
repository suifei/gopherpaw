// Package skills provides skill selection and matching utilities.
package skills

import (
	"strings"
	"unicode"
)

// Selector provides advanced skill selection capabilities.
type Selector struct {
	skills map[string]*Skill
}

// NewSelector creates a new skill selector.
func NewSelector(skills map[string]*Skill) *Selector {
	return &Selector{
		skills: skills,
	}
}

// ScoreResult represents a skill with its match score.
type ScoreResult struct {
	Skill  *Skill
	Score  float64
	Reason string
}

// SelectByScore selects skills based on relevance scoring.
// Returns skills sorted by score in descending order.
func (s *Selector) SelectByScore(query string, threshold float64) []ScoreResult {
	if query == "" {
		return nil
	}

	query = strings.ToLower(query)
	queryWords := s.tokenize(query)

	var results []ScoreResult
	for _, skill := range s.skills {
		if !skill.Enabled {
			continue
		}

		score, reason := s.calculateScore(skill, query, queryWords)
		if score >= threshold {
			results = append(results, ScoreResult{
				Skill:  skill,
				Score:  score,
				Reason: reason,
			})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// calculateScore calculates a relevance score for a skill against the query.
// Returns a score between 0 and 1, plus a reason for the match.
func (s *Selector) calculateScore(skill *Skill, query string, queryWords []string) (float64, string) {
	var score float64
	var reason string

	// Exact name match: highest score
	if query == strings.ToLower(skill.Name) {
		return 1.0, "exact name match"
	}

	// Name contains match
	if strings.Contains(query, strings.ToLower(skill.Name)) {
		score += 0.8
		reason = "name contains"
	}

	// Description word matching
	if skill.Description != "" {
		desc := strings.ToLower(skill.Description)
		descWords := s.tokenize(desc)
		matches := 0
		for _, qw := range queryWords {
			for _, dw := range descWords {
				if qw == dw {
					matches++
					break
				}
			}
		}
		if matches > 0 {
			descScore := float64(matches) / float64(len(queryWords))
			if descScore > score {
				score = descScore
				reason = "description match"
			}
		}
	}

	// Keyword matching
	if len(skill.Keywords) > 0 {
		matches := 0
		for _, keyword := range skill.Keywords {
			lowerKeyword := strings.ToLower(keyword)
			for _, qw := range queryWords {
				if qw == lowerKeyword || strings.Contains(qw, lowerKeyword) || strings.Contains(lowerKeyword, qw) {
					matches++
					break
				}
			}
		}
		if matches > 0 {
			keywordScore := float64(matches) * 0.3
			if keywordScore > score {
				score = keywordScore
				reason = "keyword match"
			}
		}
	}

	// Substring matching in content (lower weight)
	if skill.Content != "" {
		content := strings.ToLower(skill.Content)
		for _, qw := range queryWords {
			if len(qw) >= 4 && strings.Contains(content, qw) {
				if score < 0.2 {
					score = 0.2
					reason = "content mention"
				}
				break
			}
		}
	}

	return score, reason
}

// tokenize splits text into words, removing punctuation and normalizing.
func (s *Selector) tokenize(text string) []string {
	var words []string
	currentWord := strings.Builder{}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			currentWord.WriteRune(r)
		} else if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
			currentWord.Reset()
		}
	}
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// CommonOfficeKeywords returns common keywords for Office document skills.
// This can be used to enhance matching for skills without explicit keywords.
func CommonOfficeKeywords() map[string][]string {
	return map[string][]string{
		"docx": {
			"word", "docx", "doc", "document", "office", "microsoft",
			"报告", "文档", "格式化", "模板", "页眉", "页脚",
			"table", "表格", "heading", "标题", "footnote", "脚注",
			".doc", ".docx",
		},
		"pptx": {
			"powerpoint", "pptx", "ppt", "presentation", "slide",
			"演示", "幻灯片", "office", "microsoft",
			"动画", "切换", "布局", "template", "模板",
			".ppt", ".pptx",
		},
		"xlsx": {
			"excel", "xlsx", "xls", "spreadsheet", "workbook",
			"表格", "电子表", "office", "microsoft",
			"chart", "图表", "formula", "公式", "pivot", "透视",
			".xls", ".xlsx",
		},
		"pdf": {
			"pdf", "portable", "document", "format",
			"acrobat", "adobe", "pdf文档",
			"提取", "转换", "merge", "合并", "split", "拆分",
			".pdf",
		},
	}
}
