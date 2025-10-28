package crypto

import (
	"math/big"
	"testing"
)

// TestHashElementDeterministic verifies that hashing the same data produces the same result.
func TestHashElementDeterministic(t *testing.T) {
	data := []byte("hello world")

	h1 := HashElement(data)
	h2 := HashElement(data)

	if !h1.Equal(h2) {
		t.Errorf("HashElement not deterministic: %s != %s", h1.Hex(), h2.Hex())
	}
}

// TestHashElementDifferent verifies that different data produces different hashes.
func TestHashElementDifferent(t *testing.T) {
	h1 := HashElement([]byte("hello"))
	h2 := HashElement([]byte("world"))

	if h1.Equal(h2) {
		t.Errorf("Different data produced same hash")
	}
}

// TestHashSetCommutative verifies that order doesn't matter.
// H({a, b, c}) should equal H({c, a, b})
func TestHashSetCommutative(t *testing.T) {
	a := []byte("alpha")
	b := []byte("beta")
	c := []byte("gamma")

	h1 := HashSet([][]byte{a, b, c})
	h2 := HashSet([][]byte{c, a, b})
	h3 := HashSet([][]byte{b, c, a})

	if !h1.Equal(h2) {
		t.Errorf("HashSet not commutative: {a,b,c} != {c,a,b}")
	}

	if !h1.Equal(h3) {
		t.Errorf("HashSet not commutative: {a,b,c} != {b,c,a}")
	}
}

// TestHomomorphicProperty verifies the core property:
// H(A ∪ B) = H(A) + H(B) mod Prime
func TestHomomorphicProperty(t *testing.T) {
	setA := [][]byte{[]byte("doc1"), []byte("doc2")}
	setB := [][]byte{[]byte("doc3"), []byte("doc4")}
	setAB := [][]byte{[]byte("doc1"), []byte("doc2"), []byte("doc3"), []byte("doc4")}

	hA := HashSet(setA)
	hB := HashSet(setB)
	hAB := HashSet(setAB)

	// H(A ∪ B) should equal H(A) + H(B)
	hCombined := Combine(hA, hB)

	if !hAB.Equal(hCombined) {
		t.Errorf("Homomorphic property violated:\n  H(A∪B) = %s\n  H(A)+H(B) = %s",
			hAB.Hex(), hCombined.Hex())
	}
}

// TestExtend verifies that Extend correctly computes H(A ∪ {new}).
func TestExtend(t *testing.T) {
	// Start with a set {a, b}
	initial := [][]byte{[]byte("a"), []byte("b")}
	hInitial := HashSet(initial)

	// Extend with "c"
	newElem := []byte("c")
	hExtended := Extend(hInitial, newElem)

	// Should equal H({a, b, c})
	full := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	hFull := HashSet(full)

	if !hExtended.Equal(hFull) {
		t.Errorf("Extend incorrect:\n  Extend(H(a,b), c) = %s\n  H(a,b,c) = %s",
			hExtended.Hex(), hFull.Hex())
	}
}

// TestVerifyExtension verifies that valid extensions pass and invalid ones fail.
func TestVerifyExtension(t *testing.T) {
	oldSet := [][]byte{[]byte("doc1"), []byte("doc2")}
	hOld := HashSet(oldSet)

	newElem := []byte("doc3")
	hNew := Extend(hOld, newElem)

	// Valid extension should verify
	if !VerifyExtension(hOld, hNew, newElem) {
		t.Errorf("Valid extension failed verification")
	}

	// Invalid extension (wrong new element) should fail
	wrongElem := []byte("doc4")
	if VerifyExtension(hOld, hNew, wrongElem) {
		t.Errorf("Invalid extension passed verification")
	}

	// Invalid extension (tampered hash) should fail
	tamperedHash := HashElement([]byte("malicious"))
	if VerifyExtension(hOld, tamperedHash, newElem) {
		t.Errorf("Tampered hash passed verification")
	}
}

// TestZeroHash verifies that Zero() returns the identity element.
func TestZeroHash(t *testing.T) {
	zero := Zero()
	h := HashElement([]byte("test"))

	// H + 0 = H
	hPlusZero := Combine(h, zero)
	if !h.Equal(hPlusZero) {
		t.Errorf("Zero not identity: H + 0 != H")
	}

	// 0 + H = H (commutativity)
	zeroPlusH := Combine(zero, h)
	if !h.Equal(zeroPlusH) {
		t.Errorf("Zero not identity: 0 + H != H")
	}
}

// TestEmptySet verifies that hashing an empty set gives zero.
func TestEmptySet(t *testing.T) {
	empty := HashSet([][]byte{})
	zero := Zero()

	if !empty.Equal(zero) {
		t.Errorf("Empty set should hash to zero:\n  got %s\n  want %s",
			empty.Hex(), zero.Hex())
	}
}

