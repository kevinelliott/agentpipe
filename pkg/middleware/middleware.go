// Package middleware provides a flexible middleware pattern for message processing.
// Middleware can intercept, transform, validate, and augment messages as they flow
// through the orchestrator.
package middleware

import (
	"context"
	"fmt"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

// MessageContext contains contextual information for middleware processing.
type MessageContext struct {
	// Ctx is the request context
	Ctx context.Context

	// AgentID is the ID of the agent processing the message
	AgentID string

	// AgentName is the name of the agent
	AgentName string

	// TurnNumber is the current turn in the conversation
	TurnNumber int

	// Metadata contains additional context information
	Metadata map[string]interface{}
}

// Middleware processes messages in a chain.
// It can modify the message, add metadata, or stop processing by returning an error.
type Middleware interface {
	// Process handles a message and optionally passes it to the next middleware.
	// Returns the processed message and any error.
	Process(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error)

	// Name returns the middleware name for logging and debugging.
	Name() string
}

// ProcessFunc is a function that processes a message.
// It's used to chain middleware together.
type ProcessFunc func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error)

// Chain represents a chain of middleware.
type Chain struct {
	middleware []Middleware
}

// NewChain creates a new middleware chain.
func NewChain(middleware ...Middleware) *Chain {
	return &Chain{
		middleware: middleware,
	}
}

// Add appends middleware to the chain.
func (c *Chain) Add(m Middleware) {
	c.middleware = append(c.middleware, m)
}

// Process executes the middleware chain for a message.
func (c *Chain) Process(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
	if len(c.middleware) == 0 {
		return msg, nil
	}

	// Build the chain from the end
	var process ProcessFunc
	process = func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
		return msg, nil
	}

	// Wrap each middleware in reverse order
	for i := len(c.middleware) - 1; i >= 0; i-- {
		m := c.middleware[i]
		next := process
		process = func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
			return m.Process(ctx, msg, next)
		}
	}

	return process(ctx, msg)
}

// Len returns the number of middleware in the chain.
func (c *Chain) Len() int {
	return len(c.middleware)
}

// MiddlewareFunc is a function adapter for the Middleware interface.
type MiddlewareFunc struct {
	name string
	fn   func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error)
}

// NewMiddlewareFunc creates a middleware from a function.
func NewMiddlewareFunc(name string, fn func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error)) Middleware {
	return &MiddlewareFunc{
		name: name,
		fn:   fn,
	}
}

// Process implements Middleware.
func (m *MiddlewareFunc) Process(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
	return m.fn(ctx, msg, next)
}

// Name implements Middleware.
func (m *MiddlewareFunc) Name() string {
	return m.name
}

// FilterFunc is a simple filter function that can reject messages.
// Return true to allow the message, false to reject it.
type FilterFunc func(ctx *MessageContext, msg *agent.Message) (bool, error)

// NewFilterMiddleware creates middleware from a filter function.
// If the filter returns false, processing stops with an error.
func NewFilterMiddleware(name string, filter FilterFunc) Middleware {
	return NewMiddlewareFunc(name, func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		allowed, err := filter(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("filter error: %w", err)
		}
		if !allowed {
			log.WithFields(map[string]interface{}{
				"middleware":  name,
				"agent_id":    ctx.AgentID,
				"turn_number": ctx.TurnNumber,
			}).Warn("message filtered by middleware")
			return nil, fmt.Errorf("message rejected by %s filter", name)
		}
		return next(ctx, msg)
	})
}

// TransformFunc transforms a message.
type TransformFunc func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error)

// NewTransformMiddleware creates middleware from a transform function.
func NewTransformMiddleware(name string, transform TransformFunc) Middleware {
	return NewMiddlewareFunc(name, func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		transformed, err := transform(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("transform error: %w", err)
		}
		return next(ctx, transformed)
	})
}

// ValidationFunc validates a message.
// Return an error if the message is invalid.
type ValidationFunc func(ctx *MessageContext, msg *agent.Message) error

// NewValidationMiddleware creates middleware from a validation function.
func NewValidationMiddleware(name string, validate ValidationFunc) Middleware {
	return NewMiddlewareFunc(name, func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		if err := validate(ctx, msg); err != nil {
			log.WithFields(map[string]interface{}{
				"middleware":  name,
				"agent_id":    ctx.AgentID,
				"turn_number": ctx.TurnNumber,
			}).WithError(err).Error("message validation failed")
			return nil, fmt.Errorf("validation failed in %s: %w", name, err)
		}
		return next(ctx, msg)
	})
}
