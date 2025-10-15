package orchestrator

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// MockAgent is a test double for agent.Agent
type MockAgent struct {
	id              string
	name            string
	agentType       string
	model           string
	available       bool
	healthCheckErr  error
	sendMessageResp string
	sendMessageErr  error
	sendDelay       time.Duration
	callCount       int
}

func (m *MockAgent) GetID() string     { return m.id }
func (m *MockAgent) GetName() string   { return m.name }
func (m *MockAgent) GetType() string   { return m.agentType }
func (m *MockAgent) GetModel() string  { return m.model }
func (m *MockAgent) IsAvailable() bool { return m.available }
func (m *MockAgent) Announce() string  { return m.name + " has joined" }
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
	if m.sendMessageErr != nil {
		return "", m.sendMessageErr
	}
	return m.sendMessageResp, nil
}

func (m *MockAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	_, err := writer.Write([]byte(m.sendMessageResp))
	return err
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
		Mode:          ModeRoundRobin,
		MaxTurns:      1,
		TurnTimeout:   100 * time.Millisecond,
		ResponseDelay: 10 * time.Millisecond,
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
		Mode:          ModeRoundRobin,
		MaxTurns:      1,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
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