// TestHexRoundtrip verifies that Hex() and FromHex() are inverses.
func TestHexRoundtrip(t *testing.T) {
	original := HashElement([]byte("test data"))
	hexStr := original.Hex()

	recovered, err := FromHex(hexStr)
	if err != nil {
		t.Fatalf("FromHex failed: %v", err)
	}

	if !original.Equal(recovered) {
		t.Errorf("Hex roundtrip failed:\n  original:  %s\n  recovered: %s",
			original.Hex(), recovered.Hex())
	}
}

// TestHexWithPrefix verifies that FromHex handles "0x" prefix.
func TestHexWithPrefix(t *testing.T) {
	original := HashElement([]byte("test"))

	withPrefix := "0x" + original.Hex()
	recovered, err := FromHex(withPrefix)
	if err != nil {
		t.Fatalf("FromHex with prefix failed: %v", err)
	}

	if !original.Equal(recovered) {
		t.Errorf("Hex with prefix failed")
	}
}

// TestBytesRoundtrip verifies that Bytes() and FromBytes() are inverses.
func TestBytesRoundtrip(t *testing.T) {
	original := HashElement([]byte("test data"))
	bytes := original.Bytes()

	recovered := FromBytes(bytes)

	if !original.Equal(recovered) {
		t.Errorf("Bytes roundtrip failed")
	}
}

// TestBytesPadding verifies that Bytes() pads to 32 bytes.
func TestBytesPadding(t *testing.T) {
	// Small hash value (will need padding)
	small := NewHash(big.NewInt(42))
	b := small.Bytes()

	if len(b) != 32 {
		t.Errorf("Bytes() should pad to 32 bytes, got %d", len(b))
	}

	// Last byte should be 42
	if b[31] != 42 {
		t.Errorf("Expected last byte to be 42, got %d", b[31])
	}

	// Most leading bytes should be zero (due to padding)
	nonZeroCount := 0
	for _, byte := range b[:31] {
		if byte != 0 {
			nonZeroCount++
		}
	}

	// For value 42, most bytes should be zero
	if nonZeroCount > 5 {
		t.Errorf("Expected mostly zero padding, got %d non-zero bytes", nonZeroCount)
	}
}

// TestMultipleExtensions verifies chaining multiple extensions.
func TestMultipleExtensions(t *testing.T) {
	// Build up incrementally
	h := Zero()
	h = Extend(h, []byte("doc1"))
	h = Extend(h, []byte("doc2"))
	h = Extend(h, []byte("doc3"))

	// Compare to direct hash
	full := HashSet([][]byte{
		[]byte("doc1"),
		[]byte("doc2"),
		[]byte("doc3"),
	})

	if !h.Equal(full) {
		t.Errorf("Multiple extensions don't match full hash")
	}
}

// TestLargeDataset verifies the hash works with larger sets.
func TestLargeDataset(t *testing.T) {
	elements := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		elements[i] = []byte{byte(i)}
	}

	h := HashSet(elements)

	// Verify it's non-zero
	if h.Equal(Zero()) {
		t.Errorf("Large dataset hashed to zero")
	}

	// Verify incremental construction matches
	hIncremental := Zero()
	for _, elem := range elements {
		hIncremental = Extend(hIncremental, elem)
	}

	if !h.Equal(hIncremental) {
		t.Errorf("Incremental construction doesn't match batch hash")
	}
}

// TestHashValue verifies that Value() returns a copy, not the original.
func TestHashValue(t *testing.T) {
	h := HashElement([]byte("test"))
	val := h.Value()

	// Modify the returned value
	val.Add(val, big.NewInt(1))

	// Original should be unchanged
	val2 := h.Value()
	if val.Cmp(val2) == 0 {
		t.Errorf("Value() should return a copy, not original")
	}
}

// BenchmarkHashElement measures the performance of hashing a single element.
func BenchmarkHashElement(b *testing.B) {
	data := []byte("benchmark data for hash element performance testing")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashElement(data)
	}
}

// BenchmarkExtend measures the performance of extending a hash.
func BenchmarkExtend(b *testing.B) {
	h := HashElement([]byte("initial"))
	newElem := []byte("extension")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Extend(h, newElem)
	}
}

// BenchmarkVerifyExtension measures verification performance.
func BenchmarkVerifyExtension(b *testing.B) {
	hOld := HashElement([]byte("old"))
	newElem := []byte("new")
	hNew := Extend(hOld, newElem)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyExtension(hOld, hNew, newElem)
	}
}
