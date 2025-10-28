// Package main demonstrates TERA's core primitives in action.
//
// This example shows:
// 1. Creating content with dual hash
// 2. Extending content with O(1) crypto updates
// 3. Gatekeeping that blocks spam while allowing legitimate content
package main

import (
	"fmt"

	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/semantic"
)

func main() {
	fmt.Println("=== TERA Demo: Integrity-Gated Semantic Search ===\n")

	// 1. Create root content
	fmt.Println("1. Creating root content about machine learning...")
	root := core.NewContent([]byte("Machine learning is a branch of artificial intelligence"))
	fmt.Printf("   Crypto hash: %s\n", root.Crypto.String()[:20]+"...")
	fmt.Printf("   Top keywords: %v\n\n", root.Semantic.TopKeywords)

	// 2. Legitimate extension
	fmt.Println("2. Legitimately extending with more ML content...")
	legit := core.NewExtension(root, []byte(" that focuses on neural networks and deep learning"))
	fmt.Printf("   New crypto hash: %s\n", legit.NewHash.Crypto.String()[:20]+"...")
	fmt.Printf("   Extension verified: %t\n\n", legit.VerifyCrypto())

	// 3. Create a query
	fmt.Println("3. User queries for 'artificial intelligence'...")
	query := core.NewQuery(
		[]byte("artificial intelligence and neural networks"),
		semantic.DefaultParams(),
	)
	query.Params.Threshold = 0.25
	fmt.Printf("   Query threshold: %.2f\n\n", query.Params.Threshold)

	// 4. Test gatekeeping
	fmt.Println("4. Running gatekeeper on legitimate extension...")
	gk := core.NewGatekeeper()
	decision := gk.ShouldForward(legit, query)
	fmt.Printf("   Decision: %t\n", decision.ShouldForward)
	fmt.Printf("   Reason: %s\n", decision.Reason)
	fmt.Printf("   Crypto valid: %t\n", decision.CryptoValid)
	fmt.Printf("   Semantically relevant: %t\n", decision.SemanticRelevant)
	fmt.Printf("   Similarity score: %.3f\n\n", decision.SimilarityScore)

	// 5. Spam attack: fake extension
	fmt.Println("5. Attacker tries to inject spam...")
	spam := &core.Extension{
		ParentHash: root.GetDualHash(),
		NewData:    []byte(" BUY CHEAP PRODUCTS NOW!!!"),
		// Attacker computes wrong hash (doesn't use proper extension)
		NewHash: core.NewContent([]byte("random spam content")).GetDualHash(),
	}

	spamDecision := gk.ShouldForward(spam, query)
	fmt.Printf("   Decision: %t\n", spamDecision.ShouldForward)
	fmt.Printf("   Reason: %s\n", spamDecision.Reason)
	fmt.Printf("   Crypto valid: %t (BLOCKED!)\n\n", spamDecision.CryptoValid)

	// 6. Irrelevant but valid extension
	fmt.Println("6. Valid extension but irrelevant topic (cooking)...")
	cookingRoot := core.NewContent([]byte("Italian cooking recipes"))
	irrelevant := core.NewExtension(cookingRoot, []byte(" with fresh pasta"))

	irrelevantDecision := gk.ShouldForward(irrelevant, query)
	fmt.Printf("   Decision: %t\n", irrelevantDecision.ShouldForward)
	fmt.Printf("   Reason: %s\n", irrelevantDecision.Reason)
	fmt.Printf("   Crypto valid: %t\n", irrelevantDecision.CryptoValid)
	fmt.Printf("   Semantically relevant: %t (BLOCKED!)\n\n", irrelevantDecision.SemanticRelevant)

	// 7. Statistics
	fmt.Println("7. Gatekeeper statistics:")
	stats := gk.GetStats()
	fmt.Printf("   Total seen: %d\n", stats.TotalSeen)
	fmt.Printf("   Forwarded: %d\n", stats.Forwarded)
	fmt.Printf("   Blocked (crypto): %d\n", stats.CryptoBlocked)
	fmt.Printf("   Blocked (semantic): %d\n", stats.SemanticBlocked)
	fmt.Printf("   Block rate: %.1f%%\n\n", stats.BlockRate*100)

	// 8. Gossip simulation
	fmt.Println("8. Simulating gossip across 3 nodes...")
	sim := core.NewGossipSimulator()

	params := semantic.DefaultParams()
	params.Threshold = 0.2 // Lower threshold for gossip

	node1 := core.NewSimulatedNode("Alice", []string{"machine learning and artificial intelligence"}, params)
	node2 := core.NewSimulatedNode("Bob", []string{"cooking recipes and food"}, params)
	node3 := core.NewSimulatedNode("Carol", []string{"artificial intelligence neural networks"}, params)

	sim.AddNode(node1)
	sim.AddNode(node2)
	sim.AddNode(node3)

	forwarded := sim.PropagateExtension(legit)
	fmt.Printf("   Extension about ML forwarded by %d/%d nodes\n", forwarded, len(sim.Nodes))
	fmt.Printf("   Alice (ML interest): %d received\n", len(node1.Received))
	fmt.Printf("   Bob (cooking interest): %d received\n", len(node2.Received))
	fmt.Printf("   Carol (ML+AI interest): %d received\n\n", len(node3.Received))

	fmt.Println("=== Demo Complete ===")
	fmt.Println("\nKey takeaway: TERA enables spam-resistant semantic search by")
	fmt.Println("combining cryptographic verification (gate 1) with semantic")
	fmt.Println("relevance filtering (gate 2). Invalid and irrelevant content")
	fmt.Println("is automatically blocked, while legitimate relevant content flows.")
}
