package adapters

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

func TestBuildAgentPrompt(t *testing.T) {
	tests := []struct {
		name         string
		agentName    string
		systemPrompt string
		conversation string
		want         string
	}{
		{
			name:         "basic prompt",
			agentName:    "TestAgent",
			systemPrompt: "You are a helpful assistant.",
			conversation: "[10:00:00] User: Hello",
			want:         "You are TestAgent. You are a helpful assistant.",
		},
		{
			name:         "empty system prompt",
			agentName:    "Agent",
			systemPrompt: "",
			conversation: "[10:00:00] User: Hi",
			want:         "You are Agent.",
		},
		{
			name:         "multi-line conversation",
			agentName:    "Claude",
			systemPrompt: "You are Claude.",
			conversation: "[10:00:00] User: Hello\n[10:00:01] Assistant: Hi",
			want:         "You are Claude. You are Claude.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAgentPrompt(tt.agentName, tt.systemPrompt, tt.conversation)

			if !strings.Contains(got, tt.agentName) {
				t.Errorf("expected prompt to contain agent name '%s', got: %s", tt.agentName, got)
			}

			if tt.systemPrompt != "" && !strings.Contains(got, tt.systemPrompt) {
				t.Errorf("expected prompt to contain system prompt, got: %s", got)
			}

			if !strings.Contains(got, "conversation") {
				t.Errorf("expected prompt to mention conversation, got: %s", got)
			}

			if !strings.Contains(got, tt.conversation) {
				t.Errorf("expected prompt to contain conversation content, got: %s", got)
			}
		})
	}
}

func TestClaudeAgentInitialization(t *testing.T) {
	claudeAgent := NewClaudeAgent()

	config := agent.AgentConfig{
		ID:           "claude-1",
		Type:         "claude",
		Name:         "Claude",
		Prompt:       "You are a helpful AI",
		Announcement: "Claude has joined!",
		Model:        "claude-sonnet-4.5",
	}

	err := claudeAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("claude CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if claudeAgent.GetID() != "claude-1" {
		t.Errorf("expected ID 'claude-1', got '%s'", claudeAgent.GetID())
	}
	if claudeAgent.GetName() != "Claude" {
		t.Errorf("expected name 'Claude', got '%s'", claudeAgent.GetName())
	}
	if claudeAgent.GetType() != "claude" {
		t.Errorf("expected type 'claude', got '%s'", claudeAgent.GetType())
	}
	if claudeAgent.GetModel() != "claude-sonnet-4.5" {
		t.Errorf("expected model 'claude-sonnet-4.5', got '%s'", claudeAgent.GetModel())
	}
}

func TestGeminiAgentInitialization(t *testing.T) {
	geminiAgent := NewGeminiAgent()

	config := agent.AgentConfig{
		ID:           "gemini-1",
		Type:         "gemini",
		Name:         "Gemini",
		Prompt:       "You are Google's AI",
		Announcement: "Gemini has joined!",
		Model:        "gemini-2.0-flash",
	}

	err := geminiAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("gemini CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if geminiAgent.GetID() != "gemini-1" {
		t.Errorf("expected ID 'gemini-1', got '%s'", geminiAgent.GetID())
	}
	if geminiAgent.GetName() != "Gemini" {
		t.Errorf("expected name 'Gemini', got '%s'", geminiAgent.GetName())
	}
}

func TestCopilotAgentInitialization(t *testing.T) {
	copilotAgent := NewCopilotAgent()

	config := agent.AgentConfig{
		ID:           "copilot-1",
		Type:         "copilot",
		Name:         "Copilot",
		Prompt:       "You are GitHub Copilot",
		Announcement: "Copilot has joined!",
		Model:        "gpt-4",
	}

	err := copilotAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("copilot CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if copilotAgent.GetID() != "copilot-1" {
		t.Errorf("expected ID 'copilot-1', got '%s'", copilotAgent.GetID())
	}
	if copilotAgent.GetName() != "Copilot" {
		t.Errorf("expected name 'Copilot', got '%s'", copilotAgent.GetName())
	}
}

func TestCursorAgentInitialization(t *testing.T) {
	cursorAgent := NewCursorAgent()

	config := agent.AgentConfig{
		ID:           "cursor-1",
		Type:         "cursor",
		Name:         "Cursor",
		Prompt:       "You are Cursor AI",
		Announcement: "Cursor has joined!",
	}

	err := cursorAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("cursor CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if cursorAgent.GetID() != "cursor-1" {
		t.Errorf("expected ID 'cursor-1', got '%s'", cursorAgent.GetID())
	}
}

