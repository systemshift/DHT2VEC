// Package core integrates cryptographic and semantic primitives for TERA.
//
// This is where the dual-output hash (H_crypto, H_semantic) comes together
// to enable integrity-gated semantic search.
package core

import (
	"encoding/json"
	"fmt"

	"github.com/systemshift/tera/crypto"
	"github.com/systemshift/tera/semantic"
)

// Content represents a piece of content with its dual hash.
type Content struct {
	// Raw data
	Data []byte

	// Dual hash outputs
	Crypto   *crypto.Hash
	Semantic *semantic.Features

	// Metadata
	ID string // Optional identifier
}

// NewContent creates a new content object with dual hash.
func NewContent(data []byte) *Content {
	return &Content{
		Data:     data,
		Crypto:   crypto.HashElement(data),
		Semantic: semantic.ExtractFeatures(data),
	}
}

// NewContentWithID creates content with an explicit ID.
func NewContentWithID(id string, data []byte) *Content {
	c := NewContent(data)
	c.ID = id
	return c
}

// Extend creates new content by extending this content with additional data.
// Returns new content with updated dual hash.
func (c *Content) Extend(newData []byte) *Content {
	// Combine data
	combinedData := append(c.Data, newData...)

	// Extend crypto hash (O(1) operation)
	newCrypto := crypto.Extend(c.Crypto, newData)

	// Extract features from combined content
	newSemantic := semantic.ExtractFeatures(combinedData)

	return &Content{
		Data:     combinedData,
		Crypto:   newCrypto,
		Semantic: newSemantic,
	}
}

// VerifyExtension checks if newContent correctly extends this content with newData.
func (c *Content) VerifyExtension(newContent *Content, newData []byte) bool {
	return crypto.VerifyExtension(c.Crypto, newContent.Crypto, newData)
}

// DualHash represents just the hash outputs (without the data).
// Useful for network transmission and storage.
type DualHash struct {
	Crypto   *crypto.Hash
	Semantic *semantic.Features
}

// GetDualHash extracts the dual hash from content.
func (c *Content) GetDualHash() DualHash {
	return DualHash{
		Crypto:   c.Crypto,
		Semantic: c.Semantic,
	}
}

// Similarity computes similarity between this content and another.
func (c *Content) Similarity(other *Content, params semantic.KernelParams) float64 {
	return semantic.Similarity(c.Semantic, other.Semantic, params)
}

// IsRelevant checks if this content is relevant to another given parameters.
func (c *Content) IsRelevant(other *Content, params semantic.KernelParams) bool {
	return semantic.IsRelevant(c.Semantic, other.Semantic, params)
}

// Extension represents an extension announcement in the network.
// Nodes gossip these to propagate new content.
type Extension struct {
	// The hash of the parent content
	ParentHash DualHash

	// The new data being added
	NewData []byte

	// The resulting hash after extension
	NewHash DualHash

	// Optional metadata
	Timestamp int64
	Publisher string
}

// NewExtension creates an extension from parent content and new data.
func NewExtension(parent *Content, newData []byte) *Extension {
	extended := parent.Extend(newData)

	return &Extension{
		ParentHash: parent.GetDualHash(),
		NewData:    newData,
		NewHash:    extended.GetDualHash(),
	}
}

// VerifyCrypto checks if the extension has a valid crypto proof.
func (ext *Extension) VerifyCrypto() bool {
	return crypto.VerifyExtension(
		ext.ParentHash.Crypto,
		ext.NewHash.Crypto,
		ext.NewData,
	)
}

// ComputeSemanticSimilarity computes similarity to a query.
func (ext *Extension) ComputeSemanticSimilarity(query *semantic.Features, params semantic.KernelParams) float64 {
	return semantic.Similarity(ext.NewHash.Semantic, query, params)
}

// IsRelevantTo checks if extension is relevant to a query.
func (ext *Extension) IsRelevantTo(query *semantic.Features, params semantic.KernelParams) bool {
	return semantic.IsRelevant(ext.NewHash.Semantic, query, params)
}

// Query represents a semantic search query with parameters.
type Query struct {
	// The content being searched for
	Content []byte

	// Extracted features
	Features *semantic.Features

	// Kernel parameters for this query
	Params semantic.KernelParams
}

// NewQuery creates a query from content and parameters.
func NewQuery(content []byte, params semantic.KernelParams) *Query {
	return &Query{
		Content:  content,
		Features: semantic.ExtractFeatures(content),
		Params:   params,
	}
}

// NewQueryDefault creates a query with default parameters.
func NewQueryDefault(content []byte) *Query {
	return NewQuery(content, semantic.DefaultParams())
}

// Matches checks if content matches this query.
func (q *Query) Matches(content *Content) bool {
	return content.IsRelevant(&Content{Semantic: q.Features}, q.Params)
}

// Rank ranks a list of content by similarity to this query.
func (q *Query) Rank(contents []*Content) []semantic.RankedResult {
	features := make([]*semantic.Features, len(contents))
	for i, c := range contents {
		features[i] = c.Semantic
	}
	return semantic.RankBySimilarity(q.Features, features, q.Params)
}

// Serialization support for network transmission

// MarshalJSON implements json.Marshaler for DualHash.
func (dh DualHash) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Crypto   string                `json:"crypto"`
		Semantic *semantic.Features    `json:"semantic"`
	}{
		Crypto:   dh.Crypto.Hex(),
		Semantic: dh.Semantic,
	})
}

// UnmarshalJSON implements json.Unmarshaler for DualHash.
func (dh *DualHash) UnmarshalJSON(data []byte) error {
	var aux struct {
		Crypto   string                `json:"crypto"`
		Semantic *semantic.Features    `json:"semantic"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	h, err := crypto.FromHex(aux.Crypto)
	if err != nil {
		return fmt.Errorf("invalid crypto hash: %w", err)
	}

	dh.Crypto = h
	dh.Semantic = aux.Semantic
	return nil
}
