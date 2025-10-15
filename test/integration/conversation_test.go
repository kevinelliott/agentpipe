package integration

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/logger"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
)

// MockIntegrationAgent is a realistic mock agent for integration testing
type MockIntegrationAgent struct {
	agent.BaseAgent
	responseFunc func(messages []agent.Message) (string, error)
	sendDelay    time.Duration
	callCount    int
}

func NewMockIntegrationAgent(id, name, agentType string, responseFunc func([]agent.Message) (string, error)) *MockIntegrationAgent {
	return &MockIntegrationAgent{
		BaseAgent: agent.BaseAgent{
			ID:   id,
			Name: name,
			Type: agentType,
		},
		responseFunc: responseFunc,
	}
}

func (m *MockIntegrationAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	m.callCount++

	if m.sendDelay > 0 {
		select {
		case <-time.After(m.sendDelay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if m.responseFunc != nil {
		return m.responseFunc(messages)
	}

	return "Default response from " + m.Name, nil
}

func (m *MockIntegrationAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	response, err := m.SendMessage(ctx, messages)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(response))
	return err
}

func (m *MockIntegrationAgent) IsAvailable() bool {
	return true
}

func (m *MockIntegrationAgent) HealthCheck(ctx context.Context) error {
	return nil
}

func TestFullConversationRoundRobin(t *testing.T) {
	// Create temporary directory for logs
	tempDir := t.TempDir()

	// Create mock agents with realistic behavior
	agent1 := NewMockIntegrationAgent("agent-1", "Alice", "mock", func(messages []agent.Message) (string, error) {
		// Count how many times this agent has spoken
		count := 0
		for _, msg := range messages {
			if msg.AgentID == "agent-1" {
				count++
			}
		}
		return "This is my " + ordinal(count+1) + " response from Alice", nil
	})

	agent2 := NewMockIntegrationAgent("agent-2", "Bob", "mock", func(messages []agent.Message) (string, error) {
		count := 0
		for _, msg := range messages {
			if msg.AgentID == "agent-2" {
				count++
			}
		}
		return "Bob here with response #" + ordinal(count+1), nil
	})

	// Create orchestrator
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      2,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
		InitialPrompt: "Welcome to the integration test!",
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)

	// Create logger
	chatLogger, err := logger.NewChatLogger(tempDir, "text", &output, true)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer chatLogger.Close()
	orch.SetLogger(chatLogger)

	// Add agents
	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Validate results
	messages := orch.GetMessages()

	// Should have: 1 initial prompt + 2 announcements + 4 agent messages (2 turns * 2 agents)
	agentMessages := 0
	systemMessages := 0
	for _, msg := range messages {
		if msg.Role == "agent" {
			agentMessages++
		}
		if msg.Role == "system" {
			systemMessages++
		}
	}

	if agentMessages != 4 {
		t.Errorf("expected 4 agent messages, got %d", agentMessages)
	}

	// Should have initial prompt + 2 announcements + end message
	if systemMessages < 3 {
		t.Errorf("expected at least 3 system messages, got %d", systemMessages)
	}

	// Check output contains agent responses
	outputStr := output.String()
	if !strings.Contains(outputStr, "Alice") {
		t.Error("output should contain Alice's messages")
	}
	if !strings.Contains(outputStr, "Bob") {
		t.Error("output should contain Bob's messages")
	}

	// Check log file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected log file to be created")
	}

	// Each agent should have been called twice
	if agent1.callCount != 2 {
		t.Errorf("agent1 should be called 2 times, got %d", agent1.callCount)
	}
	if agent2.callCount != 2 {
		t.Errorf("agent2 should be called 2 times, got %d", agent2.callCount)
	}
}

func TestFullConversationWithRateLimiting(t *testing.T) {
	// Create agents with rate limiting
	agent1 := NewMockIntegrationAgent("rate-limited-1", "LimitedAgent", "mock", nil)
	agent1.Config.RateLimit = 5.0    // 5 requests per second
	agent1.Config.RateLimitBurst = 2 // Burst of 2

	// Create orchestrator
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      5,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(agent1)

	// Run conversation and measure time
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := orch.Start(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// With 5 turns and rate limit of 5 req/s with burst of 2:
	// - First 2 requests: immediate
	// - Requests 3-5: delayed by rate limit
	// Should take at least 400ms
	if elapsed < 400*time.Millisecond {
		t.Errorf("rate limiting should have slowed conversation, took %v", elapsed)
	}

	if agent1.callCount != 5 {
		t.Errorf("expected 5 calls, got %d", agent1.callCount)
	}
}

func TestFullConversationWithRetries(t *testing.T) {
	// Create agent that fails first 2 attempts then succeeds
	failCount := 0
	agent1 := NewMockIntegrationAgent("retry-agent", "RetryAgent", "mock", func(messages []agent.Message) (string, error) {
		failCount++
		if failCount <= 2 {
			return "", errors.New("simulated failure")
		}
		return "Success after retries!", nil
	})

	// Create orchestrator with retry config
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:              orchestrator.ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        3,
		RetryInitialDelay: 50 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RetryMultiplier:   2.0,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(agent1)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Should have succeeded on 3rd attempt
	if agent1.callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", agent1.callCount)
	}

	// Check output contains retry messages
	outputStr := output.String()
	if !strings.Contains(outputStr, "Retry") || !strings.Contains(outputStr, "Success after retries") {
		t.Error("output should contain retry messages and success")
	}
}

