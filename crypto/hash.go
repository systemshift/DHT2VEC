// Package crypto implements the homomorphic hash primitive for TERA.
//
// The core innovation is a hash function over an abelian group (integers mod prime)
// that supports O(1) incremental extensions:
//
//	H(A ∪ B) = H(A) + H(B) mod p
//
// This allows cryptographic verification of content extensions without recomputation.
package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Prime is the field modulus (secp256k1 curve order).
// Using a well-known prime from elliptic curve cryptography.
var Prime = mustParseBigInt("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

// Hash represents a homomorphic hash value (integer mod Prime).
type Hash struct {
	value *big.Int
}

// mustParseBigInt parses a hex string to big.Int, panics on error.
func mustParseBigInt(s string, base int) *big.Int {
	val, ok := new(big.Int).SetString(s, base)
	if !ok {
		panic(fmt.Sprintf("failed to parse: %s", s))
	}
	return val
}

// NewHash creates a Hash from a big.Int, reducing it modulo Prime.
func NewHash(value *big.Int) *Hash {
	h := &Hash{value: new(big.Int)}
	h.value.Mod(value, Prime)
	return h
}

// Zero returns the identity element (0).
func Zero() *Hash {
	return NewHash(big.NewInt(0))
}

// HashElement computes the hash of a single data element.
// Uses SHA3-256 as the base hash function, then reduces modulo Prime.
func HashElement(data []byte) *Hash {
	// Use SHA3-256 as the base hash
	digest := sha3.Sum256(data)

	// Convert digest to big.Int
	val := new(big.Int).SetBytes(digest[:])

	return NewHash(val)
}

// HashSet computes the homomorphic hash of multiple elements.
// H({e1, e2, ..., en}) = (H(e1) + H(e2) + ... + H(en)) mod Prime
//
// This is commutative: order of elements doesn't matter.
func HashSet(elements [][]byte) *Hash {
	sum := big.NewInt(0)

	for _, elem := range elements {
		h := HashElement(elem)
		sum.Add(sum, h.value)
	}

	return NewHash(sum)
}

// Extend computes H(A ∪ {new_element}) given H(A).
// This is O(1) - no need to rehash all of A.
//
//	H_new = H_old + H(new_element) mod Prime
func Extend(oldHash *Hash, newElement []byte) *Hash {
	newElemHash := HashElement(newElement)
	result := new(big.Int).Add(oldHash.value, newElemHash.value)
	return NewHash(result)
}

// Combine adds two hashes together.
// Represents the union of two disjoint sets: H(A ∪ B) = H(A) + H(B)
func Combine(h1, h2 *Hash) *Hash {
	result := new(big.Int).Add(h1.value, h2.value)
	return NewHash(result)
}

// VerifyExtension checks if newHash correctly extends oldHash with newElement.
// Returns true if: newHash == oldHash + H(newElement) mod Prime
func VerifyExtension(oldHash, newHash *Hash, newElement []byte) bool {
	expected := Extend(oldHash, newElement)
	return expected.Equal(newHash)
}

// Equal checks if two hashes are equal.
func (h *Hash) Equal(other *Hash) bool {
	return h.value.Cmp(other.value) == 0
}

// Bytes returns the hash as a 32-byte array (big-endian).
func (h *Hash) Bytes() []byte {
	bytes := h.value.Bytes()

	// Pad to 32 bytes if needed
	if len(bytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(bytes):], bytes)
		return padded
	}

	return bytes
}

// Hex returns the hash as a hexadecimal string.
func (h *Hash) Hex() string {
	return hex.EncodeToString(h.Bytes())
}

// String returns the hash as a hex string with "0x" prefix.
func (h *Hash) String() string {
	return "0x" + h.Hex()
}

// FromHex parses a hex string (with or without "0x" prefix) into a Hash.
func FromHex(hexStr string) (*Hash, error) {
	// Remove "0x" prefix if present
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	val, ok := new(big.Int).SetString(hexStr, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex string: %s", hexStr)
	}

	return NewHash(val), nil
}

// FromBytes creates a Hash from a byte slice.
func FromBytes(data []byte) *Hash {
	val := new(big.Int).SetBytes(data)
	return NewHash(val)
}

// Value returns the underlying big.Int value (read-only).
func (h *Hash) Value() *big.Int {
	return new(big.Int).Set(h.value)
}

// checksumSHA256 computes a SHA-256 checksum for error detection.
// This is NOT part of the homomorphic hash - just for integrity checks.
func checksumSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
