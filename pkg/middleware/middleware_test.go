package middleware

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// TestNewChain tests creating a new middleware chain
func TestNewChain(t *testing.T) {
	chain := NewChain()
	if chain == nil {
		t.Fatal("NewChain should return a non-nil chain")
	}

	if chain.Len() != 0 {
		t.Errorf("Expected empty chain, got length %d", chain.Len())
	}
}

// TestChain_Add tests adding middleware to chain
func TestChain_Add(t *testing.T) {
	chain := NewChain()

	m1 := NewMiddlewareFunc("test1", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		return next(ctx, msg)
	})

	chain.Add(m1)
	if chain.Len() != 1 {
		t.Errorf("Expected chain length 1, got %d", chain.Len())
	}

	m2 := NewMiddlewareFunc("test2", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		return next(ctx, msg)
	})

	chain.Add(m2)
	if chain.Len() != 2 {
		t.Errorf("Expected chain length 2, got %d", chain.Len())
	}
}

// TestChain_Process_EmptyChain tests processing with empty chain
func TestChain_Process_EmptyChain(t *testing.T) {
	chain := NewChain()
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test-agent",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{
		Content: "test message",
		Role:    "user",
	}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Empty chain should not return error: %v", err)
	}

	if result != msg {
		t.Error("Empty chain should return original message")
	}
}

// TestChain_Process_ExecutionOrder tests middleware execution order
func TestChain_Process_ExecutionOrder(t *testing.T) {
	var order []string
	mu := sync.Mutex{}

	m1 := NewMiddlewareFunc("first", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		mu.Lock()
		order = append(order, "first-before")
		mu.Unlock()
		result, err := next(ctx, msg)
		mu.Lock()
		order = append(order, "first-after")
		mu.Unlock()
		return result, err
	})

	m2 := NewMiddlewareFunc("second", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		mu.Lock()
		order = append(order, "second-before")
		mu.Unlock()
		result, err := next(ctx, msg)
		mu.Lock()
		order = append(order, "second-after")
		mu.Unlock()
		return result, err
	})

	m3 := NewMiddlewareFunc("third", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		mu.Lock()
		order = append(order, "third-before")
		mu.Unlock()
		result, err := next(ctx, msg)
		mu.Lock()
		order = append(order, "third-after")
		mu.Unlock()
		return result, err
	})

	chain := NewChain(m1, m2, m3)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	_, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expected := []string{
		"first-before",
		"second-before",
		"third-before",
		"third-after",
		"second-after",
		"first-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("Expected %d execution steps, got %d", len(expected), len(order))
	}

	for i, step := range expected {
		if order[i] != step {
			t.Errorf("Step %d: expected %s, got %s", i, step, order[i])
		}
	}
}

// TestTransformMiddleware tests message transformation
func TestTransformMiddleware(t *testing.T) {
	transform := NewTransformMiddleware("uppercase", func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
		msg.Content = strings.ToUpper(msg.Content)
		return msg, nil
	})

	chain := NewChain(transform)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "hello world"}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Content != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got '%s'", result.Content)
	}
}

// TestTransformMiddleware_Error tests error handling in transform
func TestTransformMiddleware_Error(t *testing.T) {
	transform := NewTransformMiddleware("error", func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
		return nil, errors.New("transform error")
	})

	chain := NewChain(transform)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	_, err := chain.Process(ctx, msg)
	if err == nil {
		t.Error("Expected error from transform")
	}

	if !strings.Contains(err.Error(), "transform error") {
		t.Errorf("Expected 'transform error', got '%s'", err.Error())
	}
}

// TestFilterMiddleware tests message filtering
func TestFilterMiddleware(t *testing.T) {
	filter := NewFilterMiddleware("length-filter", func(ctx *MessageContext, msg *agent.Message) (bool, error) {
		return len(msg.Content) > 5, nil
	})

	chain := NewChain(filter)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test allowed message
	msg1 := &agent.Message{Content: "hello world"}
	result, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for allowed message: %v", err)
	}
	if result == nil {
		t.Error("Expected result for allowed message")
	}

	// Test rejected message
	msg2 := &agent.Message{Content: "hi"}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for rejected message")
	}
	if !strings.Contains(err.Error(), "length-filter") {
		t.Errorf("Expected error to mention filter name, got: %s", err.Error())
	}
}

// TestFilterMiddleware_Error tests filter error handling
func TestFilterMiddleware_Error(t *testing.T) {
	filter := NewFilterMiddleware("error-filter", func(ctx *MessageContext, msg *agent.Message) (bool, error) {
		return false, errors.New("filter error")
	})

	chain := NewChain(filter)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	_, err := chain.Process(ctx, msg)
	if err == nil {
		t.Error("Expected error from filter")
	}

	if !strings.Contains(err.Error(), "filter error") {
		t.Errorf("Expected 'filter error', got '%s'", err.Error())
	}
}

