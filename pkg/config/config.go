package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

type Config struct {
	Version      string              `yaml:"version"`
	Agents       []agent.AgentConfig `yaml:"agents"`
	Orchestrator OrchestratorConfig  `yaml:"orchestrator"`
	Logging      LoggingConfig       `yaml:"logging"`
}

type OrchestratorConfig struct {
	Mode          string        `yaml:"mode"`
	MaxTurns      int           `yaml:"max_turns"`
	TurnTimeout   time.Duration `yaml:"turn_timeout"`
	ResponseDelay time.Duration `yaml:"response_delay"`
	InitialPrompt string        `yaml:"initial_prompt"`
}

type LoggingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	ChatLogDir  string `yaml:"chat_log_dir"`
	LogFormat   string `yaml:"log_format"` // "text" or "json"
	ShowMetrics bool   `yaml:"show_metrics"`
}

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
