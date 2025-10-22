package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/internal/bridge"
	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// MockAgent is a test double for agent.Agent
type MockAgent struct {
	id              string
	name            string
	agentType       string
	model           string
	rateLimit       float64
	rateLimitBurst  int
	available       bool
	healthCheckErr  error
	sendMessageResp string
	sendMessageErr  error
	sendDelay       time.Duration
	callCount       int
	// For retry testing: fail first N attempts
	failFirstN int
	failCount  int
}

func (m *MockAgent) GetID() string          { return m.id }
func (m *MockAgent) GetName() string        { return m.name }
func (m *MockAgent) GetType() string        { return m.agentType }
func (m *MockAgent) GetModel() string       { return m.model }
func (m *MockAgent) GetRateLimit() float64  { return m.rateLimit }
func (m *MockAgent) GetRateLimitBurst() int { return m.rateLimitBurst }
func (m *MockAgent) IsAvailable() bool      { return m.available }
func (m *MockAgent) Announce() string       { return m.name + " has joined" }
func (m *MockAgent) GetCLIVersion() string  { return "1.0.0" }
func (m *MockAgent) GetPrompt() string      { return "You are a helpful assistant" }
func (m *MockAgent) Initialize(config agent.AgentConfig) error {
	m.id = config.ID
	m.name = config.Name
	m.agentType = config.Type
	m.model = config.Model
	return nil
}

func (m *MockAgent) HealthCheck(ctx context.Context) error {
	return m.healthCheckErr
}

func (m *MockAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	m.callCount++
	if m.sendDelay > 0 {
		select {
		case <-time.After(m.sendDelay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Support conditional failures for retry testing
	if m.failFirstN > 0 {
		m.failCount++
		if m.failCount <= m.failFirstN {
			return "", errors.New("simulated failure")
		}
	}

	if m.sendMessageErr != nil {
		return "", m.sendMessageErr
	}
	return m.sendMessageResp, nil
}

func (m *MockAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	_, err := writer.Write([]byte(m.sendMessageResp))
	return err
}

// MockBridgeEmitter is a test double for bridge.Emitter
type MockBridgeEmitter struct {
	conversationStartedCalled   bool
	conversationCompletedCalled bool
	completedStatus             string
	messageCreatedCount         int
	errorCalled                 bool
}

func (m *MockBridgeEmitter) GetConversationID() string {
	return "test-conv-123"
}

func (m *MockBridgeEmitter) EmitConversationStarted(mode string, initialPrompt string, maxTurns int, agents []bridge.AgentParticipant, commandInfo *bridge.CommandInfo) {
	m.conversationStartedCalled = true
}

func (m *MockBridgeEmitter) EmitMessageCreated(agentID, agentType, agentName, content, model string, turnNumber, tokensUsed, inputTokens, outputTokens int, cost float64, duration time.Duration) {
	m.messageCreatedCount++
}

func (m *MockBridgeEmitter) EmitConversationCompleted(status string, totalMessages, totalTurns, totalTokens int, totalCost float64, duration time.Duration, summary *bridge.SummaryMetadata) {
	m.conversationCompletedCalled = true
	m.completedStatus = status
}

func (m *MockBridgeEmitter) EmitConversationError(errorMessage, errorType, agentType string) {
	m.errorCalled = true
}

func (m *MockBridgeEmitter) Close() error {
	return nil
}

func TestNewOrchestrator(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		TurnTimeout:   10 * time.Second,
		MaxTurns:      5,
		ResponseDelay: 1 * time.Second,
	}

	orch := NewOrchestrator(config, nil)

	if orch == nil {
		t.Fatal("expected orchestrator to be created")
	}
	if orch.config.Mode != ModeRoundRobin {
		t.Errorf("expected mode %s, got %s", ModeRoundRobin, orch.config.Mode)
	}
	if orch.config.TurnTimeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", orch.config.TurnTimeout)
	}
}

func TestNewOrchestratorDefaults(t *testing.T) {
	config := OrchestratorConfig{
		Mode: ModeRoundRobin,
	}

	orch := NewOrchestrator(config, nil)

	if orch.config.TurnTimeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", orch.config.TurnTimeout)
	}
	if orch.config.ResponseDelay != 1*time.Second {
		t.Errorf("expected default delay 1s, got %v", orch.config.ResponseDelay)
	}
}

