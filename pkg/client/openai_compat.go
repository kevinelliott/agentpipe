// Package client provides HTTP clients for API-based AI providers.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
)

// OpenAICompatClient is an HTTP client for OpenAI-compatible APIs.
// It supports both streaming and non-streaming requests.
type OpenAICompatClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

// NewOpenAICompatClient creates a new OpenAI-compatible API client.
func NewOpenAICompatClient(baseURL, apiKey string) *OpenAICompatClient {
	return &OpenAICompatClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		maxRetries: 3,
	}
}

// ChatCompletionRequest represents a request to the chat completions endpoint.
type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	Temperature *float64                `json:"temperature,omitempty"`
	MaxTokens   *int                    `json:"max_tokens,omitempty"`
	Stream      bool                    `json:"stream,omitempty"`
	// Provider-specific fields
	Provider map[string]interface{} `json:"provider,omitempty"`
}

// ChatCompletionMessage represents a message in the conversation.
type ChatCompletionMessage struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"` // The message content
}

// ChatCompletionResponse represents the response from the chat completions endpoint.
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *ChatCompletionUsage   `json:"usage,omitempty"`
	Error   *ChatCompletionError   `json:"error,omitempty"`
}

// ChatCompletionChoice represents a single completion choice.
type ChatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

// ChatCompletionUsage contains token usage information.
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionError represents an error response from the API.
type ChatCompletionError struct {
	Message  string                 `json:"message"`
	Type     string                 `json:"type"`
	Param    string                 `json:"param,omitempty"`
	Code     string                 `json:"code,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChatCompletionStreamChunk represents a chunk in a streaming response.
type ChatCompletionStreamChunk struct {
	ID      string                       `json:"id"`
	Object  string                       `json:"object"`
	Created int64                        `json:"created"`
	Model   string                       `json:"model"`
	Choices []ChatCompletionStreamChoice `json:"choices"`
	Usage   *ChatCompletionUsage         `json:"usage,omitempty"`
}

// ChatCompletionStreamChoice represents a choice in a streaming response.
type ChatCompletionStreamChoice struct {
	Index        int                        `json:"index"`
	Delta        ChatCompletionMessageDelta `json:"delta"`
	FinishReason *string                    `json:"finish_reason"`
}

// ChatCompletionMessageDelta represents incremental message content in streaming.
type ChatCompletionMessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// CreateChatCompletion sends a non-streaming chat completion request.
func (c *OpenAICompatClient) CreateChatCompletion(
	ctx context.Context,
	req ChatCompletionRequest,
) (*ChatCompletionResponse, error) {
	req.Stream = false

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			// Safely convert attempt-1 to uint (bounded by maxRetries, max value is 3)
			shift := min(attempt-1, 30) // Cap at 2^30 to prevent overflow
			//nolint:gosec // G115: shift is bounded by min(maxRetries, 30), safe from overflow
			backoff := time.Duration(1<<uint(shift)) * time.Second
			log.WithFields(map[string]interface{}{
				"attempt": attempt,
				"backoff": backoff.String(),
			}).Debug("retrying chat completion request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, req)
		if err != nil {
			lastErr = err
			// Only retry on server errors (5xx) or network errors
			if shouldRetry(err) {
				continue
			}
			return nil, err
		}

		return resp, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

// CreateChatCompletionStream sends a streaming chat completion request.
func (c *OpenAICompatClient) CreateChatCompletionStream(
	ctx context.Context,
	req ChatCompletionRequest,
	writer io.Writer,
) (*ChatCompletionUsage, error) {
	req.Stream = true

	httpReq, err := c.prepareStreamRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	return c.processStreamResponse(resp.Body, writer)
}

// prepareStreamRequest creates and configures an HTTP request for streaming.
func (c *OpenAICompatClient) prepareStreamRequest(ctx context.Context, req ChatCompletionRequest) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	log.WithFields(map[string]interface{}{
		"url":   httpReq.URL.String(),
		"model": req.Model,
	}).Debug("sending streaming chat completion request")

	return httpReq, nil
}

// processStreamResponse reads and processes an SSE stream response.
func (c *OpenAICompatClient) processStreamResponse(body io.Reader, writer io.Writer) (*ChatCompletionUsage, error) {
	scanner := bufio.NewScanner(body)
	var usage *ChatCompletionUsage

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// SSE format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// OpenAI sends "[DONE]" to signal end of stream
		if data == "[DONE]" {
			break
		}

		streamUsage, err := c.processStreamChunk(data, writer)
		if err != nil {
			return usage, err
		}

		// Capture usage if provided (usually in last chunk)
		if streamUsage != nil {
			usage = streamUsage
		}
	}

	if err := scanner.Err(); err != nil {
		return usage, fmt.Errorf("error reading stream: %w", err)
	}

	return usage, nil
}

// processStreamChunk parses and processes a single SSE chunk.
func (c *OpenAICompatClient) processStreamChunk(data string, writer io.Writer) (*ChatCompletionUsage, error) {
	var chunk ChatCompletionStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		log.WithError(err).WithField("data", data).Warn("failed to parse stream chunk")
		return nil, nil // Non-fatal error, continue processing
	}

	// Write delta content to writer
	if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
		if _, writeErr := writer.Write([]byte(chunk.Choices[0].Delta.Content)); writeErr != nil {
			return nil, fmt.Errorf("failed to write stream content: %w", writeErr)
		}
	}

	return chunk.Usage, nil
}

// HealthCheck performs a simple health check by making a minimal API request.
func (c *OpenAICompatClient) HealthCheck(ctx context.Context) error {
	req := ChatCompletionRequest{
		Model: "gpt-3.5-turbo", // Use a common fallback model
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "hi"},
		},
		MaxTokens: intPtr(1),
	}

	_, err := c.CreateChatCompletion(ctx, req)
	return err
}

// doRequest performs the actual HTTP request for non-streaming completions.
func (c *OpenAICompatClient) doRequest(
	ctx context.Context,
	req ChatCompletionRequest,
) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	log.WithFields(map[string]interface{}{
		"url":   httpReq.URL.String(),
		"model": req.Model,
	}).Debug("sending chat completion request")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	return &result, nil
}

// setHeaders sets the required HTTP headers for the request.
func (c *OpenAICompatClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}

// handleErrorResponse parses and returns an error from an HTTP error response.
func (c *OpenAICompatClient) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d (failed to read error body: %w)", resp.StatusCode, err)
	}

	var errorResp struct {
		Error *ChatCompletionError `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if errorResp.Error != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorResp.Error.Message)
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// shouldRetry determines if a request should be retried based on the error.
func shouldRetry(err error) bool {
	// Retry on network errors
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on 5xx server errors
	if strings.Contains(errStr, "HTTP 5") {
		return true
	}

	// Retry on connection errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF") {
		return true
	}

	// Don't retry on 4xx client errors (auth, bad request, etc.)
	return false
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int {
	return &i
}
