package bridge

import (
	"encoding/json"
	"time"
)

// EventType represents the type of streaming event
type EventType string

const (
	// EventBridgeConnected is emitted when the bridge connection is established
	EventBridgeConnected EventType = "bridge.connected"
	// EventConversationStarted is emitted when a conversation begins
	EventConversationStarted EventType = "conversation.started"
	// EventMessageCreated is emitted after each agent completes a message
	EventMessageCreated EventType = "message.created"
	// EventConversationCompleted is emitted when conversation ends normally or reaches max turns
	EventConversationCompleted EventType = "conversation.completed"
	// EventConversationError is emitted when an error occurs during the conversation
	EventConversationError EventType = "conversation.error"
	// EventBridgeTest is emitted when testing the bridge connection
	EventBridgeTest EventType = "bridge.test"
)

// UTCTime wraps time.Time to ensure JSON marshaling always uses UTC with Z suffix
type UTCTime struct {
	time.Time
}

// MarshalJSON implements json.Marshaler to output time in UTC with Z suffix
func (t UTCTime) MarshalJSON() ([]byte, error) {
	// Convert to UTC and format with Z suffix
	utcTime := t.Time.UTC()
	formatted := utcTime.Format("2006-01-02T15:04:05.999999999Z07:00")
	// Ensure we use 'Z' for UTC, not '+00:00'
	if utcTime.Location() == time.UTC {
		formatted = utcTime.Format("2006-01-02T15:04:05.999999999") + "Z"
	}
	return json.Marshal(formatted)
}

// Event represents a streaming event sent to the web app
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp UTCTime     `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// ConversationStartedData contains data for conversation.started events
type ConversationStartedData struct {
	ConversationID string             `json:"conversation_id"`
	Mode           string             `json:"mode"`
	InitialPrompt  string             `json:"initial_prompt"`
	MaxTurns       int                `json:"max_turns,omitempty"`
	Participants   []AgentParticipant `json:"participants"`
	SystemInfo     SystemInfo         `json:"system_info"`
}

// AgentParticipant contains information about an agent participating in the conversation
type AgentParticipant struct {
	AgentType  string `json:"agent_type"`
	Model      string `json:"model,omitempty"`
	Name       string `json:"name,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
	CLIVersion string `json:"cli_version,omitempty"`
}

// MessageCreatedData contains data for message.created events
type MessageCreatedData struct {
	ConversationID string  `json:"conversation_id"`
	MessageID      string  `json:"message_id"`
	AgentType      string  `json:"agent_type"`
	AgentName      string  `json:"agent_name,omitempty"`
	Content        string  `json:"content"`
	SequenceNumber int     `json:"sequence_number,omitempty"`
	TurnNumber     int     `json:"turn_number,omitempty"`
	TokensUsed     int     `json:"tokens_used,omitempty"`
	InputTokens    int     `json:"input_tokens,omitempty"`
	OutputTokens   int     `json:"output_tokens,omitempty"`
	Cost           float64 `json:"cost,omitempty"`
	Model          string  `json:"model,omitempty"`
	DurationMs     int64   `json:"duration_ms,omitempty"`
}

// ConversationCompletedData contains data for conversation.completed events
type ConversationCompletedData struct {
	ConversationID  string  `json:"conversation_id"`
	Status          string  `json:"status"` // "completed", "interrupted", "error"
	TotalMessages   int     `json:"total_messages,omitempty"`
	TotalTurns      int     `json:"total_turns,omitempty"`
	TotalTokens     int     `json:"total_tokens,omitempty"`
	TotalCost       float64 `json:"total_cost,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

// ConversationErrorData contains data for conversation.error events
type ConversationErrorData struct {
	ConversationID string `json:"conversation_id"`
	ErrorMessage   string `json:"error_message"`
	ErrorType      string `json:"error_type,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`
}

// BridgeTestData contains data for bridge.test events
type BridgeTestData struct {
	Message    string     `json:"message"`
	SystemInfo SystemInfo `json:"system_info"`
}

// BridgeConnectedData contains data for bridge.connected events
type BridgeConnectedData struct {
	SystemInfo  SystemInfo `json:"system_info"`
	ConnectedAt string     `json:"connected_at"`
}
