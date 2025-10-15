// Package ratelimit provides token bucket rate limiting for agent requests.
// It implements a simple but effective rate limiting strategy that allows
// burst traffic while maintaining an average rate limit.
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Limiter implements a token bucket rate limiter.
// It is safe for concurrent use.
type Limiter struct {
	mu         sync.Mutex
	rate       float64   // tokens per second
	burst      int       // maximum tokens in bucket
	tokens     float64   // current tokens
	lastRefill time.Time // last time tokens were refilled
	disabled   bool      // if true, limiter always allows requests
}

// NewLimiter creates a new rate limiter with the given rate (requests per second) and burst size.
// Rate of 0 or negative disables rate limiting entirely.
// Burst must be at least 1 if rate limiting is enabled.
func NewLimiter(rate float64, burst int) *Limiter {
	if rate <= 0 {
		return &Limiter{
			disabled: true,
		}
	}

	if burst < 1 {
		burst = 1
	}

	return &Limiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst), // start with full bucket
		lastRefill: time.Now(),
		disabled:   false,
	}
}

// Wait blocks until the rate limiter allows the request or the context is canceled.
// It returns an error if the context is canceled before the request can proceed.
func (l *Limiter) Wait(ctx context.Context) error {
	if l.disabled {
		return nil
	}

	for {
		// Try to take a token
		if l.tryTake() {
			return nil
		}

		// Calculate how long to wait for next token
		waitTime := l.calculateWaitTime()

		// Wait or check context
		select {
		case <-time.After(waitTime):
			// Try again after waiting
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Allow checks if a request can proceed immediately without waiting.
// It returns true if a token is available, false otherwise.
func (l *Limiter) Allow() bool {
	if l.disabled {
		return true
	}

	return l.tryTake()
}

// tryTake attempts to take a token from the bucket.
// It refills the bucket based on elapsed time before attempting to take.
func (l *Limiter) tryTake() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	l.tokens += elapsed * l.rate
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}
	l.lastRefill = now

	// Try to take a token
	if l.tokens >= 1.0 {
		l.tokens -= 1.0
		return true
	}

	return false
}

// calculateWaitTime determines how long to wait for the next token.
func (l *Limiter) calculateWaitTime() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Calculate time needed to accumulate 1 token
	tokensNeeded := 1.0 - l.tokens
	if tokensNeeded <= 0 {
		return time.Millisecond // minimal wait
	}

	seconds := tokensNeeded / l.rate
	return time.Duration(seconds * float64(time.Second))
}

// SetRate updates the rate limit. If rate is 0 or negative, rate limiting is disabled.
// This is useful for dynamic rate limit adjustments.
func (l *Limiter) SetRate(rate float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if rate <= 0 {
		l.disabled = true
		return
	}

	l.disabled = false
	l.rate = rate
	l.lastRefill = time.Now()
}

// SetBurst updates the burst size. Burst must be at least 1.
func (l *Limiter) SetBurst(burst int) {
	if burst < 1 {
		burst = 1
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.burst = burst
	// Clamp current tokens to new burst
	if l.tokens > float64(burst) {
		l.tokens = float64(burst)
	}
}

// Stats returns current rate limiter statistics.
type Stats struct {
	Rate            float64
	Burst           int
	AvailableTokens float64
	Disabled        bool
}

// GetStats returns current statistics about the rate limiter.
func (l *Limiter) GetStats() Stats {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Refill before returning stats
	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	tokens := l.tokens + (elapsed * l.rate)
	if tokens > float64(l.burst) {
		tokens = float64(l.burst)
	}

	return Stats{
		Rate:            l.rate,
		Burst:           l.burst,
		AvailableTokens: tokens,
		Disabled:        l.disabled,
	}
}

// String returns a human-readable representation of the rate limiter configuration.
func (l *Limiter) String() string {
	if l.disabled {
		return "rate limiting disabled"
	}
	return fmt.Sprintf("%.2f req/s, burst=%d", l.rate, l.burst)
}
