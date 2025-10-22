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

// CommandInfo contains information about the agentpipe command that was run
type CommandInfo struct {
	FullCommand    string            `json:"full_command"`             // Complete command as executed
	Args           []string          `json:"args,omitempty"`           // Command arguments
	Mode           string            `json:"mode,omitempty"`           // Orchestrator mode
	MaxTurns       int               `json:"max_turns,omitempty"`      // Maximum turns
	InitialPrompt  string            `json:"initial_prompt,omitempty"` // Initial prompt
	ConfigFile     string            `json:"config_file,omitempty"`    // Config file path
	TUIEnabled     bool              `json:"tui_enabled"`              // TUI mode enabled
	LoggingEnabled bool              `json:"logging_enabled"`          // Logging enabled
	ShowMetrics    bool              `json:"show_metrics"`             // Show metrics
	Timeout        int               `json:"timeout,omitempty"`        // Timeout in seconds
	Options        map[string]string `json:"options,omitempty"`        // Additional options
}

// ConversationStartedData contains data for conversation.started events
type ConversationStartedData struct {
	ConversationID string             `json:"conversation_id"`
	Mode           string             `json:"mode"`
	InitialPrompt  string             `json:"initial_prompt"`
	MaxTurns       int                `json:"max_turns,omitempty"`
	Participants   []AgentParticipant `json:"participants"`
	SystemInfo     SystemInfo         `json:"system_info"`
	Command        *CommandInfo       `json:"command,omitempty"` // Command that started the conversation
}

// AgentParticipant contains information about an agent participating in the conversation
type AgentParticipant struct {
	AgentID    string `json:"agent_id"`              // Unique identifier for this agent instance
	AgentType  string `json:"agent_type"`            // Type of agent (e.g., "claude", "gemini")
	Model      string `json:"model,omitempty"`       // Model used by the agent
	Name       string `json:"name,omitempty"`        // Display name of the agent
	Prompt     string `json:"prompt,omitempty"`      // System prompt for the agent
	CLIVersion string `json:"cli_version,omitempty"` // Version of the agent CLI
}

// MessageCreatedData contains data for message.created events
type MessageCreatedData struct {
	ConversationID string  `json:"conversation_id"`
	MessageID      string  `json:"message_id"`
	AgentID        string  `json:"agent_id"`             // Unique identifier for the agent instance
	AgentType      string  `json:"agent_type"`           // Type of agent (e.g., "claude", "gemini")
	AgentName      string  `json:"agent_name,omitempty"` // Display name of the agent
	Content        string  `json:"content"`              // Message content
	SequenceNumber int     `json:"sequence_number,omitempty"`
	TurnNumber     int     `json:"turn_number,omitempty"`
	TokensUsed     int     `json:"tokens_used,omitempty"`
	InputTokens    int     `json:"input_tokens,omitempty"`
	OutputTokens   int     `json:"output_tokens,omitempty"`
	Cost           float64 `json:"cost,omitempty"`
	Model          string  `json:"model,omitempty"`
	DurationMs     int64   `json:"duration_ms,omitempty"`
}

// SummaryMetadata contains information about the AI-generated conversation summary
type SummaryMetadata struct {
	Text         string  `json:"text"`                    // The summary text
	AgentType    string  `json:"agent_type"`              // Type of agent used to generate summary (e.g., "gemini")
	Model        string  `json:"model,omitempty"`         // Model used for summary generation
	InputTokens  int     `json:"input_tokens,omitempty"`  // Tokens used for input (conversation)
	OutputTokens int     `json:"output_tokens,omitempty"` // Tokens used for output (summary)
	TotalTokens  int     `json:"total_tokens,omitempty"`  // Total tokens used
	Cost         float64 `json:"cost,omitempty"`          // Cost of generating the summary
	DurationMs   int64   `json:"duration_ms,omitempty"`   // Time taken to generate summary
}

// ConversationCompletedData contains data for conversation.completed events
type ConversationCompletedData struct {
	ConversationID  string           `json:"conversation_id"`
	Status          string           `json:"status"` // "completed", "interrupted", "error"
	TotalMessages   int              `json:"total_messages,omitempty"`
	TotalTurns      int              `json:"total_turns,omitempty"`
	TotalTokens     int              `json:"total_tokens,omitempty"`     // Includes summary tokens
	TotalCost       float64          `json:"total_cost,omitempty"`       // Includes summary cost
	DurationSeconds float64          `json:"duration_seconds,omitempty"` // Does not include summary generation time
	Summary         *SummaryMetadata `json:"summary,omitempty"`          // AI-generated conversation summary with metadata
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