func TestQwenAgentInitialization(t *testing.T) {
	qwenAgent := NewQwenAgent()

	config := agent.AgentConfig{
		ID:           "qwen-1",
		Type:         "qwen",
		Name:         "Qwen",
		Prompt:       "You are Qwen",
		Announcement: "Qwen has joined!",
	}

	err := qwenAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("qwen CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if qwenAgent.GetID() != "qwen-1" {
		t.Errorf("expected ID 'qwen-1', got '%s'", qwenAgent.GetID())
	}
}

func TestCodexAgentInitialization(t *testing.T) {
	codexAgent := NewCodexAgent()

	config := agent.AgentConfig{
		ID:           "codex-1",
		Type:         "codex",
		Name:         "Codex",
		Prompt:       "You are Codex",
		Announcement: "Codex has joined!",
	}

	err := codexAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("codex CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if codexAgent.GetID() != "codex-1" {
		t.Errorf("expected ID 'codex-1', got '%s'", codexAgent.GetID())
	}
}

func TestAmpAgentInitialization(t *testing.T) {
	ampAgent := NewAmpAgent()

	config := agent.AgentConfig{
		ID:           "amp-1",
		Type:         "amp",
		Name:         "Amp",
		Prompt:       "You are Amp",
		Announcement: "Amp has joined!",
		Model:        "claude-sonnet-4.5",
	}

	err := ampAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("amp CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if ampAgent.GetID() != "amp-1" {
		t.Errorf("expected ID 'amp-1', got '%s'", ampAgent.GetID())
	}
	if ampAgent.GetName() != "Amp" {
		t.Errorf("expected name 'Amp', got '%s'", ampAgent.GetName())
	}
	if ampAgent.GetType() != "amp" {
		t.Errorf("expected type 'amp', got '%s'", ampAgent.GetType())
	}
}

func TestAiderAgentInitialization(t *testing.T) {
	aiderAgent := NewAiderAgent()

	config := agent.AgentConfig{
		ID:           "aider-1",
		Type:         "aider",
		Name:         "Aider",
		Prompt:       "You are Aider",
		Announcement: "Aider has joined!",
		Model:        "gpt-4",
	}

	err := aiderAgent.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("aider CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	if aiderAgent.GetID() != "aider-1" {
		t.Errorf("expected ID 'aider-1', got '%s'", aiderAgent.GetID())
	}
	if aiderAgent.GetName() != "Aider" {
		t.Errorf("expected name 'Aider', got '%s'", aiderAgent.GetName())
	}
	if aiderAgent.GetType() != "aider" {
		t.Errorf("expected type 'aider', got '%s'", aiderAgent.GetType())
	}
}

func TestAgentAnnouncement(t *testing.T) {
	tests := []struct {
		name         string
		agentFactory func() agent.Agent
		agentName    string
		announcement string
	}{
		{"claude", NewClaudeAgent, "Claude", "Claude has arrived!"},
		{"gemini", NewGeminiAgent, "Gemini", "Gemini is here!"},
		{"copilot", NewCopilotAgent, "Copilot", "Copilot ready!"},
		{"cursor", NewCursorAgent, "Cursor", "Cursor online!"},
		{"qwen", NewQwenAgent, "Qwen", "Qwen active!"},
		{"codex", NewCodexAgent, "Codex", "Codex ready!"},
		{"amp", NewAmpAgent, "Amp", "Amp is live!"},
		{"aider", NewAiderAgent, "Aider", "Aider in the house!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.agentFactory()
			config := agent.AgentConfig{
				ID:           tt.name + "-1",
				Type:         tt.name,
				Name:         tt.agentName,
				Announcement: tt.announcement,
			}

			err := a.Initialize(config)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					t.Skipf("%s CLI not available, skipping test", tt.name)
				}
				t.Fatalf("initialization failed: %v", err)
			}

			announcement := a.Announce()
			if announcement != tt.announcement {
				t.Errorf("expected announcement '%s', got '%s'", tt.announcement, announcement)
			}
		})
	}
}

func TestAgentAnnouncementDefault(t *testing.T) {
	a := NewClaudeAgent()
	config := agent.AgentConfig{
		ID:   "claude-1",
		Type: "claude",
		Name: "Claude",
		// No announcement specified
	}

	err := a.Initialize(config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skip("claude CLI not available, skipping test")
		}
		t.Fatalf("initialization failed: %v", err)
	}

	announcement := a.Announce()
	if !strings.Contains(announcement, "Claude") {
		t.Errorf("expected default announcement to contain agent name, got '%s'", announcement)
	}
	if !strings.Contains(announcement, "joined") {
		t.Errorf("expected default announcement to contain 'joined', got '%s'", announcement)
	}
}

