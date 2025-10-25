package utils

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
		delta    int // allowed difference
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
			delta:    0,
		},
		{
			name:     "single word",
			text:     "hello",
			expected: 1,
			delta:    1,
		},
		{
			name:     "short sentence",
			text:     "The quick brown fox jumps over the lazy dog",
			expected: 9,
			delta:    3,
		},
		{
			name:     "with punctuation",
			text:     "Hello, world! How are you today?",
			expected: 8,
			delta:    2,
		},
		{
			name:     "with numbers",
			text:     "The year 2024 has 365 days",
			expected: 8,
			delta:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.delta {
				t.Errorf("EstimateTokens() = %v, want %v ± %v", got, tt.expected, tt.delta)
			}
		})
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantCost     float64
		delta        float64
	}{
		{
			name:         "claude-sonnet-4-5",
			model:        "claude-sonnet-4-5-20250929",
			inputTokens:  1000000, // 1M tokens
			outputTokens: 1000000, // 1M tokens
			wantCost:     18.00,   // $3 in + $15 out per 1M = $18
			delta:        0.01,
		},
		{
			name:         "claude-3-5-haiku",
			model:        "claude-3-5-haiku-20241022",
			inputTokens:  1000000, // 1M tokens
			outputTokens: 1000000, // 1M tokens
			wantCost:     4.80,    // $0.80 in + $4 out per 1M = $4.80
			delta:        0.01,
		},
		{
			name:         "gpt-5",
			model:        "gpt-5",
			inputTokens:  1000000, // 1M tokens
			outputTokens: 1000000, // 1M tokens
			wantCost:     11.25,   // $1.25 in + $10 out per 1M = $11.25
			delta:        0.01,
		},
		{
			name:         "unknown model",
			model:        "completely-unknown-model-xyz",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0, // unknown model returns 0
			delta:        0.0001,
		},
		{
			name:         "zero tokens",
			model:        "claude-sonnet-4-5-20250929",
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0,
			delta:        0,
		},
		{
			name:         "small token count",
			model:        "claude-sonnet-4-5-20250929",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.0105, // (1000/1M * $3) + (500/1M * $15) = 0.003 + 0.0075 = 0.0105
			delta:        0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCost(tt.model, tt.inputTokens, tt.outputTokens)
			diff := got - tt.wantCost
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.delta {
				t.Errorf("EstimateCost() = %v, want %v ± %v", got, tt.wantCost, tt.delta)
			}
		})
	}
}

func TestEstimateCostLegacy(t *testing.T) {
	// Test the legacy function to ensure it still works
	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantCost     float64
		delta        float64
	}{
		{
			name:         "claude-3-opus",
			model:        "claude-3-opus",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.0525, // (1000/1M * $15) + (500/1M * $75) = 0.015 + 0.0375 = 0.0525
			delta:        0.0001,
		},
		{
			name:         "gpt-4",
			model:        "gpt-4",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.06, // (1000/1M * $30) + (500/1M * $60) = 0.03 + 0.03 = 0.06
			delta:        0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCostLegacy(tt.model, tt.inputTokens, tt.outputTokens)
			diff := got - tt.wantCost
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.delta {
				t.Errorf("EstimateCostLegacy() = %v, want %v ± %v", got, tt.wantCost, tt.delta)
			}
		})
	}
}
