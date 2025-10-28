// Command tera-node runs a TERA network node.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/network"
	"github.com/systemshift/tera/semantic"
)

func main() {
	// Command-line flags
	port := flag.Int("port", 0, "Listen port (0 for random)")
	bootstrap := flag.String("bootstrap", "", "Bootstrap peer multiaddr")
	interests := flag.String("interests", "machine learning,artificial intelligence", "Comma-separated list of interests")
	threshold := flag.Float64("threshold", 0.3, "Similarity threshold (0.0-1.0)")
	flag.Parse()

	// Parse interests
	interestList := strings.Split(*interests, ",")
	for i := range interestList {
		interestList[i] = strings.TrimSpace(interestList[i])
	}

	// Create node config
	config := network.NodeConfig{
		ListenPort: *port,
		Interests:  interestList,
		Params: semantic.KernelParams{
			WeightSemantic:   0.6,
			WeightLexical:    0.3,
			WeightStructural: 0.1,
			Threshold:        *threshold,
		},
	}

	if *bootstrap != "" {
		config.BootstrapPeers = []string{*bootstrap}
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start node
	fmt.Println("Starting TERA node...")
	node, err := network.NewNode(ctx, config)
	if err != nil {
		fmt.Printf("Failed to start node: %v\n", err)
		os.Exit(1)
	}
	defer node.Close()

	fmt.Printf("\n=== TERA Node Started ===\n")
	fmt.Printf("Peer ID: %s\n", node.ID())
	fmt.Printf("Listen address: %s\n", node.FullAddr())
	fmt.Printf("Interests: %v\n", interestList)
	fmt.Printf("Threshold: %.2f\n\n", *threshold)

	if *bootstrap != "" {
		fmt.Printf("Connected to bootstrap: %s\n\n", *bootstrap)
	} else {
		fmt.Println("Running as bootstrap node (no peers to connect to)\n")
	}

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Interactive shell
	fmt.Println("Commands:")
	fmt.Println("  publish <text>  - Publish new content")
	fmt.Println("  stats           - Show gatekeeping statistics")
	fmt.Println("  peers           - Show connected peers")
	fmt.Println("  quit            - Shut down node")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		cmd := parts[0]

		switch cmd {
		case "publish":
			if len(parts) < 2 {
				fmt.Println("Usage: publish <text>")
				continue
			}

			text := parts[1]
			content := core.NewContent([]byte(text))

			if err := node.Publish(content); err != nil {
				fmt.Printf("Failed to publish: %v\n", err)
			} else {
				fmt.Printf("Published: %s\n", text)
				fmt.Printf("Crypto hash: %s\n", content.Crypto.String()[:20]+"...")
			}

		case "stats":
			stats := node.GetStats()
			fmt.Printf("\n=== Gatekeeping Statistics ===\n")
			fmt.Printf("Total seen:        %d\n", stats.TotalSeen)
			fmt.Printf("Forwarded:         %d\n", stats.Forwarded)
			fmt.Printf("Blocked (crypto):  %d\n", stats.CryptoBlocked)
			fmt.Printf("Blocked (semantic): %d\n", stats.SemanticBlocked)
			fmt.Printf("Block rate:        %.1f%%\n\n", stats.BlockRate*100)

		case "peers":
			peers := node.Peers()
			fmt.Printf("\n=== Connected Peers (%d) ===\n", len(peers))
			for _, p := range peers {
				fmt.Printf("  %s\n", p)
			}
			fmt.Println()

		case "quit", "exit":
			fmt.Println("Shutting down...")
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}
