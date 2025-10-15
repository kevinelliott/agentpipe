package integration

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
)

func TestConversationWithFailingAgent(t *testing.T) {
	// Create one failing agent and one working agent
	failingAgent := NewMockIntegrationAgent("failing", "FailingAgent", "mock", func(messages []agent.Message) (string, error) {
		return "", errors.New("persistent failure")
	})

	workingAgent := NewMockIntegrationAgent("working", "WorkingAgent", "mock", func(messages []agent.Message) (string, error) {
		return "I work fine!", nil
	})

	// Create orchestrator with no retries to test error handling
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:              orchestrator.ModeRoundRobin,
		MaxTurns:          2,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        0,
		RetryInitialDelay: 1 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(failingAgent)
	orch.AddAgent(workingAgent)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("orchestrator should continue despite failing agent: %v", err)
	}

	// Working agent should have completed its turns
	if workingAgent.callCount != 2 {
		t.Errorf("working agent should be called 2 times, got %d", workingAgent.callCount)
	}

	// Output should contain error messages
	outputStr := output.String()
	if !strings.Contains(outputStr, "failed") || !strings.Contains(outputStr, "Error") {
		t.Error("output should contain error messages")
	}
}

func TestConversationContextCancellation(t *testing.T) {
	// Create agent with delay
	slowAgent := NewMockIntegrationAgent("slow", "SlowAgent", "mock", nil)
	slowAgent.sendDelay = 100 * time.Millisecond

	// Create orchestrator
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      100, // High number to ensure we don't finish naturally
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(slowAgent)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	err := orch.Start(ctx)

	// Should return context error
	if err == nil {
		t.Error("expected context cancellation error")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got %v", err)
	}

	// Agent should have been called at least once
	if slowAgent.callCount == 0 {
		t.Error("agent should have been called before cancellation")
	}
}

func TestConversationAgentTimeout(t *testing.T) {
	// Create agent that takes longer than timeout
	slowAgent := NewMockIntegrationAgent("timeout", "TimeoutAgent", "mock", nil)
	slowAgent.sendDelay = 500 * time.Millisecond

	// Create orchestrator with short turn timeout
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:              orchestrator.ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       100 * time.Millisecond,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        0,
		RetryInitialDelay: 1 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(slowAgent)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("orchestrator should handle agent timeout gracefully: %v", err)
	}

	// Agent should have been called but timed out
	if slowAgent.callCount != 1 {
		t.Errorf("agent should be called once, got %d", slowAgent.callCount)
	}

	// Check output contains timeout indication
	outputStr := output.String()
	if !strings.Contains(outputStr, "context deadline exceeded") && !strings.Contains(outputStr, "failed") {
		t.Error("output should indicate timeout or failure")
	}
}

func TestConversationNoAgents(t *testing.T) {
	// Create orchestrator without agents
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      5,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)

	// Try to start without agents
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := orch.Start(ctx)

	// Should return error
	if err == nil {
		t.Error("expected error when starting with no agents")
	}

	if !strings.Contains(err.Error(), "no agents") {
		t.Errorf("error should mention no agents, got: %v", err)
	}
}

func TestConversationMaxTurnsLimit(t *testing.T) {
	// Create fast agent
	fastAgent := NewMockIntegrationAgent("fast", "FastAgent", "mock", nil)

	// Create orchestrator with turn limit
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      3,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(fastAgent)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Agent should be called exactly max_turns times
	if fastAgent.callCount != 3 {
		t.Errorf("agent should be called 3 times, got %d", fastAgent.callCount)
	}

	// Output should indicate max turns reached
	outputStr := output.String()
	if !strings.Contains(outputStr, "Maximum turns reached") {
		t.Error("output should indicate max turns reached")
	}
}