func TestAddAgent(t *testing.T) {
	config := OrchestratorConfig{
		Mode: ModeRoundRobin,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	mockAgent := &MockAgent{
		id:        "test-1",
		name:      "TestAgent",
		agentType: "mock",
		available: true,
	}

	orch.AddAgent(mockAgent)

	messages := orch.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Errorf("expected system message, got %s", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "TestAgent") {
		t.Errorf("expected announcement to contain agent name")
	}
}

func TestRoundRobinMode(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		MaxTurns:      2,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	agent1 := &MockAgent{
		id:              "agent-1",
		name:            "Agent1",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response from Agent1",
	}
	agent2 := &MockAgent{
		id:              "agent-2",
		name:            "Agent2",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response from Agent2",
	}

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: 2 announcements + 4 agent responses (2 turns * 2 agents)
	messages := orch.GetMessages()
	agentMessages := 0
	for _, msg := range messages {
		if msg.Role == "agent" {
			agentMessages++
		}
	}

	if agentMessages != 4 {
		t.Errorf("expected 4 agent messages, got %d", agentMessages)
	}

	// Each agent should be called twice (2 turns)
	if agent1.callCount != 2 {
		t.Errorf("expected agent1 to be called 2 times, got %d", agent1.callCount)
	}
	if agent2.callCount != 2 {
		t.Errorf("expected agent2 to be called 2 times, got %d", agent2.callCount)
	}
}

func TestReactiveMode(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeReactive,
		MaxTurns:      3,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	agent1 := &MockAgent{
		id:              "agent-1",
		name:            "Agent1",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response from Agent1",
	}
	agent2 := &MockAgent{
		id:              "agent-2",
		name:            "Agent2",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response from Agent2",
	}

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := orch.GetMessages()
	agentMessages := 0
	for _, msg := range messages {
		if msg.Role == "agent" {
			agentMessages++
		}
	}

	// Should have 3 agent messages (max turns = 3)
	if agentMessages != 3 {
		t.Errorf("expected 3 agent messages, got %d", agentMessages)
	}
}

func TestContextCancellation(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		MaxTurns:      100, // High number to ensure we don't finish naturally
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 50 * time.Millisecond,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	mockAgent := &MockAgent{
		id:              "agent-1",
		name:            "Agent1",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response",
	}

	orch.AddAgent(mockAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := orch.Start(ctx)

	// Should return context error
	if err == nil {
		t.Error("expected context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got %v", err)
	}
}

func TestAgentTimeout(t *testing.T) {
	config := OrchestratorConfig{
		Mode:              ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       100 * time.Millisecond,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        0,                    // Disable retries for this test
		RetryInitialDelay: 1 * time.Millisecond, // Must set to indicate retry config is explicit
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	slowAgent := &MockAgent{
		id:              "slow-agent",
		name:            "SlowAgent",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response",
		sendDelay:       500 * time.Millisecond, // Longer than timeout
	}

	orch.AddAgent(slowAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected orchestrator error: %v", err)
	}

	// Agent should have been called once but timed out
	if slowAgent.callCount != 1 {
		t.Errorf("expected agent to be called 1 time, got %d", slowAgent.callCount)
	}
}

func TestNoAgentsConfigured(t *testing.T) {
	config := OrchestratorConfig{
		Mode: ModeRoundRobin,
	}
	orch := NewOrchestrator(config, nil)

	ctx := context.Background()
	err := orch.Start(ctx)

	if err == nil {
		t.Error("expected error for no agents, got nil")
	}
	if !strings.Contains(err.Error(), "no agents") {
		t.Errorf("expected 'no agents' error, got: %v", err)
	}
}

func TestInitialPrompt(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		MaxTurns:      1,
		InitialPrompt: "Hello, let's discuss testing!",
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	mockAgent := &MockAgent{
		id:              "agent-1",
		name:            "Agent1",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Sure!",
	}

	orch.AddAgent(mockAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := orch.GetMessages()
	foundInitialPrompt := false
	for _, msg := range messages {
		if msg.Role == "system" && strings.Contains(msg.Content, "Hello, let's discuss testing!") {
			foundInitialPrompt = true
			break
		}
	}

	if !foundInitialPrompt {
		t.Error("initial prompt not found in messages")
	}
}

func TestAgentError(t *testing.T) {
	config := OrchestratorConfig{
		Mode:              ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        0,                    // Disable retries for this test
		RetryInitialDelay: 1 * time.Millisecond, // Must set to indicate retry config is explicit
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	failingAgent := &MockAgent{
		id:             "failing-agent",
		name:           "FailingAgent",
		agentType:      "mock",
		available:      true,
		sendMessageErr: errors.New("simulated error"),
	}

	workingAgent := &MockAgent{
		id:              "working-agent",
		name:            "WorkingAgent",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "I'm working fine",
	}

	orch.AddAgent(failingAgent)
	orch.AddAgent(workingAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected orchestrator error: %v", err)
	}

	// Orchestrator should continue despite failing agent
	if workingAgent.callCount != 1 {
		t.Errorf("expected working agent to be called, got %d calls", workingAgent.callCount)
	}

	// Check that error was written to output
	output := buf.String()
	if !strings.Contains(output, "failed") && !strings.Contains(output, "Error") {
		t.Error("expected error message in output")
	}
}

func TestSelectNextAgent(t *testing.T) {
	config := OrchestratorConfig{Mode: ModeReactive}
	orch := NewOrchestrator(config, nil)

	agent1 := &MockAgent{id: "agent-1", name: "Agent1"}
	agent2 := &MockAgent{id: "agent-2", name: "Agent2"}
	agent3 := &MockAgent{id: "agent-3", name: "Agent3"}

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)
	orch.AddAgent(agent3)

	// Test excluding last speaker
	selected := orch.selectNextAgent("agent-1")
	if selected == nil {
		t.Fatal("expected agent to be selected")
	}
	if selected.GetID() == "agent-1" {
		t.Error("selected agent should not be the last speaker")
	}

	// Test with no exclusion
	selected = orch.selectNextAgent("")
	if selected == nil {
		t.Fatal("expected agent to be selected")
	}

	// Test when all agents are excluded (should return nil)
	orch2 := NewOrchestrator(config, nil)
	orch2.AddAgent(agent1)
	selected = orch2.selectNextAgent("agent-1")
	if selected != nil {
		t.Error("expected nil when all agents excluded")
	}
}

func TestRetrySuccessAfterFailures(t *testing.T) {
	config := OrchestratorConfig{
		Mode:              ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        3,
		RetryInitialDelay: 50 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RetryMultiplier:   2.0,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	// Agent that fails twice then succeeds
	mockAgent := &MockAgent{
		id:              "retry-agent",
		name:            "RetryAgent",
		agentType:       "mock",
		available:       true,
		failFirstN:      2,
		sendMessageResp: "Success after retries",
	}

	orch.AddAgent(mockAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have succeeded on 3rd attempt
	if mockAgent.callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", mockAgent.callCount)
	}

	// Should have 1 agent message (success)
	messages := orch.GetMessages()
	agentMessages := 0
	for _, msg := range messages {
		if msg.Role == "agent" {
			agentMessages++
			if !strings.Contains(msg.Content, "Success after retries") {
				t.Error("expected success message in conversation")
			}
		}
	}

	if agentMessages != 1 {
		t.Errorf("expected 1 agent message, got %d", agentMessages)
	}

	// Check output contains retry messages
	output := buf.String()
	if !strings.Contains(output, "Retry") && !strings.Contains(output, "attempt") {
		t.Error("expected retry messages in output")
	}
}

func TestRetryExhaustion(t *testing.T) {
	config := OrchestratorConfig{
		Mode:              ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        2,
		RetryInitialDelay: 50 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RetryMultiplier:   2.0,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	// Agent that always fails
	failingAgent := &MockAgent{
		id:             "failing-agent",
		name:           "FailingAgent",
		agentType:      "mock",
		available:      true,
		sendMessageErr: errors.New("persistent failure"),
	}

	orch.AddAgent(failingAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected orchestrator error: %v", err)
	}

	// Should have tried MaxRetries + 1 times (initial + 2 retries)
	if failingAgent.callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", failingAgent.callCount)
	}

	// Should have no agent messages (all failed)
	messages := orch.GetMessages()
	agentMessages := 0
	for _, msg := range messages {
		if msg.Role == "agent" {
			agentMessages++
		}
	}

	if agentMessages != 0 {
		t.Errorf("expected 0 agent messages, got %d", agentMessages)
	}

	// Check output contains error and retry messages
	output := buf.String()
	if !strings.Contains(output, "Error") {
		t.Error("expected error message in output")
	}
}

func TestCalculateBackoffDelay(t *testing.T) {
	config := OrchestratorConfig{
		Mode:              ModeRoundRobin,
		MaxRetries:        5,
		RetryInitialDelay: 1 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		RetryMultiplier:   2.0,
	}
	orch := NewOrchestrator(config, nil)

	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
		description string
	}{
		{1, 2 * time.Second, 2 * time.Second, "first retry: 1s * 2^1 = 2s"},
		{2, 4 * time.Second, 4 * time.Second, "second retry: 1s * 2^2 = 4s"},
		{3, 8 * time.Second, 8 * time.Second, "third retry: 1s * 2^3 = 8s"},
		{4, 16 * time.Second, 16 * time.Second, "fourth retry: 1s * 2^4 = 16s"},
		{5, 30 * time.Second, 30 * time.Second, "fifth retry: capped at max 30s"},
		{10, 30 * time.Second, 30 * time.Second, "large retry: capped at max 30s"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			delay := orch.calculateBackoffDelay(tt.attempt)

			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("attempt %d: expected delay between %v and %v, got %v",
					tt.attempt, tt.expectedMin, tt.expectedMax, delay)
			}
		})
	}
}

func TestRetryWithCustomConfig(t *testing.T) {
	config := OrchestratorConfig{
		Mode:              ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        1,
		RetryInitialDelay: 100 * time.Millisecond,
		RetryMaxDelay:     1 * time.Second,
		RetryMultiplier:   3.0,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	// Agent fails once, then succeeds
	mockAgent := &MockAgent{
		id:              "custom-retry-agent",
		name:            "CustomRetryAgent",
		agentType:       "mock",
		available:       true,
		failFirstN:      1,
		sendMessageResp: "Success on retry",
	}

	orch.AddAgent(mockAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockAgent.callCount != 2 {
		t.Errorf("expected 2 attempts, got %d", mockAgent.callCount)
	}

	messages := orch.GetMessages()
	agentMessages := 0
	for _, msg := range messages {
		if msg.Role == "agent" {
			agentMessages++
		}
	}

	if agentMessages != 1 {
		t.Errorf("expected 1 agent message after retry, got %d", agentMessages)
	}
}

func TestRetryDefaults(t *testing.T) {
	config := OrchestratorConfig{
		Mode: ModeRoundRobin,
		// Don't set retry configs - should use defaults
	}
	orch := NewOrchestrator(config, nil)

	// Check defaults were applied
	if orch.config.MaxRetries != 3 {
		t.Errorf("expected default MaxRetries=3, got %d", orch.config.MaxRetries)
	}
	if orch.config.RetryInitialDelay != 1*time.Second {
		t.Errorf("expected default RetryInitialDelay=1s, got %v", orch.config.RetryInitialDelay)
	}
	if orch.config.RetryMaxDelay != 30*time.Second {
		t.Errorf("expected default RetryMaxDelay=30s, got %v", orch.config.RetryMaxDelay)
	}
	if orch.config.RetryMultiplier != 2.0 {
		t.Errorf("expected default RetryMultiplier=2.0, got %v", orch.config.RetryMultiplier)
	}
}

func TestRateLimitingCreation(t *testing.T) {
	config := OrchestratorConfig{
		Mode: ModeRoundRobin,
	}
	orch := NewOrchestrator(config, nil)

	mockAgent := &MockAgent{
		id:             "rate-limited-agent",
		name:           "RateLimitedAgent",
		agentType:      "mock",
		available:      true,
		rateLimit:      10.0, // 10 requests per second
		rateLimitBurst: 5,
	}

	orch.AddAgent(mockAgent)

	// Verify rate limiter was created
	orch.mu.RLock()
	limiter := orch.rateLimiters[mockAgent.GetID()]
	orch.mu.RUnlock()

	if limiter == nil {
		t.Fatal("expected rate limiter to be created for agent")
	}

	// Verify rate limiter has correct configuration
	stats := limiter.GetStats()
	if stats.Rate != 10.0 {
		t.Errorf("expected rate 10.0, got %.2f", stats.Rate)
	}
	if stats.Burst != 5 {
		t.Errorf("expected burst 5, got %d", stats.Burst)
	}
}

func TestRateLimitingEnforcement(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		MaxTurns:      5,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	// Agent with tight rate limit: 5 req/s, burst 2
	mockAgent := &MockAgent{
		id:              "rate-limited-agent",
		name:            "RateLimitedAgent",
		agentType:       "mock",
		available:       true,
		rateLimit:       5.0, // 5 requests per second
		rateLimitBurst:  2,
		sendMessageResp: "Response",
	}

	orch.AddAgent(mockAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := orch.Start(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 5 turns and rate limit of 5 req/s with burst of 2:
	// - First 2 requests: immediate (from burst)
	// - Requests 3-5: need to wait for token refill
	// - At 5 req/s, each token takes 200ms
	// - So 3 more requests need ~600ms minimum
	// Total should be at least 400ms (accounting for burst and response delays)
	if elapsed < 400*time.Millisecond {
		t.Errorf("expected rate limiting to slow down requests, took only %v", elapsed)
	}

	// Verify all turns completed
	if mockAgent.callCount != 5 {
		t.Errorf("expected 5 calls, got %d", mockAgent.callCount)
	}
}

func TestRateLimitingUnlimited(t *testing.T) {
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		MaxTurns:      3,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	// Agent with no rate limit (0 = unlimited)
	mockAgent := &MockAgent{
		id:              "unlimited-agent",
		name:            "UnlimitedAgent",
		agentType:       "mock",
		available:       true,
		rateLimit:       0, // Unlimited
		rateLimitBurst:  0,
		sendMessageResp: "Response",
	}

	orch.AddAgent(mockAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := orch.Start(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should complete quickly without rate limiting
	// 3 turns * 10ms response delay = ~30ms + overhead
	if elapsed > 200*time.Millisecond {
		t.Errorf("unlimited rate limit took too long: %v", elapsed)
	}

	if mockAgent.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mockAgent.callCount)
	}
}

func TestBridgeEventOnCancellation(t *testing.T) {
	// Track received events
	var receivedEvents []bridge.Event
	var mu sync.Mutex

	// Create mock HTTP server to capture bridge events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event bridge.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// Create bridge config pointing to mock server
	bridgeConfig := &bridge.Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "test-key",
		TimeoutMs:     5000,
		RetryAttempts: 0,
		LogLevel:      "debug",
	}

	// Create orchestrator config
	config := OrchestratorConfig{
		Mode:          ModeRoundRobin,
		MaxTurns:      100, // High number to ensure we don't finish naturally
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 50 * time.Millisecond,
	}
	var buf bytes.Buffer
	orch := NewOrchestrator(config, &buf)

	mockAgent := &MockAgent{
		id:              "agent-1",
		name:            "Agent1",
		agentType:       "mock",
		available:       true,
		sendMessageResp: "Response",
	}

	orch.AddAgent(mockAgent)

	// Create real bridge emitter
	emitter := bridge.NewEmitter(bridgeConfig, "0.3.7-test")
	orch.SetBridgeEmitter(emitter)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := orch.Start(ctx)

	// Should return context error
	if err == nil {
		t.Error("expected context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got %v", err)
	}

	// No need to sleep - conversation.completed is sent synchronously before Start() returns

	// Verify we received events
	mu.Lock()
	eventCount := len(receivedEvents)
	mu.Unlock()

	if eventCount == 0 {
		t.Fatal("expected to receive bridge events, got none")
	}

	// Find the conversation.completed event
	mu.Lock()
	var completedEvent *bridge.Event
	for i := range receivedEvents {
		if receivedEvents[i].Type == bridge.EventConversationCompleted {
			completedEvent = &receivedEvents[i]
			break
		}
	}
	mu.Unlock()

	if completedEvent == nil {
		t.Fatal("expected to receive conversation.completed event")
	}

	// Verify the status is "interrupted"
	completedData, ok := completedEvent.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected conversation.completed data to be a map")
	}

	status, ok := completedData["status"].(string)
	if !ok {
		t.Fatal("expected status to be a string")
	}

	if status != "interrupted" {
		t.Errorf("expected completed status to be 'interrupted', got '%s'", status)
	}
}
