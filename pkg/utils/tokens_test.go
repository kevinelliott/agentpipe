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
			name:         "claude-3-opus",
			model:        "claude-3-opus",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.0525, // Current implementation returns this
			delta:        0.0001,
		},
		{
			name:         "gpt-4",
			model:        "gpt-4",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0.06, // Current implementation returns this
			delta:        0.0001,
		},
		{
			name:         "unknown model",
			model:        "unknown",
			inputTokens:  1000,
			outputTokens: 500,
			wantCost:     0, // unknown model returns 0
			delta:        0.0000001,
		},
		{
			name:         "zero tokens",
			model:        "claude-3-opus",
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0,
			delta:        0,
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
