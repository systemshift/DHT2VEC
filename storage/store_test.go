package storage

import (
	"testing"

	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/crypto"
)

// TestNewStore verifies store creation and cleanup.
func TestNewStore(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if store.blocks == nil {
		t.Error("BlockStore not initialized")
	}

	if store.extensions == nil {
		t.Error("ExtensionGraph not initialized")
	}
}

// TestPutGetContent tests basic content storage.
func TestPutGetContent(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	data := []byte("test content")

	// Put content
	hash, err := store.PutContent(data)
	if err != nil {
		t.Fatalf("PutContent failed: %v", err)
	}

	// Get content
	retrieved, err := store.GetContent(hash)
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("Retrieved content = %s, want %s", retrieved, data)
	}
}

// TestHasContent tests content existence checking.
func TestHasContent(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	data := []byte("test")
	hash, _ := store.PutContent(data)

	// Should exist
	exists, err := store.HasContent(hash)
	if err != nil {
		t.Fatalf("HasContent failed: %v", err)
	}
	if !exists {
		t.Error("Content should exist")
	}

	// Non-existent hash
	fakeHash := crypto.HashElement([]byte("nonexistent"))
	exists, err = store.HasContent(fakeHash)
	if err != nil {
		t.Fatalf("HasContent failed: %v", err)
	}
	if exists {
		t.Error("Fake content should not exist")
	}
}

// TestExtensions tests extension storage and retrieval.
func TestExtensions(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create root content
	root := core.NewContent([]byte("root content"))

	// Store root
	_, err = store.PutContent(root.Data)
	if err != nil {
		t.Fatalf("Store root failed: %v", err)
	}

	// Create extension
	delta := []byte(" extended")
	extended := root.Extend(delta)

	ext := &core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    delta,
		NewHash:    extended.GetDualHash(),
	}

	// Store extension
	err = store.PutExtension(ext)
	if err != nil {
		t.Fatalf("PutExtension failed: %v", err)
	}

	// Retrieve extension
	record, err := store.GetExtension(extended.Crypto)
	if err != nil {
		t.Fatalf("GetExtension failed: %v", err)
	}

	if !record.Parent.Equal(root.Crypto) {
		t.Error("Parent mismatch in extension record")
	}

	if !record.Child.Equal(extended.Crypto) {
		t.Error("Child mismatch in extension record")
	}
}

// TestGetChildren tests retrieving children of a parent.
func TestGetChildren(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Root
	root := core.NewContent([]byte("root"))
	store.PutContent(root.Data)

	// Create two children
	child1 := root.Extend([]byte(" child1"))
	child2 := root.Extend([]byte(" child2"))

	ext1 := &core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" child1"),
		NewHash:    child1.GetDualHash(),
	}

	ext2 := &core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" child2"),
		NewHash:    child2.GetDualHash(),
	}

	store.PutExtension(ext1)
	store.PutExtension(ext2)

	// Get children
	children, err := store.GetChildren(root.Crypto)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("Got %d children, want 2", len(children))
	}
}

// TestGetChain tests retrieving extension chains.
func TestGetChain(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create chain: root -> ext1 -> ext2
	root := core.NewContent([]byte("root"))
	store.PutContent(root.Data)

	ext1 := root.Extend([]byte(" ext1"))
	extRecord1 := &core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" ext1"),
		NewHash:    ext1.GetDualHash(),
	}
	store.PutExtension(extRecord1)

	ext2 := ext1.Extend([]byte(" ext2"))
	extRecord2 := &core.Extension{
		ParentHash: ext1.GetDualHash(),
		NewData:    []byte(" ext2"),
		NewHash:    ext2.GetDualHash(),
	}
	store.PutExtension(extRecord2)

	// Get chain
	chain, err := store.GetChain(ext2.Crypto)
	if err != nil {
		t.Fatalf("GetChain failed: %v", err)
	}

	if len(chain) != 2 {
		t.Errorf("Chain length = %d, want 2", len(chain))
	}

	// Verify order
	if !chain[0].Parent.Equal(root.Crypto) {
		t.Error("First link parent mismatch")
	}
	if !chain[1].Parent.Equal(ext1.Crypto) {
		t.Error("Second link parent mismatch")
	}
}

