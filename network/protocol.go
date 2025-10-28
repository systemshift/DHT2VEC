// Package network implements the TERA networking layer using libp2p.
package network

import (
	"encoding/json"
	"fmt"

	"github.com/systemshift/tera/core"
	"github.com/systemshift/tera/crypto"
	"github.com/systemshift/tera/semantic"
)

// ProtocolVersion identifies the TERA protocol version.
const ProtocolVersion = "tera/1.0.0"

// TopicExtensions is the pubsub topic for extension announcements.
const TopicExtensions = "tera/extensions/v1"

// MessageType identifies the type of message.
type MessageType string

const (
	// MessageTypeExtension announces a new content extension.
	MessageTypeExtension MessageType = "extension"

	// MessageTypeQuery requests similar content.
	MessageTypeQuery MessageType = "query"

	// MessageTypeQueryResponse responds to a query.
	MessageTypeQueryResponse MessageType = "query_response"
)

// Message is the top-level network message format.
type Message struct {
	Type    MessageType `json:"type"`
	Version string      `json:"version"`
	Payload json.RawMessage `json:"payload"`
}

// ExtensionPayload is the payload for extension announcements.
type ExtensionPayload struct {
	// Parent hash (both crypto and semantic)
	ParentCrypto   string                `json:"parent_crypto"`
	ParentSemantic *semantic.Features    `json:"parent_semantic"`

	// New data being added
	NewData []byte `json:"new_data"`

	// Resulting hash after extension
	NewCrypto   string                `json:"new_crypto"`
	NewSemantic *semantic.Features    `json:"new_semantic"`

	// Metadata
	Timestamp int64  `json:"timestamp,omitempty"`
	Publisher string `json:"publisher,omitempty"`
}

// QueryPayload is the payload for content queries.
type QueryPayload struct {
	// Query content
	Content []byte `json:"content"`

	// Kernel parameters
	Params semantic.KernelParams `json:"params"`

	// Optional: specific content hash to query from
	FromHash string `json:"from_hash,omitempty"`

	// Request ID for tracking responses
	RequestID string `json:"request_id"`
}

// QueryResponsePayload is the payload for query responses.
type QueryResponsePayload struct {
	RequestID string              `json:"request_id"`
	Matches   []ExtensionPayload  `json:"matches"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return &Message{
		Type:    msgType,
		Version: ProtocolVersion,
		Payload: payloadBytes,
	}, nil
}

// Marshal serializes a message to JSON.
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal deserializes a message from JSON.
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	// Version check
	if msg.Version != ProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: %s", msg.Version)
	}

	return &msg, nil
}

// GetExtensionPayload extracts the extension payload from a message.
func (m *Message) GetExtensionPayload() (*ExtensionPayload, error) {
	if m.Type != MessageTypeExtension {
		return nil, fmt.Errorf("wrong message type: %s", m.Type)
	}

	var payload ExtensionPayload
	if err := json.Unmarshal(m.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal extension payload: %w", err)
	}

	return &payload, nil
}

// GetQueryPayload extracts the query payload from a message.
func (m *Message) GetQueryPayload() (*QueryPayload, error) {
	if m.Type != MessageTypeQuery {
		return nil, fmt.Errorf("wrong message type: %s", m.Type)
	}

	var payload QueryPayload
	if err := json.Unmarshal(m.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal query payload: %w", err)
	}

	return &payload, nil
}

// GetQueryResponsePayload extracts the query response payload from a message.
func (m *Message) GetQueryResponsePayload() (*QueryResponsePayload, error) {
	if m.Type != MessageTypeQueryResponse {
		return nil, fmt.Errorf("wrong message type: %s", m.Type)
	}

	var payload QueryResponsePayload
	if err := json.Unmarshal(m.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal query response payload: %w", err)
	}

	return &payload, nil
}

// ToExtension converts an ExtensionPayload to a core.Extension.
func (ep *ExtensionPayload) ToExtension() (*core.Extension, error) {
	// Parse crypto hashes
	parentCrypto, err := crypto.FromHex(ep.ParentCrypto)
	if err != nil {
		return nil, fmt.Errorf("parse parent crypto: %w", err)
	}

	newCrypto, err := crypto.FromHex(ep.NewCrypto)
	if err != nil {
		return nil, fmt.Errorf("parse new crypto: %w", err)
	}

	return &core.Extension{
		ParentHash: core.DualHash{
			Crypto:   parentCrypto,
			Semantic: ep.ParentSemantic,
		},
		NewData: ep.NewData,
		NewHash: core.DualHash{
			Crypto:   newCrypto,
			Semantic: ep.NewSemantic,
		},
		Timestamp: ep.Timestamp,
		Publisher: ep.Publisher,
	}, nil
}

// FromExtension creates an ExtensionPayload from a core.Extension.
func FromExtension(ext *core.Extension) *ExtensionPayload {
	return &ExtensionPayload{
		ParentCrypto:   ext.ParentHash.Crypto.Hex(),
		ParentSemantic: ext.ParentHash.Semantic,
		NewData:        ext.NewData,
		NewCrypto:      ext.NewHash.Crypto.Hex(),
		NewSemantic:    ext.NewHash.Semantic,
		Timestamp:      ext.Timestamp,
		Publisher:      ext.Publisher,
	}
}

// NewExtensionMessage creates a message announcing an extension.
func NewExtensionMessage(ext *core.Extension) (*Message, error) {
	payload := FromExtension(ext)
	return NewMessage(MessageTypeExtension, payload)
}

// NewQueryMessage creates a message for a content query.
func NewQueryMessage(query *core.Query, requestID string) (*Message, error) {
	payload := &QueryPayload{
		Content:   query.Content,
		Params:    query.Params,
		RequestID: requestID,
	}
	return NewMessage(MessageTypeQuery, payload)
}

// NewQueryResponseMessage creates a message responding to a query.
func NewQueryResponseMessage(requestID string, matches []*core.Extension) (*Message, error) {
	payloadMatches := make([]ExtensionPayload, len(matches))
	for i, match := range matches {
		payloadMatches[i] = *FromExtension(match)
	}

	payload := &QueryResponsePayload{
		RequestID: requestID,
		Matches:   payloadMatches,
	}
	return NewMessage(MessageTypeQueryResponse, payload)
}
