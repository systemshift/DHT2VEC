// Package semantic implements parameterized kernel-based similarity for TERA.
//
// Unlike fixed embedding models (ResNet, BERT), this approach extracts universal
// features and allows users to parameterize similarity at query time.
package semantic

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Features represents extracted features from content.
// These are universal (model-independent) and can be combined via kernel functions.
type Features struct {
	// TF-IDF vector (sparse representation)
	TFIDF map[string]float64

	// Character n-grams for lexical similarity
	Ngrams map[string]bool

	// Document statistics
	WordCount  int
	CharCount  int
	UniqueWords int

	// Top keywords (for debugging/display)
	TopKeywords []string
}

// Tokenize splits text into words (lowercased, alphanumeric only).
func Tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split by whitespace and punctuation
	words := []string{}
	currentWord := strings.Builder{}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			currentWord.WriteRune(r)
		} else {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
	}

	// Don't forget last word
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// GenerateNgrams creates character n-grams from text (n=3 by default).
func GenerateNgrams(text string, n int) map[string]bool {
	if n <= 0 {
		n = 3
	}

	text = strings.ToLower(text)
	ngrams := make(map[string]bool)

	if len(text) < n {
		ngrams[text] = true
		return ngrams
	}

	for i := 0; i <= len(text)-n; i++ {
		ngram := text[i : i+n]
		ngrams[ngram] = true
	}

	return ngrams
}

// ComputeTF calculates term frequency (normalized by document length).
func ComputeTF(words []string) map[string]float64 {
	tf := make(map[string]float64)
	total := float64(len(words))

	if total == 0 {
		return tf
	}

	// Count occurrences
	counts := make(map[string]int)
	for _, word := range words {
		counts[word]++
	}

	// Normalize by total words
	for word, count := range counts {
		tf[word] = float64(count) / total
	}

	return tf
}

// IDF stores inverse document frequencies.
// In a real system, this would be computed from a corpus.
// For MVP, we use a simplified version.
type IDF struct {
	frequencies map[string]float64
}

// NewIDF creates a new IDF structure.
func NewIDF() *IDF {
	return &IDF{
		frequencies: make(map[string]float64),
	}
}

// Add adds a document to the IDF corpus.
func (idf *IDF) Add(words []string) {
	seen := make(map[string]bool)
	for _, word := range words {
		if !seen[word] {
			idf.frequencies[word]++
			seen[word] = true
		}
	}
}

// Get returns the IDF value for a word.
// IDF = log(total_docs / docs_containing_word)
func (idf *IDF) Get(word string, totalDocs int) float64 {
	if totalDocs == 0 {
		return 0
	}

	freq := idf.frequencies[word]
	if freq == 0 {
		// Word not seen in corpus - give it low IDF
		return math.Log(float64(totalDocs) + 1)
	}

	return math.Log(float64(totalDocs) / freq)
}

// ExtractFeatures extracts features from raw content.
// This is the main entry point for feature extraction.
func ExtractFeatures(content []byte) *Features {
	text := string(content)
	words := Tokenize(text)

	// Compute TF (we'll use simplified TF without corpus-wide IDF for MVP)
	tf := ComputeTF(words)

	// Generate n-grams
	ngrams := GenerateNgrams(text, 3)

	// Compute statistics
	uniqueWords := len(tf)

	// Get top keywords (highest TF)
	topKeywords := getTopKeywords(tf, 10)

	return &Features{
		TFIDF:       tf, // For MVP, this is just TF (can extend with IDF later)
		Ngrams:      ngrams,
		WordCount:   len(words),
		CharCount:   len(text),
		UniqueWords: uniqueWords,
		TopKeywords: topKeywords,
	}
}

// getTopKeywords returns the top N words by frequency.
func getTopKeywords(tf map[string]float64, n int) []string {
	type kv struct {
		Key   string
		Value float64
	}

	// Convert to slice
	var pairs []kv
	for k, v := range tf {
		pairs = append(pairs, kv{k, v})
	}

	// Sort by value (descending)
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	// Take top N
	result := []string{}
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].Key)
	}

	return result
}

// CosineSimilarity computes cosine similarity between two TF-IDF vectors.
func CosineSimilarity(a, b map[string]float64) float64 {
	// Compute dot product and magnitudes
	var dotProduct, magA, magB float64

	// Build union of keys
	allKeys := make(map[string]bool)
	for k := range a {
		allKeys[k] = true
	}
	for k := range b {
		allKeys[k] = true
	}

	// Compute
	for k := range allKeys {
		valA := a[k]
		valB := b[k]

		dotProduct += valA * valB
		magA += valA * valA
		magB += valB * valB
	}

	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)

	if magA == 0 || magB == 0 {
		return 0
	}

	return dotProduct / (magA * magB)
}

// JaccardSimilarity computes Jaccard similarity between two sets.
func JaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // Both empty
	}

	// Compute intersection and union
	intersection := 0
	union := make(map[string]bool)

	for k := range a {
		union[k] = true
	}
	for k := range b {
		union[k] = true
		if a[k] {
			intersection++
		}
	}

	if len(union) == 0 {
		return 0
	}

	return float64(intersection) / float64(len(union))
}

// StructuralSimilarity compares document structure (word count, unique words).
func StructuralSimilarity(a, b *Features) float64 {
	// Normalize by max to get [0, 1] range
	wordCountDiff := math.Abs(float64(a.WordCount - b.WordCount))
	maxWordCount := math.Max(float64(a.WordCount), float64(b.WordCount))

	uniqueWordsDiff := math.Abs(float64(a.UniqueWords - b.UniqueWords))
	maxUniqueWords := math.Max(float64(a.UniqueWords), float64(b.UniqueWords))

	if maxWordCount == 0 && maxUniqueWords == 0 {
		return 1.0
	}

	// Average of similarities (1 - normalized difference)
	similarities := []float64{}

	if maxWordCount > 0 {
		similarities = append(similarities, 1.0-wordCountDiff/maxWordCount)
	}
	if maxUniqueWords > 0 {
		similarities = append(similarities, 1.0-uniqueWordsDiff/maxUniqueWords)
	}

	sum := 0.0
	for _, s := range similarities {
		sum += s
	}

	return sum / float64(len(similarities))
}
