package config

import (
	"strings"
	"testing"
	"time"
	
	"github.com/kevinelliott/agentpipe/pkg/agent"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.Version != "1.0" {
		t.Errorf("Expected Version to be '1.0', got %s", cfg.Version)
	}

	if cfg.Orchestrator.Mode != "round-robin" {
		t.Errorf("Expected default mode to be 'round-robin', got %s", cfg.Orchestrator.Mode)
	}

	if cfg.Orchestrator.MaxTurns != 10 {
		t.Errorf("Expected default MaxTurns to be 10, got %d", cfg.Orchestrator.MaxTurns)
	}

	if cfg.Orchestrator.TurnTimeout != 30*time.Second {
		t.Errorf("Expected default TurnTimeout to be 30s, got %v", cfg.Orchestrator.TurnTimeout)
	}

	if !cfg.Logging.Enabled {
		t.Error("Expected logging to be enabled by default")
	}

	if !strings.Contains(cfg.Logging.ChatLogDir, ".agentpipe/chats") {
		t.Errorf("Expected ChatLogDir to contain '.agentpipe/chats', got %s", cfg.Logging.ChatLogDir)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty agents",
			config: &Config{
				Agents: []agent.AgentConfig{},
			},
			wantErr: true,
			errMsg:  "at least one agent must be configured",
		},
		{
			name: "duplicate agent IDs",
			config: &Config{
				Agents: []agent.AgentConfig{
					{ID: "agent1", Type: "claude", Name: "Agent 1"},
					{ID: "agent1", Type: "gemini", Name: "Agent 2"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate agent ID",
		},
		{
			name: "invalid mode",
			config: &Config{
				Agents: []agent.AgentConfig{
					{ID: "agent1", Type: "claude", Name: "Agent 1"},
				},
				Orchestrator: OrchestratorConfig{
					Mode: "invalid-mode",
				},
			},
			wantErr: true,
			errMsg:  "invalid orchestrator mode",
		},
		{
			name: "valid config",
			config: &Config{
				Agents: []agent.AgentConfig{
					{ID: "agent1", Type: "claude", Name: "Agent 1"},
					{ID: "agent2", Type: "gemini", Name: "Agent 2"},
				},
				Orchestrator: OrchestratorConfig{
					Mode:     "round-robin",
					MaxTurns: 10,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
			}
		})
	}
}