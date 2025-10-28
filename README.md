# TERA

**Terrestrial Extendable Retrieval & Addressing**

A semantic content discovery network with cryptographic integrity guarantees.

## What is TERA?

TERA is a peer-to-peer network that enables similarity-based content discovery while preventing spam through cryptographic verification. Unlike traditional content-addressed networks (like IPFS) that require exact content hashes, TERA allows you to find "similar" content with provable integrity.

## The Core Innovation

Traditional DHTs have a fundamental limitation: they can only verify exact content identity. TERA introduces a **dual-output hash function** that provides:

1. **H_crypto** - A homomorphic hash supporting O(1) incremental extensions
2. **H_semantic** - Parameterized kernel-based similarity metrics

This enables **integrity-gated semantic search**: nodes can verify that content legitimately extends a root hash while filtering spam based on relevance.

### Key Properties

- **Extendable**: Add content to a collection in O(1) time without recomputing the entire hash
- **Verifiable**: Cryptographically prove that content B extends content A
- **Spam-resistant**: Invalid extensions are automatically rejected by the network
- **Parameterized**: Users define their own notion of "similarity" via kernel parameters

## How It Works

```
Traditional IPFS:
Content → SHA-256 → Exact lookup → Retrieve

TERA:
Content → (H_crypto, H_semantic) → Similarity search + Verification → Discover
```

**Example:**

```go
// Publish root content
root := tera.NewContent([]byte("Initial document"))
// H_crypto: 0xabc123..., H_semantic: [features...]

// Extend with new content
extended := root.Extend([]byte("Additional paragraph"))
// H_crypto: 0xdef456... (= 0xabc123... + hash("Additional paragraph"))

// Query with custom similarity parameters
query := tera.Query{
    Content: []byte("Looking for documents about..."),
    Params: KernelParams{
        WeightSemantic: 0.7,
        WeightLexical:  0.3,
    },
}

// Network forwards only if:
// 1. H_crypto verifies (legitimate extension)
// 2. Similarity exceeds threshold (relevant)
```

## Architecture

```
┌─────────────────────────────────────────┐
│ Application Layer                       │
│ (Queries, Publications, Subscriptions)  │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ Gossip Protocol + Gatekeeping           │
│ (Forward if: valid_extension ∧ similar) │
└─────────────────────────────────────────┘
                  ↓
┌──────────────────┬──────────────────────┐
│ H_crypto         │ H_semantic           │
│ (Homomorphic)    │ (Kernel-based)       │
│ Integrity ✓      │ Discovery ✓          │
└──────────────────┴──────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ libp2p Transport Layer                  │
└─────────────────────────────────────────┘
```

## Project Status

**Phase 1: Primitives** (Complete ✓)
- [x] Homomorphic hash implementation (`crypto/`)
- [x] Parameterized kernel functions (`semantic/`)
- [x] Gatekeeping logic (`core/`)
- [x] Working demo (`examples/demo.go`)

**Phase 2: Network** (Next)
- [ ] libp2p integration
- [ ] Gossip protocol
- [ ] Basic node implementation

**Phase 3: Applications**
- [ ] Content discovery API
- [ ] Interest subscription mechanism
- [ ] IPFS integration for storage

## Quick Start

```bash
# Run tests
go test ./...

# Run demo
go run examples/demo.go
```

## Why "Terrestrial"?

It's a pun. IPFS is "InterPlanetary" File System, so naturally this is the "Terrestrial" version. Supposedly inverted priorities: we're focused on Earth-based problems like spam, discovery, and semantic search rather than interplanetary data transfer.

## Related Work

- **IPFS**: Content-addressed storage (exact matching only)
- **DHT2VEC** (2020): Early exploration of semantic DHTs
- **Kademlia**: XOR-metric DHT routing (no semantic awareness)

TERA combines ideas from content-addressed networks, homomorphic cryptography, and kernel methods to create something new: **semantic content addressing with integrity**.

## License

BSD 3-Clause License (see LICENSE file)

## Contributing

This project is in early development. Contributions welcome once the core primitives are stable.
