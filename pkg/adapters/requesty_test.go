package adapters

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

func TestNewRequestyAgent(t *testing.T) {
	a := NewRequestyAgent()
	if a == nil {
		t.Fatal("NewRequestyAgent returned nil")
	}

	_, ok := a.(*RequestyAgent)
	if !ok {
		t.Error("NewRequestyAgent did not return *RequestyAgent")
	}
}

func TestRequestyAgent_Initialize(t *testing.T) {
	tests := []struct {
		name        string
		config      agent.AgentConfig
		apiKey      string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "successful initialization",
			config: agent.AgentConfig{
				ID:     "test-1",
				Type:   "requesty",
				Name:   "Test Requesty",
				Model:  "anthropic/claude-sonnet-4-5",
				Prompt: "You are a helpful assistant",
			},
			apiKey:      "test-api-key",
			shouldError: false,
		},
		{
			name: "missing api key",
			config: agent.AgentConfig{
				ID:     "test-2",
				Type:   "requesty",
				Name:   "Test Requesty",
				Model:  "openai/gpt-4o-mini",
				Prompt: "You are a helpful assistant",
			},
			apiKey:      "",
			shouldError: true,
			errorMsg:    "REQUESTY_API_KEY",
		},
		{
			name: "missing model",
			config: agent.AgentConfig{
				ID:     "test-3",
				Type:   "requesty",
				Name:   "Test Requesty",
				Prompt: "You are a helpful assistant",
			},
			apiKey:      "test-api-key",
			shouldError: true,
			errorMsg:    "model must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.apiKey != "" {
				os.Setenv("REQUESTY_API_KEY", tt.apiKey)
				defer os.Unsetenv("REQUESTY_API_KEY")
			} else {
				os.Unsetenv("REQUESTY_API_KEY")
			}

			a := NewRequestyAgent()
			err := a.Initialize(tt.config)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify initialization
				requestyAgent, ok := a.(*RequestyAgent)
				if !ok {
					t.Fatal("Agent is not *RequestyAgent")
				}

				if requestyAgent.GetID() != tt.config.ID {
					t.Errorf("Expected ID %s, got %s", tt.config.ID, requestyAgent.GetID())
				}

				if requestyAgent.GetName() != tt.config.Name {
					t.Errorf("Expected Name %s, got %s", tt.config.Name, requestyAgent.GetName())
				}

				if requestyAgent.GetModel() != tt.config.Model {
					t.Errorf("Expected Model %s, got %s", tt.config.Model, requestyAgent.GetModel())
				}

				if requestyAgent.client == nil {
					t.Error("Expected client to be initialized, got nil")
				}
			}
		})
	}
}

func TestRequestyAgent_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		available bool
	}{
		{
			name:      "api key set",
			apiKey:    "test-api-key",
			available: true,
		},
		{
			name:      "api key not set",
			apiKey:    "",
			available: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.apiKey != "" {
				os.Setenv("REQUESTY_API_KEY", tt.apiKey)
				defer os.Unsetenv("REQUESTY_API_KEY")
			} else {
				os.Unsetenv("REQUESTY_API_KEY")
			}

			a := NewRequestyAgent()
			available := a.IsAvailable()

			if available != tt.available {
				t.Errorf("Expected IsAvailable() to return %v, got %v", tt.available, available)
			}
		})
	}
}

func TestRequestyAgent_GetCLIVersion(t *testing.T) {
	a := NewRequestyAgent()
	version := a.GetCLIVersion()

	expected := "N/A (API)"
	if version != expected {
		t.Errorf("Expected GetCLIVersion() to return '%s', got '%s'", expected, version)
	}
}

