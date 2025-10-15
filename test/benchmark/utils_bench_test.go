package benchmark

import (
	"testing"

	"github.com/kevinelliott/agentpipe/pkg/utils"
)

// BenchmarkEstimateTokens benchmarks token estimation for various text lengths
func BenchmarkEstimateTokens(b *testing.B) {
	benchmarks := []struct {
		name string
		text string
	}{
		{"Short", "Hello, world!"},
		{"Medium", "This is a medium-length text that represents a typical agent response with some interesting content that needs to be tokenized."},
		{"Long", "This is a much longer text that simulates a detailed agent response. It contains multiple sentences, various punctuation marks, numbers like 123 and 456, and represents the kind of content you might see in a real conversation between AI agents. The text goes on for quite a while to ensure we're testing performance with realistic message sizes that might occur in production use cases. We want to make sure our token estimation algorithm performs well even with larger inputs."},
		{"VeryLong", longText()},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = utils.EstimateTokens(bm.text)
			}
		})
	}
}

// BenchmarkEstimateCost benchmarks cost estimation
func BenchmarkEstimateCost(b *testing.B) {
	models := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"gpt-4",
		"gpt-3.5-turbo",
		"gemini-pro",
	}

	for _, model := range models {
		b.Run(model, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = utils.EstimateCost(model, 1000, 500)
			}
		})
	}
}

// BenchmarkEstimateTokensParallel benchmarks token estimation with concurrency
func BenchmarkEstimateTokensParallel(b *testing.B) {
	text := "This is a test message that will be tokenized concurrently multiple times to test thread safety and parallel performance."

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = utils.EstimateTokens(text)
		}
	})
}

func longText() string {
	text := ""
	for i := 0; i < 100; i++ {
		text += "This is sentence number " + string(rune('0'+i%10)) + " in a very long conversation history. "
	}
	return text
}
