// Package agent provides the core interfaces and types for AI agent communication.
// It defines the Agent interface that all agent implementations must satisfy,
// along with message types and configuration structures.
package agent

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Message represents a single message in an agent conversation.
// Messages can be sent by agents, users, or the system.
type Message struct {
	// AgentID is the unique identifier of the agent or entity that sent the message
	AgentID string
	// AgentName is the display name of the agent
	AgentName string
	// Content is the actual message text
	Content string
	// Timestamp is the Unix timestamp when the message was created
	Timestamp int64
	// Role indicates the message type: "agent", "user", or "system"
	Role string
	// Metrics contains optional performance and cost metrics for agent responses
	Metrics *ResponseMetrics
}

// ResponseMetrics captures performance and cost information for an agent response.
// This is used for monitoring, billing, and optimization purposes.
type ResponseMetrics struct {
	// Duration is how long the agent took to generate the response
	Duration time.Duration
	// InputTokens is the number of tokens in the input (prompt + conversation history)
	InputTokens int
	// OutputTokens is the number of tokens in the agent's response
	OutputTokens int
	// TotalTokens is InputTokens + OutputTokens
	TotalTokens int
	// Model is the specific model used by the agent
	Model string
	// Cost is the estimated monetary cost of the API call in USD
	Cost float64
}

// AgentConfig defines the configuration for creating and initializing an agent.
// It supports both standard fields and custom settings for extensibility.
type AgentConfig struct {
	// ID is the unique identifier for this agent instance
	ID string `yaml:"id"`
	// Type is the agent type (e.g., "claude", "gemini", "copilot")
	Type string `yaml:"type"`
	// Name is the display name for the agent
	Name string `yaml:"name"`
	// Prompt is the system prompt that defines the agent's behavior
	Prompt string `yaml:"prompt"`
	// Announcement is the message shown when the agent joins
	Announcement string `yaml:"announcement"`
	// Model is the specific model to use (e.g., "claude-sonnet-4.5")
	Model string `yaml:"model"`
	// Temperature controls randomness in responses (0.0 to 1.0)
	Temperature float64 `yaml:"temperature"`
	// MaxTokens limits the length of generated responses
	MaxTokens int `yaml:"max_tokens"`
	// CustomSettings allows agent-specific configuration options
	CustomSettings map[string]interface{} `yaml:"custom_settings"`
}

// Agent is the core interface that all agent implementations must satisfy.
// It provides methods for communication, health checking, and metadata access.
type Agent interface {
	// GetID returns the unique identifier of the agent
	GetID() string
	// GetName returns the display name of the agent
	GetName() string
	// GetType returns the agent type (e.g., "claude", "gemini")
	GetType() string
	// GetModel returns the specific model being used
	GetModel() string
	// Initialize configures the agent with the provided configuration
	Initialize(config AgentConfig) error
	// SendMessage sends a message to the agent and returns the response
	SendMessage(ctx context.Context, messages []Message) (string, error)
	// StreamMessage sends a message and streams the response to the writer
	StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error
	// Announce returns the agent's join announcement message
	Announce() string
	// IsAvailable checks if the agent's CLI tool is available
	IsAvailable() bool
	// HealthCheck performs a comprehensive health check of the agent
	HealthCheck(ctx context.Context) error
}

// BaseAgent provides a default implementation of common Agent interface methods.
// Agent implementations can embed BaseAgent to avoid reimplementing basic functionality.
type BaseAgent struct {
	// ID is the unique identifier for this agent instance
	ID string
	// Name is the display name
	Name string
	// Type is the agent type
	Type string
	// Config stores the full agent configuration
	Config AgentConfig
	// Announcement is the custom join message
	Announcement string
}

// GetID returns the unique identifier of the agent.
func (b *BaseAgent) GetID() string {
	return b.ID
}

// GetName returns the display name of the agent.
func (b *BaseAgent) GetName() string {
	return b.Name
}

// GetType returns the agent type (e.g., "claude", "gemini", "copilot").
func (b *BaseAgent) GetType() string {
	return b.Type
}

// GetModel returns the specific model being used by the agent.
// If no model is configured, it falls back to the agent type.
func (b *BaseAgent) GetModel() string {
	if b.Config.Model != "" {
		return b.Config.Model
	}
	// Return type as fallback
	return b.Type
}

// Announce returns the agent's announcement message.
// If a custom announcement is set, it is returned; otherwise,
// a default message is generated using the agent's name.
func (b *BaseAgent) Announce() string {
	if b.Announcement != "" {
		return b.Announcement
	}
	return fmt.Sprintf("%s has joined the conversation.", b.Name)
}

// Initialize configures the BaseAgent with the provided configuration.
// This sets up the basic fields that all agents need.
func (b *BaseAgent) Initialize(config AgentConfig) error {
	b.ID = config.ID
	b.Name = config.Name
	b.Type = config.Type
	b.Config = config
	b.Announcement = config.Announcement
	return nil
}
