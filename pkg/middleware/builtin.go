package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

// LoggingMiddleware creates middleware that logs all messages.
// It logs before and after processing with structured fields.
func LoggingMiddleware() Middleware {
	return NewMiddlewareFunc("logging", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		start := time.Now()

		log.WithFields(map[string]interface{}{
			"agent_id":    ctx.AgentID,
			"agent_name":  ctx.AgentName,
			"turn_number": ctx.TurnNumber,
			"role":        msg.Role,
			"content_len": len(msg.Content),
		}).Debug("processing message")

		result, err := next(ctx, msg)

		duration := time.Since(start)

		if err != nil {
			log.WithFields(map[string]interface{}{
				"agent_id":    ctx.AgentID,
				"agent_name":  ctx.AgentName,
				"turn_number": ctx.TurnNumber,
				"duration_ms": duration.Milliseconds(),
			}).WithError(err).Error("message processing failed")
			return nil, err
		}

		log.WithFields(map[string]interface{}{
			"agent_id":    ctx.AgentID,
			"agent_name":  ctx.AgentName,
			"turn_number": ctx.TurnNumber,
			"duration_ms": duration.Milliseconds(),
		}).Debug("message processed successfully")

		return result, nil
	})
}

// MetricsMiddleware creates middleware that tracks message processing metrics.
// It stores metrics in the message context metadata.
func MetricsMiddleware() Middleware {
	return NewMiddlewareFunc("metrics", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		start := time.Now()

		result, err := next(ctx, msg)

		duration := time.Since(start)

		// Store metrics in metadata
		if ctx.Metadata == nil {
			ctx.Metadata = make(map[string]interface{})
		}

		ctx.Metadata["processing_duration_ms"] = duration.Milliseconds()
		ctx.Metadata["input_length"] = len(msg.Content)

		if result != nil {
			ctx.Metadata["output_length"] = len(result.Content)
		}

		return result, err
	})
}

// ContentFilterMiddlewareConfig configures content filtering.
type ContentFilterMiddlewareConfig struct {
	// MaxLength is the maximum allowed message length (0 = unlimited)
	MaxLength int

	// MinLength is the minimum required message length (0 = no minimum)
	MinLength int

	// BlockedWords is a list of words that are not allowed
	BlockedWords []string

	// RequiredWords is a list of words that must be present
	RequiredWords []string

	// CaseSensitive determines if word matching is case-sensitive
	CaseSensitive bool
}

// ContentFilterMiddleware creates middleware that filters messages based on content rules.
func ContentFilterMiddleware(config ContentFilterMiddlewareConfig) Middleware {
	return NewFilterMiddleware("content-filter", func(ctx *MessageContext, msg *agent.Message) (bool, error) {
		content := msg.Content

		// Check length constraints
		if config.MaxLength > 0 && len(content) > config.MaxLength {
			return false, fmt.Errorf("message exceeds maximum length of %d characters", config.MaxLength)
		}

		if config.MinLength > 0 && len(content) < config.MinLength {
			return false, fmt.Errorf("message is below minimum length of %d characters", config.MinLength)
		}

		// Prepare content for word matching
		checkContent := content
		if !config.CaseSensitive {
			checkContent = strings.ToLower(checkContent)
		}

		// Check for blocked words
		for _, word := range config.BlockedWords {
			checkWord := word
			if !config.CaseSensitive {
				checkWord = strings.ToLower(checkWord)
			}

			if strings.Contains(checkContent, checkWord) {
				return false, fmt.Errorf("message contains blocked word: %s", word)
			}
		}

		// Check for required words
		for _, word := range config.RequiredWords {
			checkWord := word
			if !config.CaseSensitive {
				checkWord = strings.ToLower(checkWord)
			}

			if !strings.Contains(checkContent, checkWord) {
				return false, fmt.Errorf("message missing required word: %s", word)
			}
		}

		return true, nil
	})
}

// SanitizationMiddleware creates middleware that sanitizes message content.
// It trims whitespace and optionally removes special characters.
func SanitizationMiddleware(removeSpecialChars bool) Middleware {
	return NewTransformMiddleware("sanitization", func(ctx *MessageContext, msg *agent.Message) (*agent.Message, error) {
		// Trim whitespace
		msg.Content = strings.TrimSpace(msg.Content)

		// Remove special characters if configured
		if removeSpecialChars {
			// Keep only alphanumeric, spaces, and basic punctuation
			var result strings.Builder
			for _, r := range msg.Content {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
					(r >= '0' && r <= '9') || r == ' ' || r == '.' ||
					r == ',' || r == '!' || r == '?' || r == '-' || r == '_' {
					result.WriteRune(r)
				}
			}
			msg.Content = result.String()
		}

		// Collapse multiple spaces
		msg.Content = strings.Join(strings.Fields(msg.Content), " ")

		return msg, nil
	})
}

