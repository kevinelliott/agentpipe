package benchmark

import (
	"context"
	"testing"

	"github.com/kevinelliott/agentpipe/pkg/ratelimit"
)

// BenchmarkLimiterAllow benchmarks the Allow method (non-blocking check)
func BenchmarkLimiterAllow(b *testing.B) {
	limiter := ratelimit.NewLimiter(1000.0, 100) // High rate to avoid blocking

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow()
	}
}

// BenchmarkLimiterAllowParallel benchmarks concurrent Allow calls
func BenchmarkLimiterAllowParallel(b *testing.B) {
	limiter := ratelimit.NewLimiter(10000.0, 1000)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = limiter.Allow()
		}
	})
}

// BenchmarkLimiterWait benchmarks the Wait method with available tokens
func BenchmarkLimiterWait(b *testing.B) {
	limiter := ratelimit.NewLimiter(10000.0, 10000) // Very high to avoid actual waiting
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Wait(ctx)
	}
}

// BenchmarkLimiterWaitParallel benchmarks concurrent Wait calls
func BenchmarkLimiterWaitParallel(b *testing.B) {
	limiter := ratelimit.NewLimiter(100000.0, 100000)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = limiter.Wait(ctx)
		}
	})
}

// BenchmarkLimiterGetStats benchmarks statistics retrieval
func BenchmarkLimiterGetStats(b *testing.B) {
	limiter := ratelimit.NewLimiter(100.0, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.GetStats()
	}
}

// BenchmarkLimiterSetRate benchmarks rate updates
func BenchmarkLimiterSetRate(b *testing.B) {
	limiter := ratelimit.NewLimiter(100.0, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.SetRate(100.0 + float64(i%10))
	}
}

// BenchmarkLimiterDisabled benchmarks a disabled limiter (should be very fast)
func BenchmarkLimiterDisabled(b *testing.B) {
	limiter := ratelimit.NewLimiter(0, 0) // Disabled
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Wait(ctx)
	}
}

// BenchmarkNewLimiter benchmarks limiter creation
func BenchmarkNewLimiter(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ratelimit.NewLimiter(100.0, 10)
	}
}
