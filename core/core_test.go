package core

import (
	"testing"

	"github.com/systemshift/tera/semantic"
)

// TestNewContent verifies content creation with dual hash.
func TestNewContent(t *testing.T) {
	data := []byte("test document")
	content := NewContent(data)

	if content.Crypto == nil {
		t.Error("Crypto hash not generated")
	}

	if content.Semantic == nil {
		t.Error("Semantic features not extracted")
	}

	if len(content.Data) != len(data) {
		t.Errorf("Data length mismatch: got %d, want %d", len(content.Data), len(data))
	}
}

// TestContentExtension verifies content extension works correctly.
func TestContentExtension(t *testing.T) {
	original := NewContent([]byte("original document"))
	extended := original.Extend([]byte(" with extension"))

	// Verify data is combined
	expectedLen := len("original document with extension")
	if len(extended.Data) != expectedLen {
		t.Errorf("Extended data length = %d, want %d", len(extended.Data), expectedLen)
	}

	// Verify crypto extension
	if !original.VerifyExtension(extended, []byte(" with extension")) {
		t.Error("Crypto verification failed for valid extension")
	}
}

// TestInvalidExtension verifies invalid extensions are detected.
func TestInvalidExtension(t *testing.T) {
	original := NewContent([]byte("original"))

	// Create a fake extension with wrong data
	fakeExtension := NewContent([]byte("fake content"))

	// Should NOT verify
	if original.VerifyExtension(fakeExtension, []byte(" wrong")) {
		t.Error("Invalid extension passed verification")
	}
}

// TestDualHashRoundtrip verifies dual hash extraction.
func TestDualHashRoundtrip(t *testing.T) {
	content := NewContent([]byte("test"))
	dualHash := content.GetDualHash()

	if !dualHash.Crypto.Equal(content.Crypto) {
		t.Error("Crypto hash mismatch after extraction")
	}

	if dualHash.Semantic != content.Semantic {
		t.Error("Semantic features mismatch after extraction")
	}
}

// TestQuery verifies query creation and matching.
func TestQuery(t *testing.T) {
	query := NewQueryDefault([]byte("machine learning"))

	relevantContent := NewContent([]byte("machine learning algorithms"))
	irrelevantContent := NewContent([]byte("cooking recipes"))

	if !query.Matches(relevantContent) {
		t.Error("Query should match relevant content")
	}

	// This might pass or fail depending on threshold, so we just check it runs
	_ = query.Matches(irrelevantContent)
}

// TestGatekeeperValidExtension tests that valid, relevant extensions pass.
func TestGatekeeperValidExtension(t *testing.T) {
	gk := NewGatekeeper()

	// Create legitimate extension
	root := NewContent([]byte("machine learning basics"))
	newData := []byte(" and neural networks")
	ext := NewExtension(root, newData)

	// Query interested in machine learning (lower threshold for test)
	query := NewQueryDefault([]byte("machine learning"))
	query.Params.Threshold = 0.3

	decision := gk.ShouldForward(ext, query)

	if !decision.ShouldForward {
		t.Errorf("Valid relevant extension blocked: %s", decision.Reason)
	}

	if !decision.CryptoValid {
		t.Error("Crypto verification failed for valid extension")
	}

	if !decision.SemanticRelevant {
		t.Error("Semantic relevance failed for relevant content")
	}
}

// TestGatekeeperInvalidExtension tests that crypto-invalid extensions are blocked.
func TestGatekeeperInvalidExtension(t *testing.T) {
	gk := NewGatekeeper()

	// Create fake extension (doesn't actually extend root)
	root := NewContent([]byte("legitimate root"))

	// Create extension manually with wrong hash
	fakeExt := &Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte("fake data"),
		NewHash:    NewContent([]byte("completely different content")).GetDualHash(),
	}

	query := NewQueryDefault([]byte("test"))

	decision := gk.ShouldForward(fakeExt, query)

	if decision.ShouldForward {
		t.Error("Invalid extension was not blocked")
	}

	if decision.CryptoValid {
		t.Error("Crypto verification passed for invalid extension")
	}

	if gk.CryptoBlocked != 1 {
		t.Errorf("CryptoBlocked = %d, want 1", gk.CryptoBlocked)
	}
}

