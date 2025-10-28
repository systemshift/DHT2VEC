package network

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/crypto"
	"github.com/systemshift/tera/semantic"
)

// Node represents a TERA network node.
type Node struct {
	// libp2p host
	host host.Host

	// Pubsub for gossip
	pubsub *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription

	// Gatekeeping
	gatekeeper *core.Gatekeeper

	// Interest filter (what content this node cares about)
	interests []string
	params    semantic.KernelParams

	// Content storage (simple in-memory for MVP)
	content map[string]*core.Content

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
}

// NodeConfig configures a TERA node.
type NodeConfig struct {
	// libp2p listen port (0 for random)
	ListenPort int

	// Bootstrap peers to connect to
	BootstrapPeers []string

	// Node interests (topics to subscribe to)
	Interests []string

	// Kernel parameters for similarity matching
	Params semantic.KernelParams
}

// NewNode creates and starts a new TERA node.
func NewNode(ctx context.Context, config NodeConfig) (*Node, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Create libp2p host
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.ListenPort)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.EnableRelay(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	// Create pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		h.Close()
		return nil, fmt.Errorf("create pubsub: %w", err)
	}

	// Join topic
	topic, err := ps.Join(TopicExtensions)
	if err != nil {
		cancel()
		h.Close()
		return nil, fmt.Errorf("join topic: %w", err)
	}

	// Subscribe to topic
	sub, err := topic.Subscribe()
	if err != nil {
		cancel()
		h.Close()
		return nil, fmt.Errorf("subscribe to topic: %w", err)
	}

	node := &Node{
		host:       h,
		pubsub:     ps,
		topic:      topic,
		sub:        sub,
		gatekeeper: core.NewGatekeeper(),
		interests:  config.Interests,
		params:     config.Params,
		content:    make(map[string]*core.Content),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Connect to bootstrap peers
	for _, peerAddr := range config.BootstrapPeers {
		if err := node.connectToPeer(peerAddr); err != nil {
			fmt.Printf("Warning: failed to connect to bootstrap peer %s: %v\n", peerAddr, err)
		}
	}

	// Start listening for messages
	go node.listen()

	return node, nil
}

// ID returns the node's peer ID.
func (n *Node) ID() peer.ID {
	return n.host.ID()
}

// Addrs returns the node's listen addresses.
func (n *Node) Addrs() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

// FullAddr returns the full multiaddr including peer ID.
func (n *Node) FullAddr() string {
	addrs := n.host.Addrs()
	if len(addrs) == 0 {
		return ""
	}
	return fmt.Sprintf("%s/p2p/%s", addrs[0], n.host.ID())
}

// connectToPeer connects to a peer given its multiaddr.
func (n *Node) connectToPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("parse multiaddr: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("parse peer info: %w", err)
	}

	if err := n.host.Connect(n.ctx, *peerInfo); err != nil {
		return fmt.Errorf("connect to peer: %w", err)
	}

	fmt.Printf("Connected to peer: %s\n", peerInfo.ID)
	return nil
}

