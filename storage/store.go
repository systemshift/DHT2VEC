package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/crypto"
)

var (
	// ErrBlockNotFound is returned when a block doesn't exist.
	ErrBlockNotFound = errors.New("block not found")

	// ErrExtensionNotFound is returned when an extension record doesn't exist.
	ErrExtensionNotFound = errors.New("extension not found")

	// ErrInvalidExtension is returned when an extension is invalid.
	ErrInvalidExtension = errors.New("invalid extension")
)

// Store is the main storage interface combining blocks and extensions.
type Store struct {
	db         *badger.DB
	blocks     *BlockStore
	extensions *ExtensionGraph
	path       string
}

// Config configures the storage system.
type Config struct {
	// Path to storage directory
	Path string

	// InMemory mode (for testing)
	InMemory bool
}

// NewStore creates a new TERA storage instance.
func NewStore(config Config) (*Store, error) {
	// Ensure directory exists
	if !config.InMemory {
		if err := os.MkdirAll(config.Path, 0755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}

	// Open BadgerDB
	opts := badger.DefaultOptions(config.Path)
	if config.InMemory {
		opts = opts.WithInMemory(true)
	}
	opts = opts.WithLogger(nil) // Disable logging for now

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}

	return &Store{
		db:         db,
		blocks:     NewBlockStore(db),
		extensions: NewExtensionGraph(db),
		path:       config.Path,
	}, nil
}

// Close closes the storage system.
func (s *Store) Close() error {
	return s.db.Close()
}

// Path returns the storage directory path.
func (s *Store) Path() string {
	return s.path
}

// PutContent stores new content and returns its hash.
func (s *Store) PutContent(data []byte) (*crypto.Hash, error) {
	block, err := s.blocks.Put(data)
	if err != nil {
		return nil, err
	}
	return block.Hash, nil
}

// GetContent retrieves content by hash.
func (s *Store) GetContent(hash *crypto.Hash) ([]byte, error) {
	block, err := s.blocks.Get(hash)
	if err != nil {
		return nil, err
	}
	return block.Data, nil
}

// HasContent checks if content exists.
func (s *Store) HasContent(hash *crypto.Hash) (bool, error) {
	return s.blocks.Has(hash)
}

// PutExtension stores an extension and updates the graph.
func (s *Store) PutExtension(ext *core.Extension) error {
	// Store the delta as a block
	_, err := s.blocks.Put(ext.NewData)
	if err != nil {
		return fmt.Errorf("store delta: %w", err)
	}

	// Record the extension relationship
	err = s.extensions.AddExtension(
		ext.ParentHash.Crypto,
		ext.NewHash.Crypto,
		ext.NewData,
		ext.NewHash,
	)
	if err != nil {
		return fmt.Errorf("add extension: %w", err)
	}

	return nil
}

// GetExtension retrieves an extension record.
func (s *Store) GetExtension(hash *crypto.Hash) (*ExtensionRecord, error) {
	return s.extensions.GetExtension(hash)
}

// GetChildren returns all children of a content hash.
func (s *Store) GetChildren(parent *crypto.Hash) ([]*crypto.Hash, error) {
	return s.extensions.GetChildren(parent)
}

// GetChain returns the full extension chain.
func (s *Store) GetChain(hash *crypto.Hash) ([]*ExtensionRecord, error) {
	return s.extensions.GetChain(hash)
}

// GetRoot finds the root of a chain.
func (s *Store) GetRoot(hash *crypto.Hash) (*crypto.Hash, error) {
	return s.extensions.GetRoot(hash)
}

// VerifyChain verifies an extension chain.
func (s *Store) VerifyChain(root, target *crypto.Hash) (*VerificationResult, error) {
	return s.extensions.VerifyChain(root, target)
}

// VerifyExtension verifies a single extension.
func (s *Store) VerifyExtension(parent, child *crypto.Hash) (*VerificationResult, error) {
	return s.extensions.VerifyExtension(parent, child)
}

// Reconstruct rebuilds content from an extension chain.
func (s *Store) Reconstruct(hash *crypto.Hash) ([]byte, error) {
	return s.extensions.ReconstructContent(s.blocks, hash)
}

// VerifyAndReconstruct verifies then reconstructs.
func (s *Store) VerifyAndReconstruct(root, target *crypto.Hash) ([]byte, *VerificationResult, error) {
	return s.extensions.VerifyAndReconstruct(s.blocks, root, target)
}

// Stats returns storage statistics.
type Stats struct {
	BlockCount      int
	ExtensionCount  int
	TotalSize       int64
	StoragePath     string
}

// GetStats returns current storage statistics.
func (s *Store) GetStats() (*Stats, error) {
	blockCount, err := s.blocks.Count()
	if err != nil {
		return nil, err
	}

	totalSize, err := s.blocks.TotalSize()
	if err != nil {
		return nil, err
	}

	// Count extensions (approximate)
	blocks, err := s.blocks.List()
	if err != nil {
		return nil, err
	}

	extensionCount := 0
	for _, block := range blocks {
		children, err := s.extensions.GetChildren(block)
		if err != nil {
			continue
		}
		extensionCount += len(children)
	}

	return &Stats{
		BlockCount:     blockCount,
		ExtensionCount: extensionCount,
		TotalSize:      totalSize,
		StoragePath:    s.path,
	}, nil
}

// GarbageCollect removes unreferenced blocks.
//
// This is a simple implementation that could be optimized.
func (s *Store) GarbageCollect(keepRoots []*crypto.Hash) (int, error) {
	// Mark phase: find all reachable blocks
	reachable := make(map[string]bool)

	for _, root := range keepRoots {
		reachable[root.Hex()] = true

		// Get all descendants
		descendants, err := s.extensions.GetAllDescendants(root)
		if err != nil {
			continue
		}

		for _, desc := range descendants {
			reachable[desc.Hex()] = true
		}
	}

	// Sweep phase: delete unreachable blocks
	deleted := 0
	blocks, err := s.blocks.List()
	if err != nil {
		return 0, err
	}

	for _, block := range blocks {
		if !reachable[block.Hex()] {
			if err := s.blocks.Delete(block); err != nil {
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}

// Compact runs BadgerDB compaction.
func (s *Store) Compact() error {
	return s.db.RunValueLogGC(0.5)
}

// Backup creates a backup of the database.
func (s *Store) Backup(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = s.db.Backup(f, 0)
	return err
}

// Restore restores from a backup.
func Restore(backupPath, storePath string) error {
	// Create new empty database
	opts := badger.DefaultOptions(storePath)
	db, err := badger.Open(opts)
	if err != nil {
		return err
	}
	defer db.Close()

	// Open backup file
	f, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return db.Load(f, 256)
}

// DefaultConfig returns a default storage configuration.
func DefaultConfig(dataDir string) Config {
	return Config{
		Path:     filepath.Join(dataDir, "storage"),
		InMemory: false,
	}
}

// InMemoryConfig returns a configuration for in-memory storage (testing).
func InMemoryConfig() Config {
	return Config{
		Path:     "",
		InMemory: true,
	}
}
