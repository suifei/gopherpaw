package memory

import (
	"regexp"
	"strings"
)

// BM25 provides full-text search scoring.
// Uses simplified BM25: score = (hit_count / query_term_count) + phrase_bonus.
type BM25 struct{}

// NewBM25 creates a BM25 scorer.
func NewBM25() *BM25 {
	return &BM25{}
}

// Score computes BM25-style score for content against query.
// - Tokenizes query and content (lowercase, split on non-alphanumeric)
// - hit_count / query_term_count as base score
// - +0.2 if full query phrase appears (case-insensitive)
func (b *BM25) Score(query, content string) float64 {
	if query == "" || content == "" {
		return 0
	}
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)
	queryTerms := tokenize(queryLower)
	if len(queryTerms) == 0 {
		return 0
	}
	hits := 0
	for _, t := range queryTerms {
		if strings.Contains(contentLower, t) {
			hits++
		}
	}
	baseScore := float64(hits) / float64(len(queryTerms))
	phraseBonus := 0.0
	if strings.Contains(contentLower, queryLower) {
		phraseBonus = 0.2
	}
	return baseScore + phraseBonus
}

// tokenize splits text into tokens (alphanumeric sequences).
func tokenize(s string) []string {
	re := regexp.MustCompile(`[\p{L}\p{N}]+`)
	matches := re.FindAllString(s, -1)
	seen := make(map[string]bool)
	var out []string
	for _, m := range matches {
		if m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}