// listen handles incoming pubsub messages.
func (n *Node) listen() {
	for {
		msg, err := n.sub.Next(n.ctx)
		if err != nil {
			if n.ctx.Err() != nil {
				// Context cancelled, shutting down
				return
			}
			fmt.Printf("Error reading message: %v\n", err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}

		n.handleMessage(msg.Data)
	}
}

// handleMessage processes an incoming message.
func (n *Node) handleMessage(data []byte) {
	// Unmarshal message
	msg, err := UnmarshalMessage(data)
	if err != nil {
		fmt.Printf("Failed to unmarshal message: %v\n", err)
		return
	}

	switch msg.Type {
	case MessageTypeExtension:
		n.handleExtension(msg)
	case MessageTypeQuery:
		n.handleQuery(msg)
	case MessageTypeQueryResponse:
		n.handleQueryResponse(msg)
	default:
		fmt.Printf("Unknown message type: %s\n", msg.Type)
	}
}

// handleExtension processes an extension announcement.
func (n *Node) handleExtension(msg *Message) {
	payload, err := msg.GetExtensionPayload()
	if err != nil {
		fmt.Printf("Invalid extension payload: %v\n", err)
		return
	}

	ext, err := payload.ToExtension()
	if err != nil {
		fmt.Printf("Failed to convert extension: %v\n", err)
		return
	}

	// Apply gatekeeping against each interest
	shouldForward := false
	for _, interest := range n.interests {
		query := core.NewQuery([]byte(interest), n.params)
		decision := n.gatekeeper.ShouldForward(ext, query)

		if decision.ShouldForward {
			shouldForward = true
			fmt.Printf("✓ Extension passed gatekeeping (interest: %s, similarity: %.3f)\n",
				interest, decision.SimilarityScore)

			// Store the extension
			contentHash := ext.NewHash.Crypto.Hex()
			content := &core.Content{
				Data:     ext.NewData,
				Crypto:   ext.NewHash.Crypto,
				Semantic: ext.NewHash.Semantic,
				ID:       contentHash,
			}
			n.content[contentHash] = content
			break
		}
	}

	if !shouldForward {
		fmt.Printf("✗ Extension blocked by gatekeeping\n")
	}

	// Note: In a full implementation, we would re-publish the message
	// if shouldForward is true. For MVP, messages propagate automatically
	// via pubsub gossip.
}

// handleQuery processes a content query.
func (n *Node) handleQuery(msg *Message) {
	payload, err := msg.GetQueryPayload()
	if err != nil {
		fmt.Printf("Invalid query payload: %v\n", err)
		return
	}

	// Search local content for matches
	query := core.NewQuery(payload.Content, payload.Params)
	var matches []*core.Extension

	for _, content := range n.content {
		if query.Matches(content) {
			// Convert content to extension (simplified)
			ext := &core.Extension{
				NewHash: content.GetDualHash(),
				NewData: content.Data,
			}
			matches = append(matches, ext)
		}
	}

	if len(matches) > 0 {
		fmt.Printf("Found %d matches for query\n", len(matches))
		// In full implementation, send response back to requester
	}
}

// handleQueryResponse processes a query response.
func (n *Node) handleQueryResponse(msg *Message) {
	payload, err := msg.GetQueryResponsePayload()
	if err != nil {
		fmt.Printf("Invalid query response payload: %v\n", err)
		return
	}

	fmt.Printf("Received %d query results for request %s\n", len(payload.Matches), payload.RequestID)
}

// Publish publishes new content to the network.
func (n *Node) Publish(content *core.Content) error {
	// For first publish, treat as extension from zero
	ext := &core.Extension{
		ParentHash: core.DualHash{
			Crypto:   crypto.Zero(),
			Semantic: &semantic.Features{},
		},
		NewData: content.Data,
		NewHash: content.GetDualHash(),
	}

	return n.PublishExtension(ext)
}

// PublishExtension publishes an extension to the network.
func (n *Node) PublishExtension(ext *core.Extension) error {
	// Create message
	msg, err := NewExtensionMessage(ext)
	if err != nil {
		return fmt.Errorf("create extension message: %w", err)
	}

	// Marshal message
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Publish to topic
	if err := n.topic.Publish(n.ctx, data); err != nil {
		return fmt.Errorf("publish to topic: %w", err)
	}

	fmt.Printf("Published extension (crypto: %s)\n", ext.NewHash.Crypto.String()[:20]+"...")
	return nil
}

// GetStats returns gatekeeping statistics.
func (n *Node) GetStats() core.GatekeeperStats {
	return n.gatekeeper.GetStats()
}

// Peers returns the list of connected peers.
func (n *Node) Peers() []peer.ID {
	return n.host.Network().Peers()
}

// Close shuts down the node.
func (n *Node) Close() error {
	n.cancel()
	n.sub.Cancel()
	n.topic.Close()
	return n.host.Close()
}
