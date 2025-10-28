package core

import (
	"fmt"

	"github.com/systemshift/tera/semantic"
)

// GatekeeperDecision represents the result of gatekeeping logic.
type GatekeeperDecision struct {
	ShouldForward bool
	Reason        string

	// Details
	CryptoValid       bool
	SemanticRelevant  bool
	SimilarityScore   float64
}

// Gatekeeper implements the core spam filtering logic.
//
// This is the innovation: nodes only forward extensions if BOTH:
// 1. Cryptographically valid (legitimate extension)
// 2. Semantically relevant (similar to query/interest)
type Gatekeeper struct {
	// Optional: track statistics
	TotalSeen     int
	CryptoBlocked int
	SemanticBlocked int
	Forwarded     int
}

// NewGatekeeper creates a new gatekeeper.
func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{}
}

// ShouldForward is THE KEY FUNCTION of TERA.
//
// Returns true only if:
// 1. Extension is cryptographically valid (gate 1)
// 2. Extension is semantically relevant to query (gate 2)
//
// This dual gating prevents spam while enabling discovery.
func (gk *Gatekeeper) ShouldForward(ext *Extension, query *Query) GatekeeperDecision {
	gk.TotalSeen++

	decision := GatekeeperDecision{}

	// Gate 1: Cryptographic verification
	decision.CryptoValid = ext.VerifyCrypto()
	if !decision.CryptoValid {
		gk.CryptoBlocked++
		decision.ShouldForward = false
		decision.Reason = "crypto verification failed: invalid extension"
		return decision
	}

	// Gate 2: Semantic relevance
	decision.SimilarityScore = ext.ComputeSemanticSimilarity(query.Features, query.Params)
	decision.SemanticRelevant = decision.SimilarityScore >= query.Params.Threshold

	if !decision.SemanticRelevant {
		gk.SemanticBlocked++
		decision.ShouldForward = false
		decision.Reason = fmt.Sprintf("semantic filter: similarity %.3f < threshold %.3f",
			decision.SimilarityScore, query.Params.Threshold)
		return decision
	}

	// Both gates passed
	gk.Forwarded++
	decision.ShouldForward = true
	decision.Reason = fmt.Sprintf("passed: valid extension, similarity %.3f", decision.SimilarityScore)
	return decision
}

// ShouldForwardSimple is a simplified version that just returns bool.
func (gk *Gatekeeper) ShouldForwardSimple(ext *Extension, query *Query) bool {
	return gk.ShouldForward(ext, query).ShouldForward
}

// Stats returns statistics about gatekeeping decisions.
type GatekeeperStats struct {
	TotalSeen       int
	CryptoBlocked   int
	SemanticBlocked int
	Forwarded       int
	BlockRate       float64
}

// GetStats returns current statistics.
func (gk *Gatekeeper) GetStats() GatekeeperStats {
	blockRate := 0.0
	if gk.TotalSeen > 0 {
		blocked := gk.CryptoBlocked + gk.SemanticBlocked
		blockRate = float64(blocked) / float64(gk.TotalSeen)
	}

	return GatekeeperStats{
		TotalSeen:       gk.TotalSeen,
		CryptoBlocked:   gk.CryptoBlocked,
		SemanticBlocked: gk.SemanticBlocked,
		Forwarded:       gk.Forwarded,
		BlockRate:       blockRate,
	}
}

// Reset clears statistics.
func (gk *Gatekeeper) Reset() {
	gk.TotalSeen = 0
	gk.CryptoBlocked = 0
	gk.SemanticBlocked = 0
	gk.Forwarded = 0
}

// InterestFilter represents a node's interest profile.
// Nodes use this to decide what content to pay attention to.
type InterestFilter struct {
	// Keywords or topics of interest
	Interests []string

	// Parameters for matching
	Params semantic.KernelParams
}

// NewInterestFilter creates an interest filter from keywords.
func NewInterestFilter(interests []string, params semantic.KernelParams) *InterestFilter {
	return &InterestFilter{
		Interests: interests,
		Params:    params,
	}
}

// Matches checks if content matches this interest filter.
func (filter *InterestFilter) Matches(content *Content) bool {
	for _, interest := range filter.Interests {
		query := NewQuery([]byte(interest), filter.Params)
		if query.Matches(content) {
			return true
		}
	}
	return false
}

// GossipSimulator simulates gossip propagation with gatekeeping.
// Useful for testing and understanding the system.
type GossipSimulator struct {
	Nodes []*SimulatedNode
}

// SimulatedNode represents a node in the gossip simulation.
type SimulatedNode struct {
	ID         string
	Interests  *InterestFilter
	Gatekeeper *Gatekeeper
	Received   []*Extension
}

// NewSimulatedNode creates a simulated node.
func NewSimulatedNode(id string, interests []string, params semantic.KernelParams) *SimulatedNode {
	return &SimulatedNode{
		ID:         id,
		Interests:  NewInterestFilter(interests, params),
		Gatekeeper: NewGatekeeper(),
		Received:   []*Extension{},
	}
}

// ProcessExtension processes an incoming extension.
// Returns true if the node would forward it to peers.
func (node *SimulatedNode) ProcessExtension(ext *Extension) bool {
	// Check against each interest
	for _, interest := range node.Interests.Interests {
		query := NewQuery([]byte(interest), node.Interests.Params)
		decision := node.Gatekeeper.ShouldForward(ext, query)

		if decision.ShouldForward {
			node.Received = append(node.Received, ext)
			return true
		}
	}

	return false
}

// NewGossipSimulator creates a gossip simulator.
func NewGossipSimulator() *GossipSimulator {
	return &GossipSimulator{
		Nodes: []*SimulatedNode{},
	}
}

// AddNode adds a node to the simulation.
func (sim *GossipSimulator) AddNode(node *SimulatedNode) {
	sim.Nodes = append(sim.Nodes, node)
}

// PropagateExtension simulates gossip propagation of an extension.
// Returns how many nodes forwarded it.
func (sim *GossipSimulator) PropagateExtension(ext *Extension) int {
	forwarded := 0

	for _, node := range sim.Nodes {
		if node.ProcessExtension(ext) {
			forwarded++
		}
	}

	return forwarded
}

// GetStats returns aggregate statistics across all nodes.
func (sim *GossipSimulator) GetStats() map[string]GatekeeperStats {
	stats := make(map[string]GatekeeperStats)

	for _, node := range sim.Nodes {
		stats[node.ID] = node.Gatekeeper.GetStats()
	}

	return stats
}
