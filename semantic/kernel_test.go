package semantic

import (
	"math"
	"testing"
)

const epsilon = 1e-9 // For float comparison

// TestTokenize verifies text tokenization works correctly.
func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"Hello, World!", []string{"hello", "world"}},
		{"one  two   three", []string{"one", "two", "three"}},
		{"test-case", []string{"test", "case"}},
		{"", []string{}},
		{"123 abc", []string{"123", "abc"}},
	}

	for _, tt := range tests {
		result := Tokenize(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("Tokenize(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("Tokenize(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

// TestGenerateNgrams verifies n-gram generation.
func TestGenerateNgrams(t *testing.T) {
	ngrams := GenerateNgrams("hello", 3)

	expected := []string{"hel", "ell", "llo"}
	if len(ngrams) != len(expected) {
		t.Errorf("Expected %d ngrams, got %d", len(expected), len(ngrams))
	}

	for _, ng := range expected {
		if !ngrams[ng] {
			t.Errorf("Missing expected ngram: %q", ng)
		}
	}
}

// TestComputeTF verifies term frequency calculation.
func TestComputeTF(t *testing.T) {
	words := []string{"hello", "world", "hello"}
	tf := ComputeTF(words)

	// "hello" appears 2/3 times
	if math.Abs(tf["hello"]-2.0/3.0) > epsilon {
		t.Errorf("TF for 'hello' = %f, want %f", tf["hello"], 2.0/3.0)
	}

	// "world" appears 1/3 times
	if math.Abs(tf["world"]-1.0/3.0) > epsilon {
		t.Errorf("TF for 'world' = %f, want %f", tf["world"], 1.0/3.0)
	}
}

// TestCosineSimilarity verifies cosine similarity calculation.
func TestCosineSimilarity(t *testing.T) {
	// Identical vectors should have similarity 1
	a := map[string]float64{"a": 1, "b": 1}
	b := map[string]float64{"a": 1, "b": 1}
	sim := CosineSimilarity(a, b)

	if math.Abs(sim-1.0) > epsilon {
		t.Errorf("Identical vectors: similarity = %f, want 1.0", sim)
	}

	// Orthogonal vectors should have similarity 0
	c := map[string]float64{"a": 1}
	d := map[string]float64{"b": 1}
	sim = CosineSimilarity(c, d)

	if math.Abs(sim) > epsilon {
		t.Errorf("Orthogonal vectors: similarity = %f, want 0.0", sim)
	}
}

// TestJaccardSimilarity verifies Jaccard similarity calculation.
func TestJaccardSimilarity(t *testing.T) {
	// Identical sets should have similarity 1
	a := map[string]bool{"a": true, "b": true}
	b := map[string]bool{"a": true, "b": true}
	sim := JaccardSimilarity(a, b)

	if math.Abs(sim-1.0) > epsilon {
		t.Errorf("Identical sets: similarity = %f, want 1.0", sim)
	}

	// Disjoint sets should have similarity 0
	c := map[string]bool{"a": true}
	d := map[string]bool{"b": true}
	sim = JaccardSimilarity(c, d)

	if math.Abs(sim) > epsilon {
		t.Errorf("Disjoint sets: similarity = %f, want 0.0", sim)
	}

	// 50% overlap: {a,b} vs {b,c} = 1/3
	e := map[string]bool{"a": true, "b": true}
	f := map[string]bool{"b": true, "c": true}
	sim = JaccardSimilarity(e, f)

	expected := 1.0 / 3.0
	if math.Abs(sim-expected) > epsilon {
		t.Errorf("Partial overlap: similarity = %f, want %f", sim, expected)
	}
}

// TestExtractFeatures verifies feature extraction works.
func TestExtractFeatures(t *testing.T) {
	content := []byte("Hello world! Hello everyone.")
	features := ExtractFeatures(content)

	if features.WordCount != 4 {
		t.Errorf("WordCount = %d, want 4", features.WordCount)
	}

	if features.UniqueWords != 3 {
		t.Errorf("UniqueWords = %d, want 3 (hello, world, everyone)", features.UniqueWords)
	}

	if len(features.Ngrams) == 0 {
		t.Errorf("Expected non-empty ngrams")
	}

	// "hello" should have highest TF
	if features.TFIDF["hello"] < features.TFIDF["world"] {
		t.Errorf("Expected 'hello' to have higher TF than 'world'")
	}
}

// TestSimilaritySameContent verifies identical content has similarity 1.
func TestSimilaritySameContent(t *testing.T) {
	content := []byte("This is a test document.")
	a := ExtractFeatures(content)
	b := ExtractFeatures(content)

	params := DefaultParams()
	sim := Similarity(a, b, params)

	if math.Abs(sim-1.0) > epsilon {
		t.Errorf("Identical content: similarity = %f, want 1.0", sim)
	}
}

// TestSimilarityDifferentContent verifies different content has similarity < 1.
func TestSimilarityDifferentContent(t *testing.T) {
	a := ExtractFeatures([]byte("machine learning algorithms"))
	b := ExtractFeatures([]byte("cooking recipes"))

	params := DefaultParams()
	sim := Similarity(a, b, params)

	if sim >= 0.5 {
		t.Errorf("Unrelated content: similarity = %f, want < 0.5", sim)
	}
}

// TestSimilarityRelatedContent verifies related content has higher similarity.
func TestSimilarityRelatedContent(t *testing.T) {
	a := ExtractFeatures([]byte("machine learning and artificial intelligence"))
	b := ExtractFeatures([]byte("deep learning and neural networks"))

	params := DefaultParams()
	sim := Similarity(a, b, params)

	// Should be somewhat similar (share "learning" and "and")
	if sim < 0.1 || sim > 0.9 {
		t.Errorf("Related content: similarity = %f, expected in [0.1, 0.9]", sim)
	}
}

// TestParameterizedSimilarity verifies different parameters give different results.
func TestParameterizedSimilarity(t *testing.T) {
	a := ExtractFeatures([]byte("artificial intelligence machine learning"))
	b := ExtractFeatures([]byte("artificial intelligence deep learning"))

	// Semantic-focused parameters
	semanticParams := SemanticFocusedParams()
	simSemantic := Similarity(a, b, semanticParams)

	// Lexical-focused parameters
	lexicalParams := LexicalFocusedParams()
	simLexical := Similarity(a, b, lexicalParams)

	// They should give different results
	if math.Abs(simSemantic-simLexical) < 0.01 {
		t.Errorf("Different parameters should produce different similarities")
	}
}

// TestIsRelevant verifies relevance threshold checking.
func TestIsRelevant(t *testing.T) {
	a := ExtractFeatures([]byte("machine learning"))
	b := ExtractFeatures([]byte("machine learning"))
	c := ExtractFeatures([]byte("cooking recipes"))

	params := DefaultParams()
	params.Threshold = 0.7

	if !IsRelevant(a, b, params) {
		t.Errorf("Identical content should be relevant")
	}

	if IsRelevant(a, c, params) {
		t.Errorf("Unrelated content should not be relevant")
	}
}

// TestRankBySimilarity verifies ranking works correctly.
func TestRankBySimilarity(t *testing.T) {
	query := ExtractFeatures([]byte("machine learning algorithms"))

	candidates := []*Features{
		ExtractFeatures([]byte("cooking recipes")),              // Least similar
		ExtractFeatures([]byte("machine learning basics")),      // Most similar
		ExtractFeatures([]byte("data science and statistics")),  // Medium similar
	}

	params := DefaultParams()
	ranked := RankBySimilarity(query, candidates, params)

	// Should have 3 results
	if len(ranked) != 3 {
		t.Fatalf("Expected 3 ranked results, got %d", len(ranked))
	}

	// First result should be the most similar
	if ranked[0].Index != 1 {
		t.Errorf("Expected candidate 1 (machine learning) to rank first, got index %d", ranked[0].Index)
	}

	// Results should be in descending order
	for i := 1; i < len(ranked); i++ {
		if ranked[i].Similarity > ranked[i-1].Similarity {
			t.Errorf("Results not sorted: ranked[%d].Similarity (%.3f) > ranked[%d].Similarity (%.3f)",
				i, ranked[i].Similarity, i-1, ranked[i-1].Similarity)
		}
	}
}

// TestFilterRelevant verifies filtering by threshold.
func TestFilterRelevant(t *testing.T) {
	query := ExtractFeatures([]byte("machine learning"))

	candidates := []*Features{
		ExtractFeatures([]byte("machine learning algorithms")),  // Similar
		ExtractFeatures([]byte("cooking recipes")),              // Not similar
		ExtractFeatures([]byte("machine learning basics")),      // Similar
	}

	params := DefaultParams()
	params.Threshold = 0.3

	relevant := FilterRelevant(query, candidates, params)

	// Should filter out the unrelated one
	if len(relevant) < 2 {
		t.Errorf("Expected at least 2 relevant results, got %d", len(relevant))
	}
}

// TestExplain verifies similarity breakdown works.
func TestExplain(t *testing.T) {
	a := ExtractFeatures([]byte("machine learning"))
	b := ExtractFeatures([]byte("machine learning algorithms"))

	params := DefaultParams()
	breakdown := Explain(a, b, params)

	// Total should be in [0, 1]
	if breakdown.Total < 0 || breakdown.Total > 1 {
		t.Errorf("Total similarity out of range: %f", breakdown.Total)
	}

	// Components should be in [0, 1]
	if breakdown.Semantic < 0 || breakdown.Semantic > 1 {
		t.Errorf("Semantic similarity out of range: %f", breakdown.Semantic)
	}
	if breakdown.Lexical < 0 || breakdown.Lexical > 1 {
		t.Errorf("Lexical similarity out of range: %f", breakdown.Lexical)
	}
	if breakdown.Structural < 0 || breakdown.Structural > 1 {
		t.Errorf("Structural similarity out of range: %f", breakdown.Structural)
	}

	// String should be non-empty
	if len(breakdown.String()) == 0 {
		t.Errorf("Expected non-empty string representation")
	}
}

// TestParamsValidation verifies parameter validation.
func TestParamsValidation(t *testing.T) {
	// Valid params
	valid := DefaultParams()
	if err := valid.Validate(); err != nil {
		t.Errorf("Valid params failed validation: %v", err)
	}

	// Negative weight (invalid)
	invalid := KernelParams{
		WeightSemantic: -0.5,
		WeightLexical:  0.5,
	}
	if err := invalid.Validate(); err == nil {
		t.Errorf("Negative weight should fail validation")
	}

	// All zero weights (invalid)
	allZero := KernelParams{}
	if err := allZero.Validate(); err == nil {
		t.Errorf("All-zero weights should fail validation")
	}

	// Invalid threshold
	invalidThreshold := DefaultParams()
	invalidThreshold.Threshold = 1.5
	if err := invalidThreshold.Validate(); err == nil {
		t.Errorf("Threshold > 1 should fail validation")
	}
}

// TestParamsNormalization verifies weight normalization.
func TestParamsNormalization(t *testing.T) {
	params := KernelParams{
		WeightSemantic:   2.0,
		WeightLexical:    2.0,
		WeightStructural: 2.0,
	}

	normalized := params.Normalize()

	// Each should be 1/3 after normalization
	expected := 1.0 / 3.0
	if math.Abs(normalized.WeightSemantic-expected) > epsilon {
		t.Errorf("WeightSemantic after normalization = %f, want %f", normalized.WeightSemantic, expected)
	}
	if math.Abs(normalized.WeightLexical-expected) > epsilon {
		t.Errorf("WeightLexical after normalization = %f, want %f", normalized.WeightLexical, expected)
	}
	if math.Abs(normalized.WeightStructural-expected) > epsilon {
		t.Errorf("WeightStructural after normalization = %f, want %f", normalized.WeightStructural, expected)
	}
}

// TestEmptyContent verifies handling of empty input.
func TestEmptyContent(t *testing.T) {
	features := ExtractFeatures([]byte(""))

	if features.WordCount != 0 {
		t.Errorf("Empty content: WordCount = %d, want 0", features.WordCount)
	}

	if features.UniqueWords != 0 {
		t.Errorf("Empty content: UniqueWords = %d, want 0", features.UniqueWords)
	}
}

// BenchmarkExtractFeatures measures feature extraction performance.
func BenchmarkExtractFeatures(b *testing.B) {
	content := []byte("Machine learning is a field of artificial intelligence that uses statistical techniques to give computer systems the ability to learn from data.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractFeatures(content)
	}
}

// BenchmarkSimilarity measures similarity computation performance.
func BenchmarkSimilarity(b *testing.B) {
	a := ExtractFeatures([]byte("machine learning algorithms"))
	c := ExtractFeatures([]byte("artificial intelligence systems"))
	params := DefaultParams()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Similarity(a, c, params)
	}
}

// BenchmarkRankBySimilarity measures ranking performance.
func BenchmarkRankBySimilarity(b *testing.B) {
	query := ExtractFeatures([]byte("machine learning"))

	candidates := make([]*Features, 100)
	for i := 0; i < 100; i++ {
		candidates[i] = ExtractFeatures([]byte("test document with various content"))
	}

	params := DefaultParams()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RankBySimilarity(query, candidates, params)
	}
}