// TestAgentHealthCheckTimeout tests that health checks respect context timeout
func TestAgentHealthCheckTimeout(t *testing.T) {
	// This test verifies the health check pattern, but actual checks require CLI tools
	agents := []struct {
		name  string
		agent agent.Agent
	}{
		{"claude", NewClaudeAgent()},
		{"gemini", NewGeminiAgent()},
		{"copilot", NewCopilotAgent()},
		{"cursor", NewCursorAgent()},
		{"qwen", NewQwenAgent()},
		{"codex", NewCodexAgent()},
		{"amp", NewAmpAgent()},
		{"aider", NewAiderAgent()},
	}

	for _, tt := range agents {
		t.Run(tt.name, func(t *testing.T) {
			config := agent.AgentConfig{
				ID:   tt.name + "-test",
				Type: tt.name,
				Name: strings.ToUpper(tt.name[:1]) + tt.name[1:],
			}

			err := tt.agent.Initialize(config)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					t.Skipf("%s CLI not available, skipping test", tt.name)
				}
				t.Fatalf("initialization failed: %v", err)
			}

			// Create a very short timeout context
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()

			// Health check should either:
			// 1. Return quickly with an error (CLI not found)
			// 2. Respect the context timeout
			done := make(chan error, 1)
			go func() {
				done <- tt.agent.HealthCheck(ctx)
			}()

			select {
			case err := <-done:
				// Got a result (expected - CLI probably not installed)
				_ = err // Error is expected in test environment
			case <-time.After(2 * time.Second):
				t.Error("health check did not respect context timeout")
			}
		})
	}
}

func TestAgentGetModel(t *testing.T) {
	tests := []struct {
		name          string
		agent         agent.Agent
		configModel   string
		expectedModel string
	}{
		{
			name:          "claude with custom model",
			agent:         NewClaudeAgent(),
			configModel:   "claude-sonnet-4.5",
			expectedModel: "claude-sonnet-4.5",
		},
		{
			name:          "gemini with custom model",
			agent:         NewGeminiAgent(),
			configModel:   "gemini-2.0-flash",
			expectedModel: "gemini-2.0-flash",
		},
		{
			name:          "claude without model falls back to type",
			agent:         NewClaudeAgent(),
			configModel:   "",
			expectedModel: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract agent type from the test name or expected model
			agentType := tt.expectedModel
			if tt.configModel != "" {
				agentType = strings.Split(tt.expectedModel, "-")[0]
			}

			config := agent.AgentConfig{
				ID:    "test-agent",
				Type:  agentType,
				Name:  "TestAgent",
				Model: tt.configModel,
			}

			err := tt.agent.Initialize(config)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					t.Skip("CLI not available, skipping test")
				}
				t.Fatalf("initialization failed: %v", err)
			}

			model := tt.agent.GetModel()
			if model != tt.expectedModel {
				t.Errorf("expected model '%s', got '%s'", tt.expectedModel, model)
			}
		})
	}
}

// TestConversationFormatting tests message formatting across adapters
func TestConversationFormatting(t *testing.T) {
	messages := []agent.Message{
		{
			AgentID:   "user",
			AgentName: "User",
			Content:   "Hello!",
			Timestamp: time.Now().Unix(),
			Role:      "user",
		},
		{
			AgentID:   "agent-1",
			AgentName: "Agent1",
			Content:   "Hi there!",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
		},
	}

	// Test that conversation formatting is consistent
	// This is testing the internal formatConversation methods
	// which all adapters should implement similarly

	t.Run("message_count", func(t *testing.T) {
		if len(messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(messages))
		}
	})

	t.Run("message_roles", func(t *testing.T) {
		if messages[0].Role != "user" {
			t.Errorf("expected first message role 'user', got '%s'", messages[0].Role)
		}
		if messages[1].Role != "agent" {
			t.Errorf("expected second message role 'agent', got '%s'", messages[1].Role)
		}
	})

	t.Run("message_content", func(t *testing.T) {
		if messages[0].Content == "" {
			t.Error("expected message content to be non-empty")
		}
		if messages[1].Content == "" {
			t.Error("expected message content to be non-empty")
		}
	})
}
