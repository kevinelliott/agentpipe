package utils

import (
	"strings"
	"unicode"
)

// EstimateTokens provides a rough estimation of token count
// This is a simplified version - actual tokenization varies by model
func EstimateTokens(text string) int {
	// Simple estimation: ~1 token per 4 characters or 0.75 words
	// This is very approximate and varies significantly by model
	
	words := strings.Fields(text)
	chars := len(text)
	
	// Use average of word-based and char-based estimation
	wordEstimate := len(words) * 4 / 3  // ~1.33 tokens per word
	charEstimate := chars / 4           // ~4 chars per token
	
	return (wordEstimate + charEstimate) / 2
}

// EstimateCost calculates estimated cost based on model and token count
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	// Pricing per 1M tokens (approximate as of 2024)
	// These are example prices and should be updated based on actual pricing
	
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
		// Default Claude pricing
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
		// Default Gemini pricing
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
	
	// Calculate cost
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