package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestNewMetrics tests creating a new metrics instance
func TestNewMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	if m == nil {
		t.Fatal("NewMetrics should return non-nil")
	}

	if m.AgentRequests == nil {
		t.Error("AgentRequests should be initialized")
	}
	if m.AgentRequestDuration == nil {
		t.Error("AgentRequestDuration should be initialized")
	}
	if m.AgentTokens == nil {
		t.Error("AgentTokens should be initialized")
	}
	if m.AgentCost == nil {
		t.Error("AgentCost should be initialized")
	}
	if m.AgentErrors == nil {
		t.Error("AgentErrors should be initialized")
	}
	if m.ActiveConversations == nil {
		t.Error("ActiveConversations should be initialized")
	}
	if m.ConversationTurns == nil {
		t.Error("ConversationTurns should be initialized")
	}
	if m.MessageSize == nil {
		t.Error("MessageSize should be initialized")
	}
	if m.RetryAttempts == nil {
		t.Error("RetryAttempts should be initialized")
	}
	if m.RateLimitHits == nil {
		t.Error("RateLimitHits should be initialized")
	}
}

// TestNewMetrics_NilRegistry tests creating metrics with nil registry
func TestNewMetrics_NilRegistry(t *testing.T) {
	m := NewMetrics(nil)
	if m == nil {
		t.Fatal("NewMetrics should handle nil registry")
	}
}

// TestRecordAgentRequest tests recording agent requests
func TestRecordAgentRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordAgentRequest("Claude", "claude", "success")
	m.RecordAgentRequest("Claude", "claude", "success")
	m.RecordAgentRequest("Claude", "claude", "error")

	// Check counter values
	successCount := testutil.ToFloat64(m.AgentRequests.WithLabelValues("Claude", "claude", "success"))
	if successCount != 2 {
		t.Errorf("Expected 2 success requests, got %f", successCount)
	}

	errorCount := testutil.ToFloat64(m.AgentRequests.WithLabelValues("Claude", "claude", "error"))
	if errorCount != 1 {
		t.Errorf("Expected 1 error request, got %f", errorCount)
	}
}

// TestRecordAgentDuration tests recording agent duration
func TestRecordAgentDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordAgentDuration("Claude", "claude", 1.5)
	m.RecordAgentDuration("Claude", "claude", 0.5)
	m.RecordAgentDuration("Gemini", "gemini", 2.0)

	// For histograms, we just verify no panic occurred
	// In production, histogram metrics are scraped and analyzed by Prometheus
	// We can't easily test histogram values in unit tests
}

// TestRecordAgentTokens tests recording token counts
func TestRecordAgentTokens(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordAgentTokens("Claude", "claude", "input", 100)
	m.RecordAgentTokens("Claude", "claude", "input", 50)
	m.RecordAgentTokens("Claude", "claude", "output", 200)

	inputTokens := testutil.ToFloat64(m.AgentTokens.WithLabelValues("Claude", "claude", "input"))
	if inputTokens != 150 {
		t.Errorf("Expected 150 input tokens, got %f", inputTokens)
	}

	outputTokens := testutil.ToFloat64(m.AgentTokens.WithLabelValues("Claude", "claude", "output"))
	if outputTokens != 200 {
		t.Errorf("Expected 200 output tokens, got %f", outputTokens)
	}
}

// TestRecordAgentCost tests recording costs
func TestRecordAgentCost(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordAgentCost("Claude", "claude", "claude-sonnet-4.5", 0.001)
	m.RecordAgentCost("Claude", "claude", "claude-sonnet-4.5", 0.002)
	m.RecordAgentCost("Gemini", "gemini", "gemini-pro", 0.0005)

	claudeCost := testutil.ToFloat64(m.AgentCost.WithLabelValues("Claude", "claude", "claude-sonnet-4.5"))
	if claudeCost != 0.003 {
		t.Errorf("Expected 0.003 cost, got %f", claudeCost)
	}

	geminiCost := testutil.ToFloat64(m.AgentCost.WithLabelValues("Gemini", "gemini", "gemini-pro"))
	if geminiCost != 0.0005 {
		t.Errorf("Expected 0.0005 cost, got %f", geminiCost)
	}
}