func TestConversationRetryExhaustion(t *testing.T) {
	// Create agent that always fails
	alwaysFailAgent := NewMockIntegrationAgent("always-fail", "AlwaysFailAgent", "mock", func(messages []agent.Message) (string, error) {
		return "", errors.New("permanent error")
	})

	// Create orchestrator with limited retries
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:              orchestrator.ModeRoundRobin,
		MaxTurns:          1,
		TurnTimeout:       5 * time.Second,
		ResponseDelay:     10 * time.Millisecond,
		MaxRetries:        2,
		RetryInitialDelay: 50 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RetryMultiplier:   2.0,
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(alwaysFailAgent)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("orchestrator should handle retry exhaustion: %v", err)
	}

	// Agent should be called MaxRetries + 1 times (initial + retries)
	if alwaysFailAgent.callCount != 3 {
		t.Errorf("agent should be called 3 times (1 initial + 2 retries), got %d", alwaysFailAgent.callCount)
	}

	// Output should show retry attempts
	outputStr := output.String()
	if !strings.Contains(outputStr, "Error") {
		t.Error("output should contain error messages")
	}
}

func TestConversationFreeFormMode(t *testing.T) {
	// Create multiple agents
	agents := make([]*MockIntegrationAgent, 3)
	for i := 0; i < 3; i++ {
		idx := i
		agents[i] = NewMockIntegrationAgent(
			"freeform-"+string(rune('1'+i)),
			"FreeAgent"+string(rune('A'+i)),
			"mock",
			func(messages []agent.Message) (string, error) {
				return "Response from agent " + string(rune('A'+idx)), nil
			},
		)
	}

	// Create orchestrator in free-form mode
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeFreeForm,
		MaxTurns:      3,
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

	// In free-form mode, all agents can respond each turn
	// So we should have 3 turns worth of responses
	totalCalls := 0
	for _, a := range agents {
		totalCalls += a.callCount
	}

	// Each turn, all non-last-speaker agents can respond
	// Exact count depends on implementation, but should be >= MaxTurns
	if totalCalls < 3 {
		t.Errorf("expected at least 3 total calls in free-form mode, got %d", totalCalls)
	}
}

func TestConversationWithMetrics(t *testing.T) {
	// Create agent
	agent1 := NewMockIntegrationAgent("metrics", "MetricsAgent", "mock", func(messages []agent.Message) (string, error) {
		return "This is a response with some tokens in it for testing metrics calculation", nil
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

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Get messages and verify metrics are attached
	messages := orch.GetMessages()
	foundMetrics := false
	for _, msg := range messages {
		if msg.Role == "agent" && msg.Metrics != nil {
			foundMetrics = true

			// Verify metrics fields are populated
			if msg.Metrics.Duration == 0 {
				t.Error("duration should be set")
			}
			if msg.Metrics.TotalTokens == 0 {
				t.Error("total tokens should be calculated")
			}
			if msg.Metrics.InputTokens == 0 {
				t.Error("input tokens should be calculated")
			}
			if msg.Metrics.OutputTokens == 0 {
				t.Error("output tokens should be calculated")
			}

			// Verify token math
			if msg.Metrics.TotalTokens != msg.Metrics.InputTokens+msg.Metrics.OutputTokens {
				t.Error("total tokens should equal input + output")
			}
		}
	}

	if !foundMetrics {
		t.Error("agent messages should have metrics attached")
	}

	// Output should contain metrics in display format
	outputStr := output.String()
	if !strings.Contains(outputStr, "ms") || !strings.Contains(outputStr, "t") {
		t.Error("output should contain metrics display (ms for milliseconds, t for tokens)")
	}
}

func TestConversationInitialPrompt(t *testing.T) {
	// Create agent
	agent1 := NewMockIntegrationAgent("initial", "InitialAgent", "mock", nil)

	// Create orchestrator with initial prompt
	var output bytes.Buffer
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      1,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 10 * time.Millisecond,
		InitialPrompt: "This is the starting prompt for our conversation!",
	}
	orch := orchestrator.NewOrchestrator(orchConfig, &output)
	orch.AddAgent(agent1)

	// Run conversation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Start(ctx)
	if err != nil {
		t.Fatalf("conversation failed: %v", err)
	}

	// Verify initial prompt is in message history
	messages := orch.GetMessages()
	foundInitial := false
	for _, msg := range messages {
		if msg.Role == "system" && strings.Contains(msg.Content, "starting prompt") {
			foundInitial = true
			break
		}
	}

	if !foundInitial {
		t.Error("initial prompt should be in message history")
	}

	// Verify it's in output
	outputStr := output.String()
	if !strings.Contains(outputStr, "starting prompt") {
		t.Error("initial prompt should be in output")
	}
}