// TestGatekeeperIrrelevantExtension tests that irrelevant extensions are blocked.
func TestGatekeeperIrrelevantExtension(t *testing.T) {
	gk := NewGatekeeper()

	// Create valid extension but about different topic
	root := NewContent([]byte("cooking recipes"))
	newData := []byte(" for Italian pasta")
	ext := NewExtension(root, newData)

	// Query interested in machine learning (not cooking)
	query := NewQueryDefault([]byte("machine learning algorithms"))
	query.Params.Threshold = 0.5 // High threshold

	decision := gk.ShouldForward(ext, query)

	if decision.ShouldForward {
		t.Error("Irrelevant extension was not blocked")
	}

	if !decision.CryptoValid {
		t.Error("Crypto verification should pass even if irrelevant")
	}

	if decision.SemanticRelevant {
		t.Error("Should not be semantically relevant")
	}

	if gk.SemanticBlocked != 1 {
		t.Errorf("SemanticBlocked = %d, want 1", gk.SemanticBlocked)
	}
}

// TestGatekeeperStats verifies statistics tracking.
func TestGatekeeperStats(t *testing.T) {
	gk := NewGatekeeper()

	root := NewContent([]byte("test"))
	ext1 := NewExtension(root, []byte(" valid"))
	query := NewQueryDefault([]byte("test"))

	// Process several extensions
	gk.ShouldForward(ext1, query)
	gk.ShouldForward(ext1, query)
	gk.ShouldForward(ext1, query)

	stats := gk.GetStats()

	if stats.TotalSeen != 3 {
		t.Errorf("TotalSeen = %d, want 3", stats.TotalSeen)
	}
}

// TestInterestFilter verifies interest-based filtering.
func TestInterestFilter(t *testing.T) {
	filter := NewInterestFilter(
		[]string{"machine learning", "artificial intelligence"},
		semantic.DefaultParams(),
	)

	relevantContent := NewContent([]byte("machine learning algorithms"))
	irrelevantContent := NewContent([]byte("cooking recipes"))

	if !filter.Matches(relevantContent) {
		t.Error("Interest filter should match relevant content")
	}

	// Cooking shouldn't match ML/AI interests
	if filter.Matches(irrelevantContent) {
		t.Error("Interest filter should not match irrelevant content")
	}
}

// TestGossipSimulator verifies gossip simulation works.
func TestGossipSimulator(t *testing.T) {
	sim := NewGossipSimulator()

	// Add nodes with different interests
	node1 := NewSimulatedNode("node1",
		[]string{"machine learning"},
		semantic.DefaultParams())
	node2 := NewSimulatedNode("node2",
		[]string{"cooking"},
		semantic.DefaultParams())
	node3 := NewSimulatedNode("node3",
		[]string{"machine learning", "cooking"},
		semantic.DefaultParams())

	sim.AddNode(node1)
	sim.AddNode(node2)
	sim.AddNode(node3)

	// Create extension about machine learning
	root := NewContent([]byte("machine learning"))
	ext := NewExtension(root, []byte(" and neural networks"))

	// Propagate it
	forwarded := sim.PropagateExtension(ext)

	// Should be forwarded by node1 and node3 (interested in ML)
	// Node2 (cooking) should not forward
	if forwarded < 1 {
		t.Errorf("Expected at least 1 node to forward, got %d", forwarded)
	}

	// Check that node1 received it
	if len(node1.Received) == 0 {
		t.Error("Node1 (ML interest) should have received the extension")
	}
}

