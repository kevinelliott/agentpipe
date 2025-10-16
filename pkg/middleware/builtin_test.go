package middleware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// TestLoggingMiddleware tests the logging middleware
func TestLoggingMiddleware(t *testing.T) {
	m := LoggingMiddleware()

	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:        context.Background(),
		AgentID:    "test-agent",
		AgentName:  "TestAgent",
		TurnNumber: 1,
		Metadata:   make(map[string]interface{}),
	}

	msg := &agent.Message{
		Content: "test message",
		Role:    "user",
	}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("LoggingMiddleware failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result from logging middleware")
	}
}

// TestMetricsMiddleware tests the metrics middleware
func TestMetricsMiddleware(t *testing.T) {
	m := MetricsMiddleware()

	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:        context.Background(),
		AgentID:    "test-agent",
		AgentName:  "TestAgent",
		TurnNumber: 1,
		Metadata:   make(map[string]interface{}),
	}

	msg := &agent.Message{
		Content: "test message",
		Role:    "user",
	}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("MetricsMiddleware failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result from metrics middleware")
	}

	// Check metadata was populated
	if _, ok := ctx.Metadata["processing_duration_ms"]; !ok {
		t.Error("Expected processing_duration_ms in metadata")
	}

	if _, ok := ctx.Metadata["input_length"]; !ok {
		t.Error("Expected input_length in metadata")
	}

	if _, ok := ctx.Metadata["output_length"]; !ok {
		t.Error("Expected output_length in metadata")
	}
}

// TestContentFilterMiddleware_MaxLength tests max length filtering
func TestContentFilterMiddleware_MaxLength(t *testing.T) {
	config := ContentFilterMiddlewareConfig{
		MaxLength: 10,
	}

	m := ContentFilterMiddleware(config)
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test message under limit
	msg1 := &agent.Message{Content: "short"}
	_, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for short message: %v", err)
	}

	// Test message over limit
	msg2 := &agent.Message{Content: "this is a very long message"}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for long message")
	}
	if !strings.Contains(err.Error(), "maximum length") {
		t.Errorf("Expected 'maximum length' error, got: %s", err.Error())
	}
}

// TestContentFilterMiddleware_MinLength tests min length filtering
func TestContentFilterMiddleware_MinLength(t *testing.T) {
	config := ContentFilterMiddlewareConfig{
		MinLength: 5,
	}

	m := ContentFilterMiddleware(config)
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test message under minimum
	msg1 := &agent.Message{Content: "hi"}
	_, err := chain.Process(ctx, msg1)
	if err == nil {
		t.Error("Expected error for short message")
	}

	// Test message meeting minimum
	msg2 := &agent.Message{Content: "hello"}
	_, err = chain.Process(ctx, msg2)
	if err != nil {
		t.Errorf("Expected no error for message meeting minimum: %v", err)
	}
}

// TestContentFilterMiddleware_BlockedWords tests blocked word filtering
func TestContentFilterMiddleware_BlockedWords(t *testing.T) {
	config := ContentFilterMiddlewareConfig{
		BlockedWords:  []string{"spam", "bad"},
		CaseSensitive: false,
	}

	m := ContentFilterMiddleware(config)
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test clean message
	msg1 := &agent.Message{Content: "this is a good message"}
	_, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for clean message: %v", err)
	}

	// Test message with blocked word
	msg2 := &agent.Message{Content: "this is spam"}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for message with blocked word")
	}

	// Test case insensitive blocking
	msg3 := &agent.Message{Content: "this is SPAM"}
	_, err = chain.Process(ctx, msg3)
	if err == nil {
		t.Error("Expected error for uppercase blocked word")
	}
}

// TestContentFilterMiddleware_RequiredWords tests required word filtering
func TestContentFilterMiddleware_RequiredWords(t *testing.T) {
	config := ContentFilterMiddlewareConfig{
		RequiredWords: []string{"hello"},
		CaseSensitive: false,
	}

	m := ContentFilterMiddleware(config)
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test message with required word
	msg1 := &agent.Message{Content: "hello world"}
	_, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for message with required word: %v", err)
	}

	// Test message without required word
	msg2 := &agent.Message{Content: "goodbye world"}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for message without required word")
	}
}

// TestSanitizationMiddleware tests message sanitization
func TestSanitizationMiddleware(t *testing.T) {
	tests := []struct {
		name               string
		removeSpecialChars bool
		input              string
		expected           string
	}{
		{
			name:               "trim whitespace",
			removeSpecialChars: false,
			input:              "  hello world  ",
			expected:           "hello world",
		},
		{
			name:               "collapse multiple spaces",
			removeSpecialChars: false,
			input:              "hello    world",
			expected:           "hello world",
		},
		{
			name:               "remove special chars",
			removeSpecialChars: true,
			input:              "hello@world#test",
			expected:           "helloworldtest",
		},
		{
			name:               "keep basic punctuation",
			removeSpecialChars: true,
			input:              "Hello, world! How are you?",
			expected:           "Hello, world! How are you?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := SanitizationMiddleware(tt.removeSpecialChars)
			chain := NewChain(m)
			ctx := &MessageContext{
				Ctx:      context.Background(),
				AgentID:  "test",
				Metadata: make(map[string]interface{}),
			}

			msg := &agent.Message{Content: tt.input}
			result, err := chain.Process(ctx, msg)
			if err != nil {
				t.Fatalf("SanitizationMiddleware failed: %v", err)
			}

			if result.Content != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.Content)
			}
		})
	}
}