// TestRecordAgentError tests recording errors
func TestRecordAgentError(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordAgentError("Claude", "claude", "timeout")
	m.RecordAgentError("Claude", "claude", "timeout")
	m.RecordAgentError("Claude", "claude", "rate_limit")

	timeoutCount := testutil.ToFloat64(m.AgentErrors.WithLabelValues("Claude", "claude", "timeout"))
	if timeoutCount != 2 {
		t.Errorf("Expected 2 timeout errors, got %f", timeoutCount)
	}

	rateLimitCount := testutil.ToFloat64(m.AgentErrors.WithLabelValues("Claude", "claude", "rate_limit"))
	if rateLimitCount != 1 {
		t.Errorf("Expected 1 rate_limit error, got %f", rateLimitCount)
	}
}

// TestActiveConversations tests the active conversations gauge
func TestActiveConversations(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	// Start at 0
	count := testutil.ToFloat64(m.ActiveConversations)
	if count != 0 {
		t.Errorf("Expected 0 active conversations, got %f", count)
	}

	// Increment
	m.IncrementActiveConversations()
	m.IncrementActiveConversations()
	count = testutil.ToFloat64(m.ActiveConversations)
	if count != 2 {
		t.Errorf("Expected 2 active conversations, got %f", count)
	}

	// Decrement
	m.DecrementActiveConversations()
	count = testutil.ToFloat64(m.ActiveConversations)
	if count != 1 {
		t.Errorf("Expected 1 active conversation, got %f", count)
	}
}

// TestRecordConversationTurn tests recording conversation turns
func TestRecordConversationTurn(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordConversationTurn("round-robin")
	m.RecordConversationTurn("round-robin")
	m.RecordConversationTurn("reactive")

	roundRobinCount := testutil.ToFloat64(m.ConversationTurns.WithLabelValues("round-robin"))
	if roundRobinCount != 2 {
		t.Errorf("Expected 2 round-robin turns, got %f", roundRobinCount)
	}

	reactiveCount := testutil.ToFloat64(m.ConversationTurns.WithLabelValues("reactive"))
	if reactiveCount != 1 {
		t.Errorf("Expected 1 reactive turn, got %f", reactiveCount)
	}
}

// TestRecordMessageSize tests recording message sizes
func TestRecordMessageSize(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordMessageSize("Claude", "input", 1000)
	m.RecordMessageSize("Claude", "input", 2000)
	m.RecordMessageSize("Claude", "output", 500)

	// For histograms, we just verify no panic occurred
	// In production, histogram metrics are scraped and analyzed by Prometheus
	// We can't easily test histogram values in unit tests
}

// TestRecordRetryAttempt tests recording retry attempts
func TestRecordRetryAttempt(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordRetryAttempt("Claude", "claude")
	m.RecordRetryAttempt("Claude", "claude")
	m.RecordRetryAttempt("Gemini", "gemini")

	claudeRetries := testutil.ToFloat64(m.RetryAttempts.WithLabelValues("Claude", "claude"))
	if claudeRetries != 2 {
		t.Errorf("Expected 2 Claude retries, got %f", claudeRetries)
	}

	geminiRetries := testutil.ToFloat64(m.RetryAttempts.WithLabelValues("Gemini", "gemini"))
	if geminiRetries != 1 {
		t.Errorf("Expected 1 Gemini retry, got %f", geminiRetries)
	}
}

// TestRecordRateLimitHit tests recording rate limit hits
func TestRecordRateLimitHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	m.RecordRateLimitHit("Claude")
	m.RecordRateLimitHit("Claude")

	hits := testutil.ToFloat64(m.RateLimitHits.WithLabelValues("Claude"))
	if hits != 2 {
		t.Errorf("Expected 2 rate limit hits, got %f", hits)
	}
}

