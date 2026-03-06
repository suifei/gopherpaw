package memory

import (
	"math"
	"regexp"
	"strings"
	"sync"
)

// BM25 provides full-text search scoring using the Okapi BM25 algorithm.
// This implementation includes:
// - IDF (Inverse Document Frequency) for term rarity weighting
// - TF saturation (term frequency with diminishing returns)
// - Document length normalization
type BM25 struct {
	mu sync.RWMutex
	// k1 controls TF saturation. Higher values give more weight to term frequency.
	// Typical range: 1.2 - 2.0. Default: 1.5
	k1 float64
	// b controls length normalization. b=0 disables normalization, b=1 full normalization.
	// Typical range: 0.5 - 0.8. Default: 0.75
	b float64
	// Document corpus for IDF calculation
	documents []string
	// Cached statistics
	avgDocLen float64
	docCount  int
	// IDF cache: term -> IDF score
	idfCache map[string]float64
}

// BM25Option configures BM25 scorer.
type BM25Option func(*BM25)

// WithK1 sets the TF saturation parameter k1.
func WithK1(k1 float64) BM25Option {
	return func(b *BM25) {
		if k1 >= 0 {
			b.k1 = k1
		}
	}
}

// WithB sets the length normalization parameter b.
func WithB(b float64) BM25Option {
	return func(bm *BM25) {
		if b >= 0 && b <= 1 {
			bm.b = b
		}
	}
}

// NewBM25 creates a BM25 scorer with optional configuration.
func NewBM25(opts ...BM25Option) *BM25 {
	bm := &BM25{
		k1:       1.5,
		b:        0.75,
		idfCache: make(map[string]float64),
	}
	for _, opt := range opts {
		opt(bm)
	}
	return bm
}

// AddDocument adds a document to the corpus for IDF calculation.
// Call this method for each document before scoring to enable IDF weighting.
func (bm *BM25) AddDocument(doc string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.documents = append(bm.documents, doc)
	bm.docCount = len(bm.documents)
	bm.recalculateStats()
}

// AddDocuments adds multiple documents to the corpus.
func (bm *BM25) AddDocuments(docs []string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.documents = append(bm.documents, docs...)
	bm.docCount = len(bm.documents)
	bm.recalculateStats()
}

// SetCorpus replaces the entire document corpus.
func (bm *BM25) SetCorpus(docs []string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.documents = make([]string, len(docs))
	copy(bm.documents, docs)
	bm.docCount = len(bm.documents)
	bm.recalculateStats()
}

// ClearCorpus removes all documents from the corpus.
func (bm *BM25) ClearCorpus() {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.documents = nil
	bm.docCount = 0
	bm.avgDocLen = 0
	bm.idfCache = make(map[string]float64)
}

// recalculateStats updates cached statistics. Must be called with lock held.
func (bm *BM25) recalculateStats() {
	if bm.docCount == 0 {
		bm.avgDocLen = 0
		return
	}
	totalLen := 0
	for _, doc := range bm.documents {
		totalLen += len(tokenize(doc))
	}
	bm.avgDocLen = float64(totalLen) / float64(bm.docCount)
	// Clear IDF cache as document frequencies may have changed
	bm.idfCache = make(map[string]float64)
}

// computeIDF calculates IDF for a term. Must be called with at least read lock.
func (bm *BM25) computeIDF(term string) float64 {
	if bm.docCount == 0 {
		return 1.0
	}

	// Count documents containing the term
	docFreq := 0
	termLower := strings.ToLower(term)
	for _, doc := range bm.documents {
		if strings.Contains(strings.ToLower(doc), termLower) {
			docFreq++
		}
	}

	// IDF formula: log((N - n + 0.5) / (n + 0.5) + 1)
	// Where N = total documents, n = documents containing term
	// The +1 ensures non-negative IDF even for common terms
	n := float64(docFreq)
	N := float64(bm.docCount)
	idf := math.Log((N-n+0.5)/(n+0.5) + 1)
	return idf
}

// getIDF returns cached IDF or computes it. Must be called with at least read lock.
func (bm *BM25) getIDF(term string) float64 {
	if idf, ok := bm.idfCache[term]; ok {
		return idf
	}
	idf := bm.computeIDF(term)
	// Note: we don't update cache here to avoid write during read lock
	// The cache is updated during Score when we have write lock
	return idf
}

