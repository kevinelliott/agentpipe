package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOpenAICompatClient(t *testing.T) {
	client := NewOpenAICompatClient("https://api.example.com/v1", "test-api-key")

	if client == nil {
		t.Fatal("NewOpenAICompatClient returned nil")
	}

	if client.baseURL != "https://api.example.com/v1" {
		t.Errorf("Expected baseURL to be 'https://api.example.com/v1', got '%s'", client.baseURL)
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey to be 'test-api-key', got '%s'", client.apiKey)
	}

	if client.maxRetries != 3 {
		t.Errorf("Expected maxRetries to be 3, got %d", client.maxRetries)
	}
}

func TestCreateChatCompletion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Authorization Bearer test-api-key, got %s", r.Header.Get("Authorization"))
		}

		// Return successful response
		resp := ChatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-3.5-turbo",
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: ChatCompletionMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: &ChatCompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatClient(server.URL, "test-api-key")

	req := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	ctx := context.Background()
	resp, err := client.CreateChatCompletion(ctx, req)

	if err != nil {
		t.Fatalf("CreateChatCompletion failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Unexpected response content: %s", resp.Choices[0].Message.Content)
	}

	if resp.Usage == nil {
		t.Error("Expected usage data, got nil")
	} else {
		if resp.Usage.TotalTokens != 18 {
			t.Errorf("Expected 18 total tokens, got %d", resp.Usage.TotalTokens)
		}
	}
}

func TestCreateChatCompletion_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		errorResp := struct {
			Error *ChatCompletionError `json:"error"`
		}{
			Error: &ChatCompletionError{
				Message: "Invalid model specified",
				Type:    "invalid_request_error",
				Code:    "model_not_found",
			},
		}

		_ = json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	client := NewOpenAICompatClient(server.URL, "test-api-key")

	req := ChatCompletionRequest{
		Model: "invalid-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	ctx := context.Background()
	_, err := client.CreateChatCompletion(ctx, req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid model specified") {
		t.Errorf("Expected error message to contain 'Invalid model specified', got: %v", err)
	}
}

func TestCreateChatCompletionStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming flag in request
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if !req.Stream {
			t.Error("Expected stream flag to be true")
		}

		// Send SSE stream
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not support flushing")
		}

		// Send chunks
		chunks := []string{
			"Hello",
			"! ",
			"How ",
			"can ",
			"I ",
			"help?",
		}

		for i, content := range chunks {
			chunk := ChatCompletionStreamChunk{
				ID:      "chatcmpl-stream",
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   "gpt-3.5-turbo",
				Choices: []ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: ChatCompletionMessageDelta{
							Content: content,
						},
					},
				},
			}

			// Add usage in last chunk
			if i == len(chunks)-1 {
				chunk.Usage = &ChatCompletionUsage{
					PromptTokens:     10,
					CompletionTokens: 6,
					TotalTokens:      16,
				}
			}

			data, _ := json.Marshal(chunk)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}

		// Send done marker
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewOpenAICompatClient(server.URL, "test-api-key")

	req := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	ctx := context.Background()
	var buf bytes.Buffer
	usage, err := client.CreateChatCompletionStream(ctx, req, &buf)

	if err != nil {
		t.Fatalf("CreateChatCompletionStream failed: %v", err)
	}

	output := buf.String()
	expected := "Hello! How can I help?"

	if output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}

	if usage == nil {
		t.Error("Expected usage data, got nil")
	} else {
		if usage.TotalTokens != 16 {
			t.Errorf("Expected 16 total tokens, got %d", usage.TotalTokens)
		}
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "HTTP 500 error",
			err:      fmt.Errorf("HTTP 500 Internal Server Error"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      fmt.Errorf("connection refused"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      fmt.Errorf("timeout exceeded"),
			expected: true,
		},
		{
			name:     "HTTP 400 error",
			err:      fmt.Errorf("HTTP 400 Bad Request"),
			expected: false,
		},
		{
			name:     "HTTP 401 error",
			err:      fmt.Errorf("HTTP 401 Unauthorized"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("shouldRetry(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCreateChatCompletion_WithRetry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		// Fail first 2 attempts, succeed on 3rd
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
			return
		}

		resp := ChatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-3.5-turbo",
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: ChatCompletionMessage{
						Role:    "assistant",
						Content: "Success after retries",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatClient(server.URL, "test-api-key")

	req := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	ctx := context.Background()
	resp, err := client.CreateChatCompletion(ctx, req)

	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if resp.Choices[0].Message.Content != "Success after retries" {
		t.Errorf("Unexpected response: %s", resp.Choices[0].Message.Content)
	}
}

func TestCreateChatCompletion_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOpenAICompatClient(server.URL, "test-api-key")

	req := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.CreateChatCompletion(ctx, req)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context error, got: %v", err)
	}
}
