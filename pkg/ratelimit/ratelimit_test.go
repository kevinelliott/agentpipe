package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		burst    int
		disabled bool
	}{
		{"normal rate", 10.0, 5, false},
		{"zero rate disables", 0, 5, true},
		{"negative rate disables", -1.0, 5, true},
		{"zero burst clamped to 1", 10.0, 0, false},
		{"negative burst clamped to 1", 10.0, -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewLimiter(tt.rate, tt.burst)

			if limiter.disabled != tt.disabled {
				t.Errorf("expected disabled=%v, got %v", tt.disabled, limiter.disabled)
			}

			if !tt.disabled {
				if limiter.rate != tt.rate {
					t.Errorf("expected rate=%.2f, got %.2f", tt.rate, limiter.rate)
				}

				expectedBurst := tt.burst
				if expectedBurst < 1 {
					expectedBurst = 1
				}
				if limiter.burst != expectedBurst {
					t.Errorf("expected burst=%d, got %d", expectedBurst, limiter.burst)
				}

				// Should start with full bucket
				if limiter.tokens != float64(expectedBurst) {
					t.Errorf("expected initial tokens=%.2f, got %.2f", float64(expectedBurst), limiter.tokens)
				}
			}
		})
	}
}

func TestLimiterDisabled(t *testing.T) {
	limiter := NewLimiter(0, 10)

	// Should always allow when disabled
	for i := 0; i < 100; i++ {
		if !limiter.Allow() {
			t.Error("disabled limiter should always allow requests")
		}
	}

	// Wait should return immediately
	ctx := context.Background()
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("wait on disabled limiter should not error: %v", err)
	}
}

func TestLimiterBurst(t *testing.T) {
	burst := 5
	limiter := NewLimiter(1.0, burst) // 1 req/s with burst of 5

	// Should allow burst number of requests immediately
	for i := 0; i < burst; i++ {
		if !limiter.Allow() {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}

	// Next request should be denied (bucket empty)
	if limiter.Allow() {
		t.Error("request beyond burst should be denied")
	}
}

func TestLimiterRefill(t *testing.T) {
	limiter := NewLimiter(10.0, 1) // 10 req/s, burst 1

	// Take the initial token
	if !limiter.Allow() {
		t.Fatal("first request should be allowed")
	}

	// Should be denied immediately
	if limiter.Allow() {
		t.Error("second request should be denied")
	}

	// Wait for refill (100ms should give us 1 token at 10 req/s)
	time.Sleep(150 * time.Millisecond)

	// Should be allowed after refill
	if !limiter.Allow() {
		t.Error("request after refill should be allowed")
	}
}

func TestLimiterWait(t *testing.T) {
	limiter := NewLimiter(5.0, 1) // 5 req/s, burst 1

	ctx := context.Background()

	// First request should succeed immediately
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("first wait should succeed: %v", err)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Error("first request should not wait")
	}

	// Second request should wait for token refill (~200ms for 1 token at 5 req/s)
	start = time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("second wait should succeed: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 150*time.Millisecond {
		t.Errorf("should wait at least 150ms, waited %v", elapsed)
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("should not wait more than 300ms, waited %v", elapsed)
	}
}

func TestLimiterWaitContext(t *testing.T) {
	limiter := NewLimiter(1.0, 1) // 1 req/s, burst 1

	// Use up the burst
	limiter.Allow()

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Wait should fail with context timeout
	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("expected context timeout error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestLimiterConcurrent(t *testing.T) {
	limiter := NewLimiter(100.0, 10) // 100 req/s, burst 10
	ctx := context.Background()

	// Run many concurrent requests
	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	start := time.Now()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Wait(ctx); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)
	elapsed := time.Since(start)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent wait failed: %v", err)
	}

	// With 100 req/s and 50 requests, should take ~400-500ms
	// (10 immediate from burst, 40 more at 100 req/s = 400ms)
	if elapsed < 300*time.Millisecond {
		t.Errorf("completed too quickly: %v (rate limiting may not be working)", elapsed)
	}
	if elapsed > 700*time.Millisecond {
		t.Errorf("took too long: %v", elapsed)
	}
}