// RoleValidationMiddleware creates middleware that validates message roles.
// It ensures messages have valid roles from the allowed list.
func RoleValidationMiddleware(allowedRoles []string) Middleware {
	return NewValidationMiddleware("role-validation", func(ctx *MessageContext, msg *agent.Message) error {
		if len(allowedRoles) == 0 {
			return nil // No validation if no roles specified
		}

		for _, role := range allowedRoles {
			if msg.Role == role {
				return nil
			}
		}

		return fmt.Errorf("invalid message role '%s', allowed roles: %v", msg.Role, allowedRoles)
	})
}

// EmptyContentValidationMiddleware creates middleware that rejects empty messages.
func EmptyContentValidationMiddleware() Middleware {
	return NewValidationMiddleware("empty-content", func(ctx *MessageContext, msg *agent.Message) error {
		if strings.TrimSpace(msg.Content) == "" {
			return fmt.Errorf("message content cannot be empty")
		}
		return nil
	})
}

// ContextEnrichmentMiddleware creates middleware that enriches the message context.
// It adds additional metadata fields to the context.
func ContextEnrichmentMiddleware(enricher func(*MessageContext, *agent.Message)) Middleware {
	return NewMiddlewareFunc("context-enrichment", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		if ctx.Metadata == nil {
			ctx.Metadata = make(map[string]interface{})
		}

		enricher(ctx, msg)

		return next(ctx, msg)
	})
}

// RateLimitMiddleware creates middleware that enforces rate limiting.
// It tracks message counts per agent and rejects if limits are exceeded.
type RateLimitConfig struct {
	MaxMessagesPerMinute int
	MaxMessagesPerHour   int
}

// RateLimitMiddleware creates middleware that enforces basic rate limiting.
// Note: This is a simple in-memory implementation. For production use,
// consider using the pkg/ratelimit package with persistent storage.
func RateLimitMiddleware(config RateLimitConfig) Middleware {
	type rateLimitState struct {
		minuteCount int
		hourCount   int
		minuteReset time.Time
		hourReset   time.Time
	}

	// In-memory state per agent
	states := make(map[string]*rateLimitState)

	return NewFilterMiddleware("rate-limit", func(ctx *MessageContext, msg *agent.Message) (bool, error) {
		now := time.Now()

		// Get or create state for this agent
		state, exists := states[ctx.AgentID]
		if !exists {
			state = &rateLimitState{
				minuteReset: now.Add(time.Minute),
				hourReset:   now.Add(time.Hour),
			}
			states[ctx.AgentID] = state
		}

		// Reset counters if time windows expired
		if now.After(state.minuteReset) {
			state.minuteCount = 0
			state.minuteReset = now.Add(time.Minute)
		}
		if now.After(state.hourReset) {
			state.hourCount = 0
			state.hourReset = now.Add(time.Hour)
		}

		// Check limits
		if config.MaxMessagesPerMinute > 0 && state.minuteCount >= config.MaxMessagesPerMinute {
			return false, fmt.Errorf("rate limit exceeded: %d messages per minute", config.MaxMessagesPerMinute)
		}
		if config.MaxMessagesPerHour > 0 && state.hourCount >= config.MaxMessagesPerHour {
			return false, fmt.Errorf("rate limit exceeded: %d messages per hour", config.MaxMessagesPerHour)
		}

		// Increment counters
		state.minuteCount++
		state.hourCount++

		return true, nil
	})
}

// MessageHistoryMiddleware creates middleware that tracks message history.
// It stores recent messages in the context metadata for pattern analysis.
func MessageHistoryMiddleware(maxHistory int) Middleware {
	history := make([]agent.Message, 0, maxHistory)

	return NewMiddlewareFunc("message-history", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (*agent.Message, error) {
		// Add to history
		history = append(history, *msg)
		if len(history) > maxHistory {
			history = history[1:] // Remove oldest
		}

		// Store in metadata
		if ctx.Metadata == nil {
			ctx.Metadata = make(map[string]interface{})
		}
		ctx.Metadata["message_history"] = history

		return next(ctx, msg)
	})
}

// ErrorRecoveryMiddleware creates middleware that recovers from panics.
// It catches panics in downstream middleware and converts them to errors.
func ErrorRecoveryMiddleware() Middleware {
	return NewMiddlewareFunc("error-recovery", func(ctx *MessageContext, msg *agent.Message, next ProcessFunc) (result *agent.Message, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(map[string]interface{}{
					"agent_id":   ctx.AgentID,
					"panic":      r,
					"agent_name": ctx.AgentName,
				}).Error("middleware panic recovered")
				err = fmt.Errorf("middleware panic: %v", r)
				result = nil
			}
		}()

		return next(ctx, msg)
	})
}
