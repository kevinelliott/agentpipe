// Package config provides configuration management for AgentPipe.
// It defines the structure for YAML configuration files and handles
// loading, validation, and default value application.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// Config is the top-level configuration structure for AgentPipe.
// It defines agents, orchestration behavior, logging settings, and bridge streaming.
type Config struct {
	// Version is the configuration file format version
	Version string `yaml:"version"`
	// Agents is the list of agent configurations
	Agents []agent.AgentConfig `yaml:"agents"`
	// Orchestrator defines conversation orchestration settings
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	// Logging defines logging behavior
	Logging LoggingConfig `yaml:"logging"`
	// Bridge defines streaming bridge settings
	Bridge BridgeConfig `yaml:"bridge"`
}

// OrchestratorConfig defines how the orchestrator manages conversations.
type OrchestratorConfig struct {
	// Mode is the orchestration mode: "round-robin", "reactive", or "free-form"
	Mode string `yaml:"mode"`
	// MaxTurns is the maximum number of conversation turns (0 = unlimited)
	MaxTurns int `yaml:"max_turns"`
	// TurnTimeout is the maximum time an agent has to respond
	TurnTimeout time.Duration `yaml:"turn_timeout"`
	// ResponseDelay is the pause between agent responses
	ResponseDelay time.Duration `yaml:"response_delay"`
	// InitialPrompt is an optional starting prompt for the conversation
	InitialPrompt string `yaml:"initial_prompt"`
}

// LoggingConfig defines conversation logging behavior.
type LoggingConfig struct {
	// Enabled determines if conversation logging is active
	Enabled bool `yaml:"enabled"`
	// ChatLogDir is the directory where chat logs are stored
	ChatLogDir string `yaml:"chat_log_dir"`
	// LogFormat is either "text" or "json"
	LogFormat string `yaml:"log_format"`
	// ShowMetrics determines if token/cost metrics are logged
	ShowMetrics bool `yaml:"show_metrics"`
}

// BridgeConfig defines streaming bridge configuration for real-time conversation updates.
type BridgeConfig struct {
	// Enabled determines if streaming bridge is active (disabled by default)
	Enabled bool `yaml:"enabled"`
	// URL is the base URL of the AgentPipe Web app (e.g., https://agentpipe.ai)
	URL string `yaml:"url"`
	// APIKey is the authentication key for the streaming API
	APIKey string `yaml:"api_key"`
	// TimeoutMs is the HTTP request timeout in milliseconds (default: 10000)
	TimeoutMs int `yaml:"timeout_ms"`
	// RetryAttempts is the number of retry attempts for failed requests (default: 3)
	RetryAttempts int `yaml:"retry_attempts"`
	// LogLevel is the logging level for bridge operations: "debug", "info", "warn", "error" (default: "info")
	LogLevel string `yaml:"log_level"`
}

// NewDefaultConfig creates a configuration with sensible defaults.
// The default log directory is ~/.agentpipe/chats.
func NewDefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	defaultLogDir := fmt.Sprintf("%s/.agentpipe/chats", homeDir)

	return &Config{
		Version: "1.0",
		Agents:  []agent.AgentConfig{},
		Orchestrator: OrchestratorConfig{
			Mode:          "round-robin",
			MaxTurns:      10,
			TurnTimeout:   30 * time.Second,
			ResponseDelay: 1 * time.Second,
		},
		Logging: LoggingConfig{
			Enabled:     true,
			ChatLogDir:  defaultLogDir,
			LogFormat:   "text",
			ShowMetrics: false,
		},
	}
}

// LoadConfig loads and validates a configuration from a YAML file.
// It applies default values for any missing optional fields.
// Returns an error if the file cannot be read, parsed, or is invalid.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	config.applyDefaults()

	return &config, nil
}

// SaveConfig writes the configuration to a YAML file.
// The file is created with 0600 permissions (read/write for owner only).
func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks the configuration for errors.
// It ensures at least one agent is configured, all required fields are present,
// agent IDs are unique, and the orchestration mode is valid.
func (c *Config) Validate() error {
	if len(c.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	agentIDs := make(map[string]bool)
	for _, agent := range c.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent ID cannot be empty")
		}
		if agent.Type == "" {
			return fmt.Errorf("agent type cannot be empty for agent %s", agent.ID)
		}
		if agent.Name == "" {
			return fmt.Errorf("agent name cannot be empty for agent %s", agent.ID)
		}
		if agentIDs[agent.ID] {
			return fmt.Errorf("duplicate agent ID: %s", agent.ID)
		}
		agentIDs[agent.ID] = true
	}

	validModes := map[string]bool{
		"round-robin": true,
		"reactive":    true,
		"free-form":   true,
	}

	if c.Orchestrator.Mode != "" && !validModes[c.Orchestrator.Mode] {
		return fmt.Errorf("invalid orchestrator mode: %s", c.Orchestrator.Mode)
	}

	return nil
}

func (c *Config) applyDefaults() {
	if c.Version == "" {
		c.Version = "1.0"
	}

	if c.Orchestrator.Mode == "" {
		c.Orchestrator.Mode = "round-robin"
	}

	if c.Orchestrator.MaxTurns == 0 {
		c.Orchestrator.MaxTurns = 10
	}

	if c.Orchestrator.TurnTimeout == 0 {
		c.Orchestrator.TurnTimeout = 30 * time.Second
	}

	if c.Orchestrator.ResponseDelay == 0 {
		c.Orchestrator.ResponseDelay = 1 * time.Second
	}

	// Logging defaults
	if c.Logging.ChatLogDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		c.Logging.ChatLogDir = fmt.Sprintf("%s/.agentpipe/chats", homeDir)
	}

	if c.Logging.LogFormat == "" {
		c.Logging.LogFormat = "text"
	}

	// Bridge defaults
	// Note: Enabled defaults to false (opt-in), URL handled by internal/bridge
	if c.Bridge.TimeoutMs == 0 {
		c.Bridge.TimeoutMs = 10000
	}
	if c.Bridge.RetryAttempts == 0 {
		c.Bridge.RetryAttempts = 3
	}
	if c.Bridge.LogLevel == "" {
		c.Bridge.LogLevel = "info"
	}

	for i := range c.Agents {
		// Only apply temperature default if not explicitly set (< 0 means not set)
		// Allow 0 as a valid temperature for deterministic outputs
		if c.Agents[i].Temperature < 0 {
			c.Agents[i].Temperature = 0.7
		}
		if c.Agents[i].MaxTokens == 0 {
			c.Agents[i].MaxTokens = 2000
		}
	}
}