// TestVerifyChain tests cryptographic chain verification.
func TestVerifyChain(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create valid chain
	root := core.NewContent([]byte("root"))
	store.PutContent(root.Data)

	ext1 := root.Extend([]byte(" ext1"))
	store.PutExtension(&core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" ext1"),
		NewHash:    ext1.GetDualHash(),
	})

	// Verify chain
	result, err := store.VerifyChain(root.Crypto, ext1.Crypto)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Chain should be valid: %s", result.Reason)
	}

	if result.ChainLength != 1 {
		t.Errorf("Chain length = %d, want 1", result.ChainLength)
	}
}

// TestReconstruct tests content reconstruction from chains.
func TestReconstruct(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create chain
	root := core.NewContent([]byte("Hello"))
	store.PutContent(root.Data)

	ext1 := root.Extend([]byte(" World"))
	store.PutExtension(&core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" World"),
		NewHash:    ext1.GetDualHash(),
	})

	ext2 := ext1.Extend([]byte("!"))
	store.PutExtension(&core.Extension{
		ParentHash: ext1.GetDualHash(),
		NewData:    []byte("!"),
		NewHash:    ext2.GetDualHash(),
	})

	// Reconstruct
	content, err := store.Reconstruct(ext2.Crypto)
	if err != nil {
		t.Fatalf("Reconstruct failed: %v", err)
	}

	expected := "Hello World!"
	if string(content) != expected {
		t.Errorf("Reconstructed = %s, want %s", content, expected)
	}
}

// TestGetRoot tests finding the root of a chain.
func TestGetRoot(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create chain
	root := core.NewContent([]byte("root"))
	store.PutContent(root.Data)

	ext1 := root.Extend([]byte(" ext1"))
	store.PutExtension(&core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" ext1"),
		NewHash:    ext1.GetDualHash(),
	})

	ext2 := ext1.Extend([]byte(" ext2"))
	store.PutExtension(&core.Extension{
		ParentHash: ext1.GetDualHash(),
		NewData:    []byte(" ext2"),
		NewHash:    ext2.GetDualHash(),
	})

	// Find root from ext2
	foundRoot, err := store.GetRoot(ext2.Crypto)
	if err != nil {
		t.Fatalf("GetRoot failed: %v", err)
	}

	if !foundRoot.Equal(root.Crypto) {
		t.Errorf("Root = %s, want %s", foundRoot.Hex(), root.Crypto.Hex())
	}
}

// TestGetStats tests statistics retrieval.
func TestGetStats(t *testing.T) {
	store, err := NewStore(InMemoryConfig())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Add some content
	root := core.NewContent([]byte("content"))
	store.PutContent(root.Data)

	ext := root.Extend([]byte(" more"))
	store.PutExtension(&core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" more"),
		NewHash:    ext.GetDualHash(),
	})

	// Get stats
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.BlockCount < 1 {
		t.Error("Expected at least 1 block")
	}

	if stats.ExtensionCount != 1 {
		t.Errorf("ExtensionCount = %d, want 1", stats.ExtensionCount)
	}
}

// BenchmarkPutContent measures content storage performance.
func BenchmarkPutContent(b *testing.B) {
	store, _ := NewStore(InMemoryConfig())
	defer store.Close()

	data := []byte("benchmark data for performance testing")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.PutContent(data)
	}
}

// BenchmarkGetContent measures content retrieval performance.
func BenchmarkGetContent(b *testing.B) {
	store, _ := NewStore(InMemoryConfig())
	defer store.Close()

	data := []byte("test data")
	hash, _ := store.PutContent(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetContent(hash)
	}
}

// BenchmarkPutExtension measures extension storage performance.
func BenchmarkPutExtension(b *testing.B) {
	store, _ := NewStore(InMemoryConfig())
	defer store.Close()

	root := core.NewContent([]byte("root"))
	store.PutContent(root.Data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ext := root.Extend([]byte(" ext"))
		store.PutExtension(&core.Extension{
			ParentHash: root.GetDualHash(),
			NewData:    []byte(" ext"),
			NewHash:    ext.GetDualHash(),
		})
	}
}