// Score computes BM25 score for content against query.
// Higher scores indicate better matches.
//
// The score is computed as:
//
//	sum over query terms of: IDF(term) * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * docLen / avgDocLen))
//
// Where:
//   - IDF(term) = inverse document frequency (term rarity)
//   - tf = term frequency in document
//   - k1 = TF saturation parameter
//   - b = length normalization parameter
//   - docLen = document length in tokens
//   - avgDocLen = average document length in corpus
func (bm *BM25) Score(query, content string) float64 {
	if query == "" || content == "" {
		return 0
	}

	bm.mu.RLock()
	k1 := bm.k1
	b := bm.b
	avgDocLen := bm.avgDocLen
	docCount := bm.docCount
	bm.mu.RUnlock()

	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)
	queryTerms := tokenize(queryLower)
	contentTokens := tokenize(contentLower)

	if len(queryTerms) == 0 {
		return 0
	}

	// Build term frequency map for content
	tfMap := make(map[string]int)
	for _, t := range contentTokens {
		tfMap[t]++
	}

	docLen := float64(len(contentTokens))
	if avgDocLen == 0 {
		avgDocLen = docLen // No corpus, use document's own length
	}

	// Compute BM25 score
	var score float64
	for _, term := range queryTerms {
		tf := float64(tfMap[term])
		if tf == 0 {
			// Check for partial matches (term contained in token)
			for token, count := range tfMap {
				if strings.Contains(token, term) || strings.Contains(term, token) {
					tf += float64(count) * 0.5 // Partial match gets 50% weight
				}
			}
		}

		// Get IDF (default to 1.0 if no corpus)
		var idf float64
		if docCount > 0 {
			bm.mu.RLock()
			idf = bm.getIDF(term)
			bm.mu.RUnlock()
		} else {
			idf = 1.0
		}

		// BM25 TF component with length normalization
		// tf_norm = tf * (k1 + 1) / (tf + k1 * (1 - b + b * docLen / avgDocLen))
		lengthNorm := 1 - b + b*(docLen/avgDocLen)
		tfNorm := (tf * (k1 + 1)) / (tf + k1*lengthNorm)

		score += idf * tfNorm
	}

	// Phrase bonus: if the entire query appears as a phrase, add bonus
	if strings.Contains(contentLower, queryLower) {
		score += 0.5
	}

	return score
}

// ScoreWithDetails returns the BM25 score along with detailed scoring components.
// Useful for debugging and understanding why certain documents rank higher.
func (bm *BM25) ScoreWithDetails(query, content string) (float64, map[string]float64) {
	details := make(map[string]float64)

	if query == "" || content == "" {
		return 0, details
	}

	bm.mu.RLock()
	k1 := bm.k1
	b := bm.b
	avgDocLen := bm.avgDocLen
	docCount := bm.docCount
	bm.mu.RUnlock()

	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)
	queryTerms := tokenize(queryLower)
	contentTokens := tokenize(contentLower)

	if len(queryTerms) == 0 {
		return 0, details
	}

	tfMap := make(map[string]int)
	for _, t := range contentTokens {
		tfMap[t]++
	}

	docLen := float64(len(contentTokens))
	if avgDocLen == 0 {
		avgDocLen = docLen
	}

	details["doc_length"] = docLen
	details["avg_doc_length"] = avgDocLen
	details["k1"] = k1
	details["b"] = b

	var score float64
	for _, term := range queryTerms {
		tf := float64(tfMap[term])
		if tf == 0 {
			for token, count := range tfMap {
				if strings.Contains(token, term) || strings.Contains(term, token) {
					tf += float64(count) * 0.5
				}
			}
		}

		var idf float64
		if docCount > 0 {
			bm.mu.RLock()
			idf = bm.getIDF(term)
			bm.mu.RUnlock()
		} else {
			idf = 1.0
		}

		lengthNorm := 1 - b + b*(docLen/avgDocLen)
		tfNorm := (tf * (k1 + 1)) / (tf + k1*lengthNorm)
		termScore := idf * tfNorm

		details["tf_"+term] = tf
		details["idf_"+term] = idf
		details["score_"+term] = termScore

		score += termScore
	}

	if strings.Contains(contentLower, queryLower) {
		details["phrase_bonus"] = 0.5
		score += 0.5
	}

	details["total_score"] = score
	return score, details
}

// tokenize splits text into tokens (alphanumeric sequences including CJK characters).
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

// tokenizeWithDuplicates returns all tokens including duplicates (for TF counting).
func tokenizeWithDuplicates(s string) []string {
	re := regexp.MustCompile(`[\p{L}\p{N}]+`)
	return re.FindAllString(s, -1)
}