func TestRequestyAgent_BuildConversationHistory(t *testing.T) {
	// Set up environment
	os.Setenv("REQUESTY_API_KEY", "test-api-key")
	defer os.Unsetenv("REQUESTY_API_KEY")

	a := NewRequestyAgent()
	config := agent.AgentConfig{
		ID:     "test-agent",
		Type:   "requesty",
		Name:   "Test Agent",
		Model:  "openai/gpt-4o-mini",
		Prompt: "You are a helpful assistant",
	}

	if err := a.Initialize(config); err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	requestyAgent, ok := a.(*RequestyAgent)
	if !ok {
		t.Fatal("Agent is not *RequestyAgent")
	}

	messages := []agent.Message{
		{
			AgentID:   "system",
			AgentName: "System",
			Role:      "system",
			Content:   "Initial prompt: Let's discuss AI",
			Timestamp: 1000,
		},
		{
			AgentID:   "other-agent",
			AgentName: "Other Agent",
			Role:      "agent",
			Content:   "AI is fascinating!",
			Timestamp: 2000,
		},
		{
			AgentID:   "test-agent",
			AgentName: "Test Agent",
			Role:      "agent",
			Content:   "I agree, let's explore it",
			Timestamp: 3000,
		},
		{
			AgentID:   "user-1",
			AgentName: "User",
			Role:      "user",
			Content:   "What are your thoughts?",
			Timestamp: 4000,
		},
	}

	apiMessages := requestyAgent.buildConversationHistory(messages)

	// Should have:
	// 1. System prompt from config
	// 2. System message (converted to user role)
	// 3. Other agent's message (converted to user role)
	// 4. Test agent's own message (skipped)
	// 5. User message
	// Total: 4 messages

	if len(apiMessages) != 4 {
		t.Fatalf("Expected 4 API messages, got %d", len(apiMessages))
	}

	// Check first message (system prompt from config)
	if apiMessages[0].Role != "system" {
		t.Errorf("Expected first message role to be 'system', got '%s'", apiMessages[0].Role)
	}
	if apiMessages[0].Content != "You are a helpful assistant" {
		t.Errorf("Expected first message to be system prompt, got: %s", apiMessages[0].Content)
	}

	// Check second message (system message converted to user)
	if apiMessages[1].Role != "user" {
		t.Errorf("Expected second message role to be 'user', got '%s'", apiMessages[1].Role)
	}
	if !strings.Contains(apiMessages[1].Content, "[System]") {
		t.Errorf("Expected system message to be prefixed with [System], got: %s", apiMessages[1].Content)
	}

	// Check third message (other agent's message)
	if apiMessages[2].Role != "user" {
		t.Errorf("Expected third message role to be 'user', got '%s'", apiMessages[2].Role)
	}
	if !strings.Contains(apiMessages[2].Content, "Other Agent:") {
		t.Errorf("Expected agent message to include agent name, got: %s", apiMessages[2].Content)
	}

	// Check fourth message (actual user message)
	if apiMessages[3].Role != "user" {
		t.Errorf("Expected fourth message role to be 'user', got '%s'", apiMessages[3].Role)
	}
	if apiMessages[3].Content != "What are your thoughts?" {
		t.Errorf("Expected user message content, got: %s", apiMessages[3].Content)
	}
}

func TestRequestyAgent_HealthCheck_NotInitialized(t *testing.T) {
	a := NewRequestyAgent()
	ctx := context.Background()

	err := a.HealthCheck(ctx)

	if err == nil {
		t.Error("Expected error for uninitialized agent, got nil")
	}

	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Expected 'not initialized' error, got: %v", err)
	}
}

// Integration tests (skipped if REQUESTY_API_KEY is not set)

func TestRequestyAgent_HealthCheck_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("REQUESTY_API_KEY")
	if apiKey == "" {
		t.Skip("REQUESTY_API_KEY not set, skipping integration test")
	}

	a := NewRequestyAgent()
	config := agent.AgentConfig{
		ID:     "test-health",
		Type:   "requesty",
		Name:   "Health Check Test",
		Model:  "openai/gpt-4o-mini",
		Prompt: "You are a test assistant",
	}

	if err := a.Initialize(config); err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	ctx := context.Background()
	err := a.HealthCheck(ctx)

	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestRequestyAgent_SendMessage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("REQUESTY_API_KEY")
	if apiKey == "" {
		t.Skip("REQUESTY_API_KEY not set, skipping integration test")
	}

	a := NewRequestyAgent()
	config := agent.AgentConfig{
		ID:        "test-send",
		Type:      "requesty",
		Name:      "Send Message Test",
		Model:     "openai/gpt-4o-mini",
		Prompt:    "You are a test assistant. Keep responses very short.",
		MaxTokens: 20,
	}

	if err := a.Initialize(config); err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	messages := []agent.Message{
		{
			AgentID:   "user",
			AgentName: "User",
			Role:      "user",
			Content:   "Say 'test successful' and nothing else",
			Timestamp: 1000,
		},
	}

	ctx := context.Background()
	response, err := a.SendMessage(ctx, messages)

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Received response: %s", response)
}
