package semantic

import (
	"fmt"
	"math"
)

// KernelParams defines user-configurable similarity weights.
// Different queries can use different parameters to express different notions of "similarity".
type KernelParams struct {
	// WeightSemantic: how much to weight semantic similarity (TF-IDF cosine)
	WeightSemantic float64

	// WeightLexical: how much to weight lexical similarity (n-gram Jaccard)
	WeightLexical float64

	// WeightStructural: how much to weight structural similarity (word count, etc.)
	WeightStructural float64

	// Threshold: minimum similarity to consider a match
	Threshold float64
}

// DefaultParams returns reasonable default parameters.
func DefaultParams() KernelParams {
	return KernelParams{
		WeightSemantic:   0.6,
		WeightLexical:    0.3,
		WeightStructural: 0.1,
		Threshold:        0.5,
	}
}

// SemanticFocusedParams returns parameters optimized for semantic search.
func SemanticFocusedParams() KernelParams {
	return KernelParams{
		WeightSemantic:   0.8,
		WeightLexical:    0.15,
		WeightStructural: 0.05,
		Threshold:        0.6,
	}
}

// LexicalFocusedParams returns parameters optimized for exact matching.
func LexicalFocusedParams() KernelParams {
	return KernelParams{
		WeightSemantic:   0.2,
		WeightLexical:    0.7,
		WeightStructural: 0.1,
		Threshold:        0.5,
	}
}

// Validate checks that parameters are valid.
func (p KernelParams) Validate() error {
	if p.WeightSemantic < 0 || p.WeightLexical < 0 || p.WeightStructural < 0 {
		return fmt.Errorf("weights must be non-negative")
	}

	total := p.WeightSemantic + p.WeightLexical + p.WeightStructural
	if total == 0 {
		return fmt.Errorf("at least one weight must be positive")
	}

	if p.Threshold < 0 || p.Threshold > 1 {
		return fmt.Errorf("threshold must be in [0, 1]")
	}

	return nil
}

// Normalize ensures weights sum to 1.
func (p KernelParams) Normalize() KernelParams {
	total := p.WeightSemantic + p.WeightLexical + p.WeightStructural

	if total == 0 {
		// Return default if all zero
		return DefaultParams()
	}

	return KernelParams{
		WeightSemantic:   p.WeightSemantic / total,
		WeightLexical:    p.WeightLexical / total,
		WeightStructural: p.WeightStructural / total,
		Threshold:        p.Threshold,
	}
}

// Similarity computes parameterized similarity between two feature vectors.
//
// This is the core kernel function that enables users to express their own
// notion of "similarity" via parameter weights.
//
// Returns a value in [0, 1] where 1 is most similar.
func Similarity(a, b *Features, params KernelParams) float64 {
	// Normalize parameters
	params = params.Normalize()

	// Compute individual similarities
	semantic := CosineSimilarity(a.TFIDF, b.TFIDF)
	lexical := JaccardSimilarity(a.Ngrams, b.Ngrams)
	structural := StructuralSimilarity(a, b)

	// Weighted combination
	score := params.WeightSemantic*semantic +
		params.WeightLexical*lexical +
		params.WeightStructural*structural

	// Clamp to [0, 1]
	return math.Max(0, math.Min(1, score))
}

// IsRelevant checks if similarity exceeds the threshold.
func IsRelevant(a, b *Features, params KernelParams) bool {
	sim := Similarity(a, b, params)
	return sim >= params.Threshold
}

// Rank sorts a list of features by similarity to a query.
type RankedResult struct {
	Features   *Features
	Similarity float64
	Index      int // Original index
}

// RankBySimilarity ranks a list of feature vectors by similarity to query.
func RankBySimilarity(query *Features, candidates []*Features, params KernelParams) []RankedResult {
	results := make([]RankedResult, len(candidates))

	for i, candidate := range candidates {
		sim := Similarity(query, candidate, params)
		results[i] = RankedResult{
			Features:   candidate,
			Similarity: sim,
			Index:      i,
		}
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// FilterRelevant returns only candidates exceeding the similarity threshold.
func FilterRelevant(query *Features, candidates []*Features, params KernelParams) []*Features {
	var relevant []*Features

	for _, candidate := range candidates {
		if IsRelevant(query, candidate, params) {
			relevant = append(relevant, candidate)
		}
	}

	return relevant
}

// ExplainSimilarity returns a breakdown of similarity components for debugging.
type SimilarityBreakdown struct {
	Total      float64
	Semantic   float64
	Lexical    float64
	Structural float64
	Params     KernelParams
}

// Explain computes similarity and returns a detailed breakdown.
func Explain(a, b *Features, params KernelParams) SimilarityBreakdown {
	params = params.Normalize()

	semantic := CosineSimilarity(a.TFIDF, b.TFIDF)
	lexical := JaccardSimilarity(a.Ngrams, b.Ngrams)
	structural := StructuralSimilarity(a, b)

	total := params.WeightSemantic*semantic +
		params.WeightLexical*lexical +
		params.WeightStructural*structural

	return SimilarityBreakdown{
		Total:      math.Max(0, math.Min(1, total)),
		Semantic:   semantic,
		Lexical:    lexical,
		Structural: structural,
		Params:     params,
	}
}

// String formats the breakdown for display.
func (sb SimilarityBreakdown) String() string {
	return fmt.Sprintf(
		"Total: %.3f | Semantic: %.3f (×%.2f) | Lexical: %.3f (×%.2f) | Structural: %.3f (×%.2f)",
		sb.Total,
		sb.Semantic, sb.Params.WeightSemantic,
		sb.Lexical, sb.Params.WeightLexical,
		sb.Structural, sb.Params.WeightStructural,
	)
}
