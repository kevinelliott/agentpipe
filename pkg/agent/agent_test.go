package agent

import (
	"testing"
)

func TestMessageType(t *testing.T) {
	msg := Message{
		AgentID:   "test-agent",
		AgentName: "Test Agent",
		Content:   "Test message",
		Timestamp: 1234567890,
		Role:      "agent",
	}

	if msg.AgentID != "test-agent" {
		t.Errorf("Expected AgentID to be 'test-agent', got %s", msg.AgentID)
	}

	if msg.AgentName != "Test Agent" {
		t.Errorf("Expected AgentName to be 'Test Agent', got %s", msg.AgentName)
	}

	if msg.Content != "Test message" {
		t.Errorf("Expected Content to be 'Test message', got %s", msg.Content)
	}

	if msg.Role != "agent" {
		t.Errorf("Expected Role to be 'agent', got %s", msg.Role)
	}
}

func TestResponseMetrics(t *testing.T) {
	metrics := &ResponseMetrics{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		Model:        "test-model",
		Cost:         0.001,
	}

	if metrics.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens to be 150, got %d", metrics.TotalTokens)
	}

	if metrics.Cost != 0.001 {
		t.Errorf("Expected Cost to be 0.001, got %f", metrics.Cost)
	}
}