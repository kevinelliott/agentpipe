package cmd

import (
	"testing"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

func TestParseAgentSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		index       int
		want        agent.AgentConfig
		wantErr     bool
		errContains string
	}{
		// Basic formats
		{
			name:  "type only",
			spec:  "claude",
			index: 0,
			want: agent.AgentConfig{
				ID:    "claude-0",
				Type:  "claude",
				Name:  "claude",
				Model: "",
			},
			wantErr: false,
		},
		{
			name:  "type:name format",
			spec:  "claude:Assistant",
			index: 0,
			want: agent.AgentConfig{
				ID:    "claude-0",
				Type:  "claude",
				Name:  "Assistant",
				Model: "",
			},
			wantErr: false,
		},
		{
			name:  "type:model:name format",
			spec:  "claude:claude-sonnet-4-5:CodeReviewer",
			index: 1,
			want: agent.AgentConfig{
				ID:    "claude-1",
				Type:  "claude",
				Name:  "CodeReviewer",
				Model: "claude-sonnet-4-5",
			},
			wantErr: false,
		},

		// OpenRouter (requires model)
		{
			name:        "openrouter without model (error)",
			spec:        "openrouter:Assistant",
			index:       0,
			wantErr:     true,
			errContains: "requires model specification",
		},
		{
			name:  "openrouter with model",
			spec:  "openrouter:anthropic/claude-sonnet-4-5:Assistant",
			index: 0,
			want: agent.AgentConfig{
				ID:    "openrouter-0",
				Type:  "openrouter",
				Name:  "Assistant",
				Model: "anthropic/claude-sonnet-4-5",
			},
			wantErr: false,
		},
		{
			name:  "openrouter with different model format",
			spec:  "openrouter:google/gemini-2.5-pro:Reviewer",
			index: 2,
			want: agent.AgentConfig{
				ID:    "openrouter-2",
				Type:  "openrouter",
				Name:  "Reviewer",
				Model: "google/gemini-2.5-pro",
			},
			wantErr: false,
		},

		// Agents that don't support model
		{
			name:        "kimi with model (error)",
			spec:        "kimi:kimi-model:Assistant",
			index:       0,
			wantErr:     true,
			errContains: "does not support model specification",
		},
		{
			name:        "cursor with model (error)",
			spec:        "cursor:some-model:Dev",
			index:       0,
			wantErr:     true,
			errContains: "does not support model specification",
		},
		{
			name:        "amp with model (error)",
			spec:        "amp:model:Assistant",
			index:       0,
			wantErr:     true,
			errContains: "does not support model specification",
		},

		// Agents that support but don't require model
		{
			name:  "gemini without model",
			spec:  "gemini:Assistant",
			index: 0,
			want: agent.AgentConfig{
				ID:    "gemini-0",
				Type:  "gemini",
				Name:  "Assistant",
				Model: "",
			},
			wantErr: false,
		},
		{
			name:  "gemini with model",
			spec:  "gemini:gemini-2.5-pro:Assistant",
			index: 1,
			want: agent.AgentConfig{
				ID:    "gemini-1",
				Type:  "gemini",
				Name:  "Assistant",
				Model: "gemini-2.5-pro",
			},
			wantErr: false,
		},
		{
			name:  "qwen with model",
			spec:  "qwen:qwen-plus:CodeGen",
			index: 0,
			want: agent.AgentConfig{
				ID:    "qwen-0",
				Type:  "qwen",
				Name:  "CodeGen",
				Model: "qwen-plus",
			},
			wantErr: false,
		},
		{
			name:  "factory with model",
			spec:  "factory:claude-sonnet-4-5:Builder",
			index: 0,
			want: agent.AgentConfig{
				ID:    "factory-0",
				Type:  "factory",
				Name:  "Builder",
				Model: "claude-sonnet-4-5",
			},
			wantErr: false,
		},
		{
			name:  "qoder with model",
			spec:  "qoder:claude-sonnet-4-5:Reviewer",
			index: 0,
			want: agent.AgentConfig{
				ID:    "qoder-0",
				Type:  "qoder",
				Name:  "Reviewer",
				Model: "claude-sonnet-4-5",
			},
			wantErr: false,
		},
		{
			name:  "codex with model",
			spec:  "codex:gpt-4:Developer",
			index: 0,
			want: agent.AgentConfig{
				ID:    "codex-0",
				Type:  "codex",
				Name:  "Developer",
				Model: "gpt-4",
			},
			wantErr: false,
		},
		{
			name:  "groq with model",
			spec:  "groq:llama3-70b:Assistant",
			index: 0,
			want: agent.AgentConfig{
				ID:    "groq-0",
				Type:  "groq",
				Name:  "Assistant",
				Model: "llama3-70b",
			},
			wantErr: false,
		},
		{
			name:  "crush with model",
			spec:  "crush:deepseek-r1:Thinker",
			index: 0,
			want: agent.AgentConfig{
				ID:    "crush-0",
				Type:  "crush",
				Name:  "Thinker",
				Model: "deepseek-r1",
			},
			wantErr: false,
		},

		// Error cases
		{
			name:        "empty spec",
			spec:        "",
			index:       0,
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "unknown agent type",
			spec:        "unknown:Assistant",
			index:       0,
			wantErr:     true,
			errContains: "unknown agent type",
		},
		{
			name:        "too many parts",
			spec:        "claude:model:name:extra",
			index:       0,
			wantErr:     true,
			errContains: "invalid agent specification format",
		},

		// Edge cases with colons in model/name
		{
			name:  "model with slash (OpenRouter style)",
			spec:  "openrouter:anthropic/claude-3.5-sonnet:MyAgent",
			index: 0,
			want: agent.AgentConfig{
				ID:    "openrouter-0",
				Type:  "openrouter",
				Name:  "MyAgent",
				Model: "anthropic/claude-3.5-sonnet",
			},
			wantErr: false,
		},
		{
			name:  "name with special characters",
			spec:  "claude:claude-sonnet-4-5:Code-Reviewer-v2",
			index: 0,
			want: agent.AgentConfig{
				ID:    "claude-0",
				Type:  "claude",
				Name:  "Code-Reviewer-v2",
				Model: "claude-sonnet-4-5",
			},
			wantErr: false,
		},

		// Auto-generated names
		{
			name:  "type only with index 0",
			spec:  "gemini",
			index: 0,
			want: agent.AgentConfig{
				ID:    "gemini-0",
				Type:  "gemini",
				Name:  "gemini",
				Model: "",
			},
			wantErr: false,
		},
		{
			name:  "type only with index 5",
			spec:  "claude",
			index: 5,
			want: agent.AgentConfig{
				ID:    "claude-5",
				Type:  "claude",
				Name:  "claude",
				Model: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentSpec(tt.spec, tt.index)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseAgentSpec() expected error but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("parseAgentSpec() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// No error expected
			if err != nil {
				t.Errorf("parseAgentSpec() unexpected error = %v", err)
				return
			}

			// Check result
			if got.ID != tt.want.ID {
				t.Errorf("parseAgentSpec() ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Type != tt.want.Type {
				t.Errorf("parseAgentSpec() Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Name != tt.want.Name {
				t.Errorf("parseAgentSpec() Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Model != tt.want.Model {
				t.Errorf("parseAgentSpec() Model = %v, want %v", got.Model, tt.want.Model)
			}
		})
	}
}

func TestParseAgentSpecWithModel(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		wantType    string
		wantModel   string
		wantName    string
		wantErr     bool
		errContains string
	}{
		{
			name:      "single part - type only",
			spec:      "claude",
			wantType:  "claude",
			wantModel: "",
			wantName:  "claude",
			wantErr:   false,
		},
		{
			name:      "two parts - type:name",
			spec:      "gemini:Assistant",
			wantType:  "gemini",
			wantModel: "",
			wantName:  "Assistant",
			wantErr:   false,
		},
		{
			name:      "three parts - type:model:name",
			spec:      "claude:claude-sonnet-4-5:CodeReviewer",
			wantType:  "claude",
			wantModel: "claude-sonnet-4-5",
			wantName:  "CodeReviewer",
			wantErr:   false,
		},
		{
			name:        "empty spec",
			spec:        "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "four parts - invalid",
			spec:        "type:model:name:extra",
			wantErr:     true,
			errContains: "invalid agent specification format",
		},
		{
			name:        "unknown type",
			spec:        "invalidtype:name",
			wantErr:     true,
			errContains: "unknown agent type",
		},
		{
			name:        "agent doesn't support model",
			spec:        "kimi:some-model:name",
			wantErr:     true,
			errContains: "does not support model specification",
		},
		{
			name:        "agent requires model but not provided",
			spec:        "openrouter:name",
			wantErr:     true,
			errContains: "requires model specification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotModel, gotName, err := parseAgentSpecWithModel(tt.spec)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseAgentSpecWithModel() expected error but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("parseAgentSpecWithModel() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseAgentSpecWithModel() unexpected error = %v", err)
				return
			}

			if gotType != tt.wantType {
				t.Errorf("parseAgentSpecWithModel() type = %v, want %v", gotType, tt.wantType)
			}
			if gotModel != tt.wantModel {
				t.Errorf("parseAgentSpecWithModel() model = %v, want %v", gotModel, tt.wantModel)
			}
			if gotName != tt.wantName {
				t.Errorf("parseAgentSpecWithModel() name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestValidateAgentType(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"valid claude", "claude", false},
		{"valid gemini", "gemini", false},
		{"valid openrouter", "openrouter", false},
		{"valid kimi", "kimi", false},
		{"empty type", "", true},
		{"unknown type", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentType(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAgentType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateModelForAgent(t *testing.T) {
	tests := []struct {
		name        string
		agentType   string
		model       string
		wantErr     bool
		errContains string
	}{
		// Agents that support model
		{"claude with model", "claude", "claude-sonnet-4-5", false, ""},
		{"claude without model", "claude", "", false, ""},
		{"gemini with model", "gemini", "gemini-2.5-pro", false, ""},
		{"openrouter with model", "openrouter", "anthropic/claude-sonnet-4-5", false, ""},

		// Agents that require model
		{"openrouter without model", "openrouter", "", true, "requires model specification"},

		// Agents that don't support model
		{"kimi with model", "kimi", "some-model", true, "does not support model specification"},
		{"kimi without model", "kimi", "", false, ""},
		{"cursor with model", "cursor", "model", true, "does not support model specification"},
		{"amp with model", "amp", "model", true, "does not support model specification"},

		// Unknown agent type
		{"unknown type", "unknown", "model", true, "unknown agent type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModelForAgent(tt.agentType, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateModelForAgent() expected error but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("validateModelForAgent() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("validateModelForAgent() unexpected error = %v", err)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