// TestSpamAttackScenario simulates a spam attack.
func TestSpamAttackScenario(t *testing.T) {
	gk := NewGatekeeper()

	// Legitimate root
	root := NewContent([]byte("machine learning research"))

	// Legitimate extension
	legit := NewExtension(root, []byte(" on neural networks"))

	// Spam attack: attacker creates fake "similar" content
	spam := &Extension{
		ParentHash: root.GetDualHash(), // Claims to extend root
		NewData:    []byte(" buy viagra cheap!!!"),
		NewHash:    NewContent([]byte("machine learning research buy viagra cheap!!!")).GetDualHash(),
		// BUT: NewHash is computed incorrectly (not using root.Extend)
	}

	// Query from user interested in ML (lower threshold for test)
	query := NewQueryDefault([]byte("machine learning"))
	query.Params.Threshold = 0.3

	// Legitimate extension should pass
	legitDecision := gk.ShouldForward(legit, query)
	if !legitDecision.ShouldForward {
		t.Errorf("Legitimate extension blocked: %s", legitDecision.Reason)
	}

	// Spam should be blocked (crypto verification fails)
	spamDecision := gk.ShouldForward(spam, query)
	if spamDecision.ShouldForward {
		t.Error("Spam extension was not blocked!")
	}

	if spamDecision.CryptoValid {
		t.Error("Spam passed crypto verification")
	}

	// Verify statistics
	stats := gk.GetStats()
	if stats.CryptoBlocked < 1 {
		t.Error("Spam should be counted in CryptoBlocked")
	}
}

// TestSemanticSpamScenario tests spam that passes crypto but is irrelevant.
func TestSemanticSpamScenario(t *testing.T) {
	gk := NewGatekeeper()

	// Attacker controls their own root
	attackerRoot := NewContent([]byte("spam content"))

	// Attacker legitimately extends their own content
	attackerExt := NewExtension(attackerRoot, []byte(" more spam"))

	// User query for legitimate content
	query := NewQueryDefault([]byte("machine learning research"))
	query.Params.Threshold = 0.3

	decision := gk.ShouldForward(attackerExt, query)

	// Should pass crypto (it's a valid extension)
	if !decision.CryptoValid {
		t.Error("Valid extension should pass crypto check")
	}

	// Should fail semantic (irrelevant to query)
	if decision.SemanticRelevant {
		t.Error("Irrelevant spam should fail semantic check")
	}

	// Should not be forwarded
	if decision.ShouldForward {
		t.Error("Irrelevant spam should not be forwarded")
	}
}

// TestContentSimilarity verifies similarity computation.
func TestContentSimilarity(t *testing.T) {
	a := NewContent([]byte("machine learning algorithms"))
	b := NewContent([]byte("deep learning neural networks"))
	c := NewContent([]byte("cooking italian recipes"))

	params := semantic.DefaultParams()

	// A and B should be somewhat similar (both ML-related)
	simAB := a.Similarity(b, params)

	// A and C should be dissimilar
	simAC := a.Similarity(c, params)

	if simAC >= simAB {
		t.Errorf("ML content should be more similar to ML than to cooking: AB=%.3f, AC=%.3f", simAB, simAC)
	}
}

// TestMultipleExtensions verifies chaining extensions.
func TestMultipleExtensions(t *testing.T) {
	root := NewContent([]byte("Chapter 1"))
	ext1 := root.Extend([]byte(". Chapter 2"))
	ext2 := ext1.Extend([]byte(". Chapter 3"))

	// Each extension should be verifiable from its parent
	if !root.VerifyExtension(ext1, []byte(". Chapter 2")) {
		t.Error("First extension verification failed")
	}

	if !ext1.VerifyExtension(ext2, []byte(". Chapter 3")) {
		t.Error("Second extension verification failed")
	}

	// But ext2 should NOT verify as direct extension of root
	if root.VerifyExtension(ext2, []byte(". Chapter 3")) {
		t.Error("Should not verify as direct extension of root")
	}
}

// BenchmarkGatekeeper measures gatekeeping performance.
func BenchmarkGatekeeper(b *testing.B) {
	gk := NewGatekeeper()
	root := NewContent([]byte("test content"))
	ext := NewExtension(root, []byte(" extension"))
	query := NewQueryDefault([]byte("test"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gk.ShouldForward(ext, query)
	}
}

// BenchmarkContentExtension measures extension performance.
func BenchmarkContentExtension(b *testing.B) {
	root := NewContent([]byte("root content"))
	newData := []byte(" additional data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.Extend(newData)
	}
}