// TestReset tests resetting all metrics
func TestReset(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	// Record some data
	m.RecordAgentRequest("Claude", "claude", "success")
	m.RecordAgentTokens("Claude", "claude", "input", 100)
	m.IncrementActiveConversations()
	m.RecordConversationTurn("round-robin")

	// Verify data exists
	if testutil.ToFloat64(m.AgentRequests.WithLabelValues("Claude", "claude", "success")) != 1 {
		t.Error("Expected metrics before reset")
	}

	// Reset
	m.Reset()

	// Verify all metrics are reset
	if testutil.ToFloat64(m.AgentRequests.WithLabelValues("Claude", "claude", "success")) != 0 {
		t.Error("AgentRequests should be reset")
	}
	if testutil.ToFloat64(m.AgentTokens.WithLabelValues("Claude", "claude", "input")) != 0 {
		t.Error("AgentTokens should be reset")
	}
	if testutil.ToFloat64(m.ActiveConversations) != 0 {
		t.Error("ActiveConversations should be reset")
	}
	if testutil.ToFloat64(m.ConversationTurns.WithLabelValues("round-robin")) != 0 {
		t.Error("ConversationTurns should be reset")
	}
}

// TestDefaultMetrics tests the default global metrics instance
func TestDefaultMetrics(t *testing.T) {
	if DefaultMetrics == nil {
		t.Fatal("DefaultMetrics should be initialized")
	}

	if DefaultRegistry == nil {
		t.Fatal("DefaultRegistry should be initialized")
	}

	// Test that we can record to default metrics
	DefaultMetrics.RecordAgentRequest("test", "test", "success")
	count := testutil.ToFloat64(DefaultMetrics.AgentRequests.WithLabelValues("test", "test", "success"))
	if count == 0 {
		t.Error("Expected to record to default metrics")
	}

	// Clean up
	DefaultMetrics.Reset()
}

// TestMetrics_MultipleAgents tests metrics for multiple agents
func TestMetrics_MultipleAgents(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	agents := []struct {
		name      string
		agentType string
	}{
		{"Claude", "claude"},
		{"Gemini", "gemini"},
		{"Qwen", "qwen"},
	}

	for _, agent := range agents {
		m.RecordAgentRequest(agent.name, agent.agentType, "success")
		m.RecordAgentDuration(agent.name, agent.agentType, 1.0)
		m.RecordAgentTokens(agent.name, agent.agentType, "input", 100)
		m.RecordAgentTokens(agent.name, agent.agentType, "output", 50)
	}

	// Verify each agent has metrics
	for _, agent := range agents {
		count := testutil.ToFloat64(m.AgentRequests.WithLabelValues(agent.name, agent.agentType, "success"))
		if count != 1 {
			t.Errorf("Expected 1 request for %s, got %f", agent.name, count)
		}

		inputTokens := testutil.ToFloat64(m.AgentTokens.WithLabelValues(agent.name, agent.agentType, "input"))
		if inputTokens != 100 {
			t.Errorf("Expected 100 input tokens for %s, got %f", agent.name, inputTokens)
		}
	}
}

// TestMetrics_LargeValues tests metrics with large values
func TestMetrics_LargeValues(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	// Record large values
	m.RecordAgentTokens("Claude", "claude", "input", 1000000)
	m.RecordAgentCost("Claude", "claude", "claude-sonnet-4.5", 100.50)
	m.RecordMessageSize("Claude", "input", 500000)

	tokens := testutil.ToFloat64(m.AgentTokens.WithLabelValues("Claude", "claude", "input"))
	if tokens != 1000000 {
		t.Errorf("Expected 1000000 tokens, got %f", tokens)
	}

	cost := testutil.ToFloat64(m.AgentCost.WithLabelValues("Claude", "claude", "claude-sonnet-4.5"))
	if cost != 100.50 {
		t.Errorf("Expected 100.50 cost, got %f", cost)
	}
}

// TestMetrics_ConcurrentAccess tests concurrent metric recording
func TestMetrics_ConcurrentAccess(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewMetrics(registry)

	// Record metrics concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.RecordAgentRequest("Claude", "claude", "success")
				m.RecordAgentTokens("Claude", "claude", "input", 1)
				m.IncrementActiveConversations()
				m.DecrementActiveConversations()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts
	requests := testutil.ToFloat64(m.AgentRequests.WithLabelValues("Claude", "claude", "success"))
	if requests != 1000 {
		t.Errorf("Expected 1000 requests, got %f", requests)
	}

	tokens := testutil.ToFloat64(m.AgentTokens.WithLabelValues("Claude", "claude", "input"))
	if tokens != 1000 {
		t.Errorf("Expected 1000 tokens, got %f", tokens)
	}
}
