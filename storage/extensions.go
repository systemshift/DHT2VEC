package storage

import (
	"encoding/json"
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/crypto"
)

// ExtensionRecord tracks the relationship between parent and child hashes.
type ExtensionRecord struct {
	// Parent hash
	Parent *crypto.Hash

	// Child hash (parent + delta)
	Child *crypto.Hash

	// The delta data that was added
	Delta []byte

	// Dual hash of the child
	ChildDualHash core.DualHash

	// Metadata
	Timestamp int64
	Publisher string
}

// MarshalJSON implements custom JSON marshaling for ExtensionRecord.
func (er ExtensionRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Parent        string         `json:"parent"`
		Child         string         `json:"child"`
		Delta         []byte         `json:"delta"`
		ChildDualHash core.DualHash  `json:"child_dual_hash"`
		Timestamp     int64          `json:"timestamp,omitempty"`
		Publisher     string         `json:"publisher,omitempty"`
	}{
		Parent:        er.Parent.Hex(),
		Child:         er.Child.Hex(),
		Delta:         er.Delta,
		ChildDualHash: er.ChildDualHash,
		Timestamp:     er.Timestamp,
		Publisher:     er.Publisher,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for ExtensionRecord.
func (er *ExtensionRecord) UnmarshalJSON(data []byte) error {
	var aux struct {
		Parent        string         `json:"parent"`
		Child         string         `json:"child"`
		Delta         []byte         `json:"delta"`
		ChildDualHash core.DualHash  `json:"child_dual_hash"`
		Timestamp     int64          `json:"timestamp,omitempty"`
		Publisher     string         `json:"publisher,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	parent, err := crypto.FromHex(aux.Parent)
	if err != nil {
		return fmt.Errorf("unmarshal parent hash: %w", err)
	}

	child, err := crypto.FromHex(aux.Child)
	if err != nil {
		return fmt.Errorf("unmarshal child hash: %w", err)
	}

	er.Parent = parent
	er.Child = child
	er.Delta = aux.Delta
	er.ChildDualHash = aux.ChildDualHash
	er.Timestamp = aux.Timestamp
	er.Publisher = aux.Publisher

	return nil
}

// ExtensionGraph tracks the graph of content extensions.
type ExtensionGraph struct {
	db *badger.DB
}

// Key prefixes for extension graph
const (
	prefixExtension = "ext:"      // Extension record: ext:<child> → ExtensionRecord
	prefixChildren  = "children:" // Children index: children:<parent> → []child_hash
	prefixRoot      = "root:"     // Root index: root:<root> → []all_descendant_hashes
)

// NewExtensionGraph creates a new extension graph.
func NewExtensionGraph(db *badger.DB) *ExtensionGraph {
	return &ExtensionGraph{db: db}
}

// AddExtension records a new extension relationship.
func (eg *ExtensionGraph) AddExtension(parent, child *crypto.Hash, delta []byte, childDualHash core.DualHash) error {
	record := &ExtensionRecord{
		Parent:        parent,
		Child:         child,
		Delta:         delta,
		ChildDualHash: childDualHash,
	}

	return eg.db.Update(func(txn *badger.Txn) error {
		// Store extension record
		extKey := extensionKey(child)
		extValue, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal extension: %w", err)
		}

		if err := txn.Set(extKey, extValue); err != nil {
			return err
		}

		// Update children index
		if err := eg.addChild(txn, parent, child); err != nil {
			return err
		}

		// Update root index
		root := eg.findRootInTxn(txn, parent)
		if err := eg.addDescendant(txn, root, child); err != nil {
			return err
		}

		return nil
	})
}

// GetExtension retrieves an extension record.
func (eg *ExtensionGraph) GetExtension(hash *crypto.Hash) (*ExtensionRecord, error) {
	var record ExtensionRecord

	err := eg.db.View(func(txn *badger.Txn) error {
		key := extensionKey(hash)
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrExtensionNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &record)
		})
	})

	return &record, err
}

// GetParent returns the parent hash of a content hash.
func (eg *ExtensionGraph) GetParent(hash *crypto.Hash) (*crypto.Hash, error) {
	record, err := eg.GetExtension(hash)
	if err != nil {
		return nil, err
	}
	return record.Parent, nil
}

// GetChildren returns all direct children of a hash.
func (eg *ExtensionGraph) GetChildren(parent *crypto.Hash) ([]*crypto.Hash, error) {
	var children []*crypto.Hash

	err := eg.db.View(func(txn *badger.Txn) error {
		key := childrenKey(parent)
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // No children
			}
			return err
		}

		return item.Value(func(val []byte) error {
			var hexHashes []string
			if err := json.Unmarshal(val, &hexHashes); err != nil {
				return err
			}

			for _, hex := range hexHashes {
				h, err := crypto.FromHex(hex)
				if err != nil {
					continue
				}
				children = append(children, h)
			}
			return nil
		})
	})

	return children, err
}

// GetRoot finds the root of an extension chain.
func (eg *ExtensionGraph) GetRoot(hash *crypto.Hash) (*crypto.Hash, error) {
	var root *crypto.Hash

	err := eg.db.View(func(txn *badger.Txn) error {
		root = eg.findRootInTxn(txn, hash)
		return nil
	})

	return root, err
}

// GetChain returns the full chain from root to the given hash.
func (eg *ExtensionGraph) GetChain(hash *crypto.Hash) ([]*ExtensionRecord, error) {
	var chain []*ExtensionRecord

	current := hash
	for {
		record, err := eg.GetExtension(current)
		if err != nil {
			if err == ErrExtensionNotFound {
				// Reached the root
				break
			}
			return nil, err
		}

		// Prepend (we're walking backwards)
		chain = append([]*ExtensionRecord{record}, chain...)
		current = record.Parent
	}

	return chain, nil
}

// GetAllDescendants returns all content hashes that extend from a root.
func (eg *ExtensionGraph) GetAllDescendants(root *crypto.Hash) ([]*crypto.Hash, error) {
	var descendants []*crypto.Hash

	err := eg.db.View(func(txn *badger.Txn) error {
		key := rootKey(root)
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // No descendants
			}
			return err
		}

		return item.Value(func(val []byte) error {
			var hexHashes []string
			if err := json.Unmarshal(val, &hexHashes); err != nil {
				return err
			}

			for _, hex := range hexHashes {
				h, err := crypto.FromHex(hex)
				if err != nil {
					continue
				}
				descendants = append(descendants, h)
			}
			return nil
		})
	})

	return descendants, err
}

// IsExtension checks if child extends parent.
func (eg *ExtensionGraph) IsExtension(parent, child *crypto.Hash) (bool, error) {
	record, err := eg.GetExtension(child)
	if err != nil {
		if err == ErrExtensionNotFound {
			return false, nil
		}
		return false, err
	}

	return record.Parent.Equal(parent), nil
}

// Helper: add a child to parent's children list
func (eg *ExtensionGraph) addChild(txn *badger.Txn, parent, child *crypto.Hash) error {
	key := childrenKey(parent)

	// Get existing children
	var hexHashes []string
	item, err := txn.Get(key)
	if err == nil {
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &hexHashes)
		})
		if err != nil {
			return err
		}
	}

	// Add new child
	hexHashes = append(hexHashes, child.Hex())

	// Save updated list
	value, err := json.Marshal(hexHashes)
	if err != nil {
		return err
	}

	return txn.Set(key, value)
}

// Helper: add a descendant to root's descendants list
func (eg *ExtensionGraph) addDescendant(txn *badger.Txn, root, descendant *crypto.Hash) error {
	key := rootKey(root)

	// Get existing descendants
	var hexHashes []string
	item, err := txn.Get(key)
	if err == nil {
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &hexHashes)
		})
		if err != nil {
			return err
		}
	}

	// Add new descendant
	hexHashes = append(hexHashes, descendant.Hex())

	// Save updated list
	value, err := json.Marshal(hexHashes)
	if err != nil {
		return err
	}

	return txn.Set(key, value)
}

// Helper: find root by walking up the chain
func (eg *ExtensionGraph) findRootInTxn(txn *badger.Txn, hash *crypto.Hash) *crypto.Hash {
	current := hash

	for {
		key := extensionKey(current)
		item, err := txn.Get(key)
		if err != nil {
			// No parent found, this is the root
			return current
		}

		var record ExtensionRecord
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &record)
		})
		if err != nil {
			return current
		}

		current = record.Parent
	}
}

// Key generators
func extensionKey(hash *crypto.Hash) []byte {
	return []byte(prefixExtension + hash.Hex())
}

func childrenKey(parent *crypto.Hash) []byte {
	return []byte(prefixChildren + parent.Hex())
}

func rootKey(root *crypto.Hash) []byte {
	return []byte(prefixRoot + root.Hex())
}