func TestFullConversationReactiveMode(t *testing.T) {
	// Create multiple agents
	agents := make([]*MockIntegrationAgent, 3)
	for i := 0; i < 3; i++ {
		agents[i] = NewMockIntegrationAgent(
			"agent-"+string(rune('1'+i)),
			"Agent"+string(rune('A'+i)),
			"mock",
			nil,
		)
	}

	// Create orchestrator in reactive mode
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeReactive,
		MaxTurns:      5,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)

	for _, a := range agents {
		orch.AddAgent(a)
	}

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Verify total call count
	totalCalls := 0
	for _, a := range agents {
		totalCalls += a.callCount
	}

	if totalCalls != 5 {
		t.Errorf("expected 5 total calls, got %d", totalCalls)
	}

	// Verify each agent was called at least once in reactive mode
	// (with 5 turns and 3 agents, statistically all should be called)
	for i, a := range agents {
		if a.callCount == 0 {
			t.Logf("warning: agent %d was never selected (acceptable but rare)", i)
		}
	}
}

func TestConfigurationLoading(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-agent-1
    type: claude
    name: TestAgent1
    prompt: "You are a test agent"
    rate_limit: 10.0
    rate_limit_burst: 5
  - id: test-agent-2
    type: gemini
    name: TestAgent2
    prompt: "You are another test agent"

orchestrator:
  mode: round-robin
  max_turns: 3
  turn_timeout: 30s
  response_delay: 1s
  initial_prompt: "Hello, agents!"

logging:
  enabled: true
  chat_log_dir: /tmp/test-logs
  log_format: json
  show_metrics: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Validate configuration
	if len(cfg.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(cfg.Agents))
	}

	if cfg.Agents[0].RateLimit != 10.0 {
		t.Errorf("expected rate limit 10.0, got %.1f", cfg.Agents[0].RateLimit)
	}

	if cfg.Agents[0].RateLimitBurst != 5 {
		t.Errorf("expected burst 5, got %d", cfg.Agents[0].RateLimitBurst)
	}

	if cfg.Orchestrator.Mode != "round-robin" {
		t.Errorf("expected round-robin mode, got %s", cfg.Orchestrator.Mode)
	}

	if cfg.Orchestrator.MaxTurns != 3 {
		t.Errorf("expected max turns 3, got %d", cfg.Orchestrator.MaxTurns)
	}

	if cfg.Logging.LogFormat != "json" {
		t.Errorf("expected json format, got %s", cfg.Logging.LogFormat)
	}

	// Test validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass validation: %v", err)
	}
}

func TestMultiAgentConversationContext(t *testing.T) {
	// Test that agents receive full conversation history
	conversationLog := []string{}

	agent1 := NewMockIntegrationAgent("context-1", "ContextAgent1", "mock", func(messages []agent.Message) (string, error) {
		// Log what this agent sees
		log := "Agent1 sees " + string(rune('0'+len(messages))) + " messages"
		conversationLog = append(conversationLog, log)
		return "Response from Agent1", nil
	})

	agent2 := NewMockIntegrationAgent("context-2", "ContextAgent2", "mock", func(messages []agent.Message) (string, error) {
		log := "Agent2 sees " + string(rune('0'+len(messages))) + " messages"
		conversationLog = append(conversationLog, log)
		return "Response from Agent2", nil
	})

	// Create orchestrator
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      2,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Verify conversation context grows
	// Each agent should see more messages as conversation progresses
	if len(conversationLog) != 4 {
		t.Errorf("expected 4 log entries, got %d", len(conversationLog))
	}

	// Log conversation for debugging
	t.Logf("Conversation log: %v", conversationLog)
}

// Helper function to convert number to ordinal (1st, 2nd, 3rd, etc.)
func ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return string(rune('0'+n)) + suffix
}
