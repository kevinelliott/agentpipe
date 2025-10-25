// Package providers manages AI provider configurations and pricing data.
// It integrates with Catwalk's provider configs to provide accurate cost estimates.
package providers

// Provider represents an AI service provider configuration.
type Provider struct {
	// Name is the display name of the provider (e.g., "Anthropic", "OpenAI")
	Name string `json:"name"`
	// ID is the unique identifier for the provider (e.g., "anthropic", "openai")
	ID string `json:"id"`
	// Type indicates the API compatibility (e.g., "openai", "anthropic")
	Type string `json:"type"`
	// APIKey is the environment variable name for the API key
	APIKey string `json:"api_key,omitempty"`
	// APIEndpoint is the base URL for API requests
	APIEndpoint string `json:"api_endpoint,omitempty"`
	// DefaultLargeModelID is the default large model for this provider
	DefaultLargeModelID string `json:"default_large_model_id,omitempty"`
	// DefaultSmallModelID is the default small model for this provider
	DefaultSmallModelID string `json:"default_small_model_id,omitempty"`
	// Models is the list of available models for this provider
	Models []Model `json:"models"`
}

// Model represents a specific AI model with its pricing and capabilities.
type Model struct {
	// ID is the unique identifier for the model (e.g., "claude-sonnet-4-5-20250929")
	ID string `json:"id"`
	// Name is the display name of the model
	Name string `json:"name"`
	// CostPer1MIn is the cost per 1 million input tokens in USD
	CostPer1MIn float64 `json:"cost_per_1m_in"`
	// CostPer1MOut is the cost per 1 million output tokens in USD
	CostPer1MOut float64 `json:"cost_per_1m_out"`
	// CostPer1MInCached is the cost per 1 million cached input tokens in USD
	CostPer1MInCached float64 `json:"cost_per_1m_in_cached,omitempty"`
	// CostPer1MOutCached is the cost per 1 million cached output tokens in USD
	CostPer1MOutCached float64 `json:"cost_per_1m_out_cached,omitempty"`
	// ContextWindow is the maximum context length in tokens
	ContextWindow int `json:"context_window"`
	// DefaultMaxTokens is the default maximum output tokens
	DefaultMaxTokens int `json:"default_max_tokens"`
	// CanReason indicates if the model supports extended reasoning
	CanReason bool `json:"can_reason"`
	// SupportsAttachments indicates if the model supports file attachments
	SupportsAttachments bool `json:"supports_attachments"`
}

// ProviderConfig represents the consolidated provider configuration file.
type ProviderConfig struct {
	// Version is the schema version for future compatibility
	Version string `json:"version"`
	// UpdatedAt is the timestamp when this config was last updated
	UpdatedAt string `json:"updated_at"`
	// Source indicates where this config was fetched from
	Source string `json:"source"`
	// Providers is the list of all provider configurations
	Providers []Provider `json:"providers"`
}