func TestLimiterSetRate(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Verify initial rate
	stats := limiter.GetStats()
	if stats.Rate != 10.0 {
		t.Errorf("expected rate 10.0, got %.2f", stats.Rate)
	}

	// Change rate
	limiter.SetRate(20.0)
	stats = limiter.GetStats()
	if stats.Rate != 20.0 {
		t.Errorf("expected rate 20.0, got %.2f", stats.Rate)
	}

	// Disable by setting to 0
	limiter.SetRate(0)
	stats = limiter.GetStats()
	if !stats.Disabled {
		t.Error("expected limiter to be disabled")
	}
}

func TestLimiterSetBurst(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Verify initial burst
	stats := limiter.GetStats()
	if stats.Burst != 5 {
		t.Errorf("expected burst 5, got %d", stats.Burst)
	}

	// Change burst
	limiter.SetBurst(10)
	stats = limiter.GetStats()
	if stats.Burst != 10 {
		t.Errorf("expected burst 10, got %d", stats.Burst)
	}

	// Test clamping
	limiter.SetBurst(0)
	stats = limiter.GetStats()
	if stats.Burst != 1 {
		t.Errorf("expected burst to be clamped to 1, got %d", stats.Burst)
	}
}

func TestLimiterGetStats(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	stats := limiter.GetStats()
	if stats.Rate != 10.0 {
		t.Errorf("expected rate 10.0, got %.2f", stats.Rate)
	}
	if stats.Burst != 5 {
		t.Errorf("expected burst 5, got %d", stats.Burst)
	}
	if stats.Disabled {
		t.Error("expected limiter to be enabled")
	}

	// Initial tokens should be full burst
	if stats.AvailableTokens < 4.9 || stats.AvailableTokens > 5.1 {
		t.Errorf("expected ~5 available tokens, got %.2f", stats.AvailableTokens)
	}
}

func TestLimiterString(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		burst    int
		contains string
	}{
		{"enabled", 10.0, 5, "10.00 req/s"},
		{"disabled", 0, 5, "disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewLimiter(tt.rate, tt.burst)
			str := limiter.String()

			if !contains(str, tt.contains) {
				t.Errorf("expected string to contain '%s', got '%s'", tt.contains, str)
			}
		})
	}
}

func TestLimiterTokenAccumulation(t *testing.T) {
	limiter := NewLimiter(10.0, 5) // 10 req/s, burst 5

	// Drain all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Verify drained
	stats := limiter.GetStats()
	if stats.AvailableTokens > 0.1 {
		t.Errorf("expected tokens to be drained, got %.2f", stats.AvailableTokens)
	}

	// Wait for accumulation
	time.Sleep(500 * time.Millisecond) // Should accumulate 5 tokens

	stats = limiter.GetStats()
	// Should have accumulated close to 5 tokens (10 req/s * 0.5s = 5 tokens)
	if stats.AvailableTokens < 4.5 || stats.AvailableTokens > 5.1 {
		t.Errorf("expected ~5 tokens after 500ms, got %.2f", stats.AvailableTokens)
	}

	// Wait more - tokens should be capped at burst
	time.Sleep(1 * time.Second)
	stats = limiter.GetStats()
	if stats.AvailableTokens > float64(stats.Burst)+0.1 {
		t.Errorf("tokens should be capped at burst %d, got %.2f", stats.Burst, stats.AvailableTokens)
	}
}

func TestCalculateWaitTime(t *testing.T) {
	limiter := NewLimiter(10.0, 1) // 10 req/s

	// Use the token
	limiter.Allow()

	// Calculate wait time (should be ~100ms for 1 token at 10 req/s)
	waitTime := limiter.calculateWaitTime()

	if waitTime < 90*time.Millisecond || waitTime > 110*time.Millisecond {
		t.Errorf("expected wait time ~100ms, got %v", waitTime)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