// TestRoleValidationMiddleware tests role validation
func TestRoleValidationMiddleware(t *testing.T) {
	allowedRoles := []string{"user", "assistant", "system"}
	m := RoleValidationMiddleware(allowedRoles)

	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test valid role
	msg1 := &agent.Message{
		Content: "test",
		Role:    "user",
	}
	_, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for valid role: %v", err)
	}

	// Test invalid role
	msg2 := &agent.Message{
		Content: "test",
		Role:    "invalid",
	}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for invalid role")
	}
}

// TestEmptyContentValidationMiddleware tests empty content validation
func TestEmptyContentValidationMiddleware(t *testing.T) {
	m := EmptyContentValidationMiddleware()
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test valid content
	msg1 := &agent.Message{Content: "valid content"}
	_, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for valid content: %v", err)
	}

	// Test empty content
	msg2 := &agent.Message{Content: ""}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for empty content")
	}

	// Test whitespace only
	msg3 := &agent.Message{Content: "   "}
	_, err = chain.Process(ctx, msg3)
	if err == nil {
		t.Error("Expected error for whitespace-only content")
	}
}

// TestContextEnrichmentMiddleware tests context enrichment
func TestContextEnrichmentMiddleware(t *testing.T) {
	enricher := func(ctx *MessageContext, msg *agent.Message) {
		ctx.Metadata["enriched"] = true
		ctx.Metadata["timestamp"] = time.Now().Unix()
	}

	m := ContextEnrichmentMiddleware(enricher)
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}
	_, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("ContextEnrichmentMiddleware failed: %v", err)
	}

	if ctx.Metadata["enriched"] != true {
		t.Error("Expected metadata to be enriched")
	}

	if _, ok := ctx.Metadata["timestamp"]; !ok {
		t.Error("Expected timestamp in metadata")
	}
}

// TestRateLimitMiddleware tests rate limiting
func TestRateLimitMiddleware(t *testing.T) {
	config := RateLimitConfig{
		MaxMessagesPerMinute: 2,
		MaxMessagesPerHour:   0, // Disable hour limit for this test
	}

	m := RateLimitMiddleware(config)
	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test-agent",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	// First message should succeed
	_, err := chain.Process(ctx, msg)
	if err != nil {
		t.Errorf("First message should succeed: %v", err)
	}

	// Second message should succeed
	_, err = chain.Process(ctx, msg)
	if err != nil {
		t.Errorf("Second message should succeed: %v", err)
	}

	// Third message should be rate limited
	_, err = chain.Process(ctx, msg)
	if err == nil {
		t.Error("Third message should be rate limited")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("Expected 'rate limit exceeded' error, got: %s", err.Error())
	}
}

// TestMessageHistoryMiddleware tests message history tracking
func TestMessageHistoryMiddleware(t *testing.T) {
	maxHistory := 3
	m := MessageHistoryMiddleware(maxHistory)

	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Process multiple messages
	for i := 1; i <= 5; i++ {
		msg := &agent.Message{Content: "message " + string(rune('0'+i))}
		_, err := chain.Process(ctx, msg)
		if err != nil {
			t.Fatalf("MessageHistoryMiddleware failed: %v", err)
		}
	}

	// Check history
	history, ok := ctx.Metadata["message_history"].([]agent.Message)
	if !ok {
		t.Fatal("Expected message_history in metadata")
	}

	if len(history) != maxHistory {
		t.Errorf("Expected history length %d, got %d", maxHistory, len(history))
	}

	// Should keep only the last 3 messages
	if history[0].Content != "message 3" {
		t.Errorf("Expected first message to be 'message 3', got '%s'", history[0].Content)
	}
}

// TestErrorRecoveryMiddleware tests panic recovery
func TestErrorRecoveryMiddleware(t *testing.T) {
	panicMiddleware := NewMiddlewareFunc("panic", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		panic("test panic")
	})

	recovery := ErrorRecoveryMiddleware()

	chain := NewChain(recovery, panicMiddleware)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	result, err := chain.Process(ctx, msg)
	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	if result != nil {
		t.Error("Expected nil result from panic recovery")
	}

	if !strings.Contains(err.Error(), "middleware panic") {
		t.Errorf("Expected 'middleware panic' error, got: %s", err.Error())
	}
}

// TestBuiltinMiddleware_Integration tests multiple built-in middleware together
func TestBuiltinMiddleware_Integration(t *testing.T) {
	chain := NewChain(
		ErrorRecoveryMiddleware(),
		LoggingMiddleware(),
		MetricsMiddleware(),
		EmptyContentValidationMiddleware(),
		SanitizationMiddleware(false),
		RoleValidationMiddleware([]string{"user", "assistant"}),
	)

	ctx := &MessageContext{
		Ctx:        context.Background(),
		AgentID:    "test-agent",
		AgentName:  "TestAgent",
		TurnNumber: 1,
		Metadata:   make(map[string]interface{}),
	}

	msg := &agent.Message{
		Content: "  hello world  ",
		Role:    "user",
	}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	// Content should be sanitized (trimmed)
	if result.Content != "hello world" {
		t.Errorf("Expected sanitized content, got '%s'", result.Content)
	}

	// Metrics should be present
	if _, ok := ctx.Metadata["processing_duration_ms"]; !ok {
		t.Error("Expected metrics in metadata")
	}
}
