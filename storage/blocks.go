// Package storage implements TERA's extension-aware content storage.
//
// Unlike traditional content-addressed storage (like IPFS), TERA storage
// natively understands and tracks content extensions using homomorphic hashes.
package storage

import (
	"encoding/json"
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/systemshift/tera/crypto"
)

// Block represents a stored content block.
type Block struct {
	// Hash of the block content
	Hash *crypto.Hash

	// The actual data
	Data []byte

	// Size in bytes
	Size int64

	// Metadata
	Timestamp int64
}

// BlockStore handles storage and retrieval of content blocks.
type BlockStore struct {
	db *badger.DB
}

// Key prefixes for different data types
const (
	prefixBlock = "blk:"  // Block data: blk:<hash> → Block
	prefixIndex = "idx:"  // Index data: idx:<hash> → metadata
)

// NewBlockStore creates a new block store backed by BadgerDB.
func NewBlockStore(db *badger.DB) *BlockStore {
	return &BlockStore{db: db}
}

// Put stores a content block.
func (bs *BlockStore) Put(data []byte) (*Block, error) {
	// Compute hash
	hash := crypto.HashElement(data)

	block := &Block{
		Hash: hash,
		Data: data,
		Size: int64(len(data)),
	}

	// Store in database
	err := bs.db.Update(func(txn *badger.Txn) error {
		key := blockKey(hash)

		// Serialize block
		value, err := json.Marshal(block)
		if err != nil {
			return fmt.Errorf("marshal block: %w", err)
		}

		return txn.Set(key, value)
	})

	if err != nil {
		return nil, fmt.Errorf("store block: %w", err)
	}

	return block, nil
}

// Get retrieves a block by hash.
func (bs *BlockStore) Get(hash *crypto.Hash) (*Block, error) {
	var block Block

	err := bs.db.View(func(txn *badger.Txn) error {
		key := blockKey(hash)

		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrBlockNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &block)
		})
	})

	if err != nil {
		return nil, err
	}

	return &block, nil
}

// Has checks if a block exists.
func (bs *BlockStore) Has(hash *crypto.Hash) (bool, error) {
	err := bs.db.View(func(txn *badger.Txn) error {
		key := blockKey(hash)
		_, err := txn.Get(key)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// Delete removes a block from storage.
func (bs *BlockStore) Delete(hash *crypto.Hash) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		key := blockKey(hash)
		return txn.Delete(key)
	})
}

// Size returns the size of a block without loading it.
func (bs *BlockStore) Size(hash *crypto.Hash) (int64, error) {
	block, err := bs.Get(hash)
	if err != nil {
		return 0, err
	}
	return block.Size, nil
}

// List returns all block hashes in storage.
func (bs *BlockStore) List() ([]*crypto.Hash, error) {
	var hashes []*crypto.Hash

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // We only need keys

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(prefixBlock)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()

			// Extract hash from key
			hashHex := string(key[len(prefix):])
			hash, err := crypto.FromHex(hashHex)
			if err != nil {
				continue // Skip invalid keys
			}

			hashes = append(hashes, hash)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return hashes, nil
}

// Count returns the number of blocks stored.
func (bs *BlockStore) Count() (int, error) {
	count := 0

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(prefixBlock)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}
		return nil
	})

	return count, err
}

// TotalSize returns the total size of all blocks.
func (bs *BlockStore) TotalSize() (int64, error) {
	var total int64

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(prefixBlock)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var block Block
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &block)
			})
			if err != nil {
				continue
			}

			total += block.Size
		}
		return nil
	})

	return total, err
}

// blockKey generates the database key for a block.
func blockKey(hash *crypto.Hash) []byte {
	return []byte(prefixBlock + hash.Hex())
}