// TestValidationMiddleware tests message validation
func TestValidationMiddleware(t *testing.T) {
	validation := NewValidationMiddleware("content-required", func(ctx *MessageContext, msg *agent.Message) error {
		if msg.Content == "" {
			return errors.New("content is required")
		}
		return nil
	})

	chain := NewChain(validation)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	// Test valid message
	msg1 := &agent.Message{Content: "valid"}
	_, err := chain.Process(ctx, msg1)
	if err != nil {
		t.Errorf("Expected no error for valid message: %v", err)
	}

	// Test invalid message
	msg2 := &agent.Message{Content: ""}
	_, err = chain.Process(ctx, msg2)
	if err == nil {
		t.Error("Expected error for invalid message")
	}
	if !strings.Contains(err.Error(), "content is required") {
		t.Errorf("Expected validation error, got: %s", err.Error())
	}
}

// TestMiddlewareFunc_Name tests middleware name
func TestMiddlewareFunc_Name(t *testing.T) {
	m := NewMiddlewareFunc("test-name", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		return next(ctx, msg)
	})

	if m.Name() != "test-name" {
		t.Errorf("Expected name 'test-name', got '%s'", m.Name())
	}
}

// TestChain_Process_MultipleTransforms tests chaining multiple transforms
func TestChain_Process_MultipleTransforms(t *testing.T) {
	upper := NewTransformMiddleware("uppercase", func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
		msg.Content = strings.ToUpper(msg.Content)
		return msg, nil
	})

	prefix := NewTransformMiddleware("prefix", func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
		msg.Content = "PREFIX: " + msg.Content
		return msg, nil
	})

	chain := NewChain(upper, prefix)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "hello"}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expected := "PREFIX: HELLO"
	if result.Content != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result.Content)
	}
}

// TestChain_Process_ErrorPropagation tests error propagation through chain
func TestChain_Process_ErrorPropagation(t *testing.T) {
	m1 := NewMiddlewareFunc("m1", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		return next(ctx, msg)
	})

	m2 := NewMiddlewareFunc("m2", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		return nil, errors.New("middleware 2 error")
	})

	m3 := NewMiddlewareFunc("m3", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		// This should never be called
		t.Error("m3 should not be called after m2 error")
		return next(ctx, msg)
	})

	chain := NewChain(m1, m2, m3)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	_, err := chain.Process(ctx, msg)
	if err == nil {
		t.Error("Expected error from middleware chain")
	}

	if !strings.Contains(err.Error(), "middleware 2 error") {
		t.Errorf("Expected 'middleware 2 error', got '%s'", err.Error())
	}
}

// TestChain_Process_MetadataAccess tests middleware access to metadata
func TestChain_Process_MetadataAccess(t *testing.T) {
	m1 := NewMiddlewareFunc("metadata-writer", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		ctx.Metadata["processed_by"] = "m1"
		return next(ctx, msg)
	})

	m2 := NewMiddlewareFunc("metadata-reader", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		if val, ok := ctx.Metadata["processed_by"]; !ok || val != "m1" {
			return nil, errors.New("metadata not found")
		}
		ctx.Metadata["read_by"] = "m2"
		return next(ctx, msg)
	})

	chain := NewChain(m1, m2)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	_, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if ctx.Metadata["processed_by"] != "m1" {
		t.Error("Expected metadata 'processed_by' to be set")
	}

	if ctx.Metadata["read_by"] != "m2" {
		t.Error("Expected metadata 'read_by' to be set")
	}
}

// TestChain_Process_ContextCancellation tests context cancellation
func TestChain_Process_ContextCancellation(t *testing.T) {
	m := NewMiddlewareFunc("context-check", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		if err := ctx.Ctx.Err(); err != nil {
			return nil, err
		}
		return next(ctx, msg)
	})

	chain := NewChain(m)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ctx := &MessageContext{
		Ctx:      cancelCtx,
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{Content: "test"}

	_, err := chain.Process(ctx, msg)
	if err == nil {
		t.Error("Expected error from cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestChain_Process_Concurrent tests concurrent middleware execution
func TestChain_Process_Concurrent(t *testing.T) {
	var counter int
	var mu sync.Mutex

	m := NewMiddlewareFunc("counter", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		mu.Lock()
		counter++
		mu.Unlock()
		return next(ctx, msg)
	})

	chain := NewChain(m)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ctx := &MessageContext{
				Ctx:      context.Background(),
				AgentID:  "test",
				Metadata: make(map[string]interface{}),
			}
			msg := &agent.Message{Content: "test"}
			_, _ = chain.Process(ctx, msg)
		}()
	}

	wg.Wait()

	if counter != goroutines {
		t.Errorf("Expected counter %d, got %d", goroutines, counter)
	}
}

// TestChain_Process_MessageModification tests message modification
func TestChain_Process_MessageModification(t *testing.T) {
	m := NewMiddlewareFunc("modifier", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		msg.Content = msg.Content + " modified"
		msg.Role = "assistant"
		return next(ctx, msg)
	})

	chain := NewChain(m)
	ctx := &MessageContext{
		Ctx:      context.Background(),
		AgentID:  "test",
		Metadata: make(map[string]interface{}),
	}

	msg := &agent.Message{
		Content: "original",
		Role:    "user",
	}

	result, err := chain.Process(ctx, msg)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Content != "original modified" {
		t.Errorf("Expected 'original modified', got '%s'", result.Content)
	}

	if result.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", result.Role)
	}
}
