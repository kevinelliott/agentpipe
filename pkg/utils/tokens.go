package utils

import (
	"strings"
	"unicode"

	"github.com/kevinelliott/agentpipe/internal/providers"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

// EstimateTokens provides a rough estimation of token count
// This is a simplified version - actual tokenization varies by model
func EstimateTokens(text string) int {
	// Simple estimation: ~1 token per 4 characters or 0.75 words
	// This is very approximate and varies significantly by model

	words := strings.Fields(text)
	chars := len(text)

	// Use average of word-based and char-based estimation
	wordEstimate := len(words) * 4 / 3 // ~1.33 tokens per word
	charEstimate := chars / 4          // ~4 chars per token

	return (wordEstimate + charEstimate) / 2
}

// EstimateCost calculates estimated cost based on model and token count.
// It uses the provider registry to lookup accurate pricing from Catwalk's provider configs.
// Falls back to zero cost if the model is not found in the registry.
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	registry := providers.GetRegistry()

	// Try to find the model in the registry
	modelInfo, provider, err := registry.GetModel(model)
	if err != nil {
		// Model not found in registry - log warning and return 0
		log.WithFields(map[string]interface{}{
			"model":         model,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		}).Warn("model not found in provider registry, cost estimate will be $0.00")
		return 0.0
	}

	// Calculate cost using provider pricing
	inputCost := (float64(inputTokens) / 1_000_000) * modelInfo.CostPer1MIn
	outputCost := (float64(outputTokens) / 1_000_000) * modelInfo.CostPer1MOut

	totalCost := inputCost + outputCost

	log.WithFields(map[string]interface{}{
		"model":         model,
		"provider":      provider.Name,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"input_cost":    inputCost,
		"output_cost":   outputCost,
		"total_cost":    totalCost,
	}).Debug("calculated cost estimate")

	return totalCost
}

// EstimateCostLegacy is the old hardcoded cost estimation function.
// Deprecated: Use EstimateCost which uses the provider registry instead.
func EstimateCostLegacy(model string, inputTokens, outputTokens int) float64 {
	var inputPricePerMillion, outputPricePerMillion float64

	modelLower := strings.ToLower(model)

	// Claude models
	if strings.Contains(modelLower, "claude-3-opus") {
		inputPricePerMillion = 15.00
		outputPricePerMillion = 75.00
	} else if strings.Contains(modelLower, "claude-3-sonnet") {
		inputPricePerMillion = 3.00
		outputPricePerMillion = 15.00
	} else if strings.Contains(modelLower, "claude-3-haiku") {
		inputPricePerMillion = 0.25
		outputPricePerMillion = 1.25
	} else if strings.Contains(modelLower, "claude-2") {
		inputPricePerMillion = 8.00
		outputPricePerMillion = 24.00
	} else if strings.Contains(modelLower, "claude") {
		inputPricePerMillion = 3.00
		outputPricePerMillion = 15.00
	}

	// Gemini models
	if strings.Contains(modelLower, "gemini-pro") {
		inputPricePerMillion = 0.50
		outputPricePerMillion = 1.50
	} else if strings.Contains(modelLower, "gemini-ultra") {
		inputPricePerMillion = 7.00
		outputPricePerMillion = 21.00
	} else if strings.Contains(modelLower, "gemini") {
		inputPricePerMillion = 0.50
		outputPricePerMillion = 1.50
	}

	// GPT models
	if strings.Contains(modelLower, "gpt-4-turbo") {
		inputPricePerMillion = 10.00
		outputPricePerMillion = 30.00
	} else if strings.Contains(modelLower, "gpt-4") {
		inputPricePerMillion = 30.00
		outputPricePerMillion = 60.00
	} else if strings.Contains(modelLower, "gpt-3.5-turbo") {
		inputPricePerMillion = 0.50
		outputPricePerMillion = 1.50
	}

	inputCost := (float64(inputTokens) / 1_000_000) * inputPricePerMillion
	outputCost := (float64(outputTokens) / 1_000_000) * outputPricePerMillion

	return inputCost + outputCost
}

// CountWords returns the number of words in a string
func CountWords(text string) int {
	count := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}

	return count
}
