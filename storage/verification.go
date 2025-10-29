package storage

import (
	"fmt"

	"github.com/systemshift/tera/crypto"
)

// VerificationResult contains the result of chain verification.
type VerificationResult struct {
	Valid  bool
	Reason string

	// Chain details
	ChainLength int
	RootHash    *crypto.Hash
	FinalHash   *crypto.Hash
}

// VerifyChain verifies that a hash correctly extends from a root.
//
// This is the key innovation: we can cryptographically prove that
// content has been properly extended without recomputing everything.
func (eg *ExtensionGraph) VerifyChain(root, target *crypto.Hash) (*VerificationResult, error) {
	// Get the full chain
	chain, err := eg.GetChain(target)
	if err != nil {
		return nil, fmt.Errorf("get chain: %w", err)
	}

	// Start with root
	currentHash := root

	// Verify each step in the chain
	for i, record := range chain {
		// Verify parent matches
		if !record.Parent.Equal(currentHash) {
			return &VerificationResult{
				Valid:  false,
				Reason: fmt.Sprintf("chain break at step %d: parent mismatch", i),
			}, nil
		}

		// Verify extension: Child should equal Parent + Delta
		expectedChild := crypto.Extend(record.Parent, record.Delta)
		if !expectedChild.Equal(record.Child) {
			return &VerificationResult{
				Valid:  false,
				Reason: fmt.Sprintf("invalid extension at step %d: hash mismatch", i),
			}, nil
		}

		currentHash = record.Child
	}

	// Verify we reached the target
	if !currentHash.Equal(target) {
		return &VerificationResult{
			Valid:  false,
			Reason: "chain does not reach target hash",
		}, nil
	}

	return &VerificationResult{
		Valid:       true,
		Reason:      "chain verified successfully",
		ChainLength: len(chain),
		RootHash:    root,
		FinalHash:   target,
	}, nil
}

// VerifyExtension verifies a single extension step.
func (eg *ExtensionGraph) VerifyExtension(parent, child *crypto.Hash) (*VerificationResult, error) {
	// Get extension record
	record, err := eg.GetExtension(child)
	if err != nil {
		return &VerificationResult{
			Valid:  false,
			Reason: fmt.Sprintf("extension record not found: %v", err),
		}, nil
	}

	// Verify parent matches
	if !record.Parent.Equal(parent) {
		return &VerificationResult{
			Valid:  false,
			Reason: "parent hash mismatch",
		}, nil
	}

	// Verify crypto hash
	expectedChild := crypto.Extend(parent, record.Delta)
	if !expectedChild.Equal(child) {
		return &VerificationResult{
			Valid:  false,
			Reason: "child hash verification failed",
		}, nil
	}

	return &VerificationResult{
		Valid:       true,
		Reason:      "extension verified",
		ChainLength: 1,
		RootHash:    parent,
		FinalHash:   child,
	}, nil
}

// ReconstructContent reconstructs full content from a chain.
//
// This walks the extension chain and concatenates all deltas
// to rebuild the complete content.
func (eg *ExtensionGraph) ReconstructContent(bs *BlockStore, hash *crypto.Hash) ([]byte, error) {
	// Get the chain
	chain, err := eg.GetChain(hash)
	if err != nil {
		return nil, fmt.Errorf("get chain: %w", err)
	}

	// Find root
	root := hash
	if len(chain) > 0 {
		root = chain[0].Parent
	}

	// Get root block
	rootBlock, err := bs.Get(root)
	if err != nil {
		return nil, fmt.Errorf("get root block: %w", err)
	}

	// Start with root data
	content := make([]byte, len(rootBlock.Data))
	copy(content, rootBlock.Data)

	// Apply each delta in sequence
	for _, record := range chain {
		content = append(content, record.Delta...)
	}

	return content, nil
}

// VerifyAndReconstruct verifies a chain and reconstructs content if valid.
func (eg *ExtensionGraph) VerifyAndReconstruct(bs *BlockStore, root, target *crypto.Hash) ([]byte, *VerificationResult, error) {
	// First verify
	result, err := eg.VerifyChain(root, target)
	if err != nil {
		return nil, nil, err
	}

	if !result.Valid {
		return nil, result, nil
	}

	// Verification passed, reconstruct
	content, err := eg.ReconstructContent(bs, target)
	if err != nil {
		return nil, result, fmt.Errorf("reconstruct: %w", err)
	}

	return content, result, nil
}

// VerifyIntegrity performs comprehensive integrity checks on storage.
type IntegrityReport struct {
	TotalBlocks      int
	TotalExtensions  int
	BrokenChains     int
	OrphanedBlocks   int
	InvalidExtensions int
	Errors           []string
}

// VerifyStorageIntegrity checks all extensions for validity.
func (eg *ExtensionGraph) VerifyStorageIntegrity(bs *BlockStore) (*IntegrityReport, error) {
	report := &IntegrityReport{
		Errors: []string{},
	}

	// Count blocks
	blocks, err := bs.List()
	if err != nil {
		return nil, err
	}
	report.TotalBlocks = len(blocks)

	// Check each block for extensions
	for _, hash := range blocks {
		// Check if this block has extensions
		children, err := eg.GetChildren(hash)
		if err != nil {
			continue
		}

		report.TotalExtensions += len(children)

		// Verify each child extension
		for _, child := range children {
			result, err := eg.VerifyExtension(hash, child)
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("Error verifying %s->%s: %v", hash.Hex()[:8], child.Hex()[:8], err))
				continue
			}

			if !result.Valid {
				report.InvalidExtensions++
				report.Errors = append(report.Errors, fmt.Sprintf("Invalid extension %s->%s: %s", hash.Hex()[:8], child.Hex()[:8], result.Reason))
			}
		}
	}

	return report, nil
}
