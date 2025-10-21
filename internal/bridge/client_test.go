package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := &Config{
		Enabled:       true,
		URL:           "https://example.com",
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "info",
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config != config {
		t.Error("Expected client config to match input config")
	}

	if client.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected timeout=5s, got %v", client.httpClient.Timeout)
	}
}

func TestGetEndpointURL(t *testing.T) {
	config := &Config{
		URL: "https://example.com",
	}

	client := NewClient(config)
	endpoint := client.getEndpointURL()

	expected := "https://example.com/api/ingest"
	if endpoint != expected {
		t.Errorf("Expected endpoint=%s, got %s", expected, endpoint)
	}
}

func TestSendEvent_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify path
		if r.URL.Path != "/api/ingest" {
			t.Errorf("Expected path=/api/ingest, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer sk_test_key" {
			t.Errorf("Expected Authorization: Bearer sk_test_key, got %s", auth)
		}

		// Verify body
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if event.Type != EventConversationStarted {
			t.Errorf("Expected event type=%s, got %s", EventConversationStarted, event.Type)
		}

		// Return success
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"conversation_id": "test-123",
		})
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test_key",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data: ConversationStartedData{
			ConversationID: "test-123",
			Mode:           "round-robin",
			InitialPrompt:  "Test",
			Agents:         []AgentParticipant{},
			SystemInfo:     SystemInfo{},
		},
	}

	err := client.SendEvent(event)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
}

func TestSendEvent_Disabled(t *testing.T) {
	config := &Config{
		Enabled: false, // Disabled
		URL:     "https://example.com",
		APIKey:  "sk_test",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	// Should return nil (no error) when disabled
	err := client.SendEvent(event)
	if err != nil {
		t.Errorf("Expected nil when disabled, got error: %v", err)
	}
}

func TestSendEvent_NoAPIKey(t *testing.T) {
	config := &Config{
		Enabled:  true,
		URL:      "https://example.com",
		APIKey:   "", // No API key
		LogLevel: "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	err := client.SendEvent(event)
	if err == nil {
		t.Error("Expected error when API key is missing")
	}

	if !strings.Contains(err.Error(), "no API key") {
		t.Errorf("Expected 'no API key' error, got: %v", err)
	}
}

func TestSendEvent_Unauthorized(t *testing.T) {
	// Create mock server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid API key",
		})
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_bad_key",
		TimeoutMs:     5000,
		RetryAttempts: 0, // No retries for faster test
		LogLevel:      "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	err := client.SendEvent(event)
	if err == nil {
		t.Error("Expected error for 401 response")
	}

	// Should be an httpError
	if httpErr, ok := err.(*httpError); ok {
		if httpErr.statusCode != 401 {
			t.Errorf("Expected status code 401, got %d", httpErr.statusCode)
		}
	} else {
		t.Errorf("Expected httpError, got %T", err)
	}
}

func TestSendEvent_ServerError(t *testing.T) {
	// Create mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test_key",
		TimeoutMs:     5000,
		RetryAttempts: 0, // No retries for faster test
		LogLevel:      "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	err := client.SendEvent(event)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestSendEvent_Retry(t *testing.T) {
	attemptCount := 0

	// Create mock server that fails first 2 times, succeeds on 3rd
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"conversation_id": "test-123",
		})
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test_key",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	err := client.SendEvent(event)
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestSendEvent_NoRetryOn4xx(t *testing.T) {
	attemptCount := 0

	// Create mock server that always returns 400
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test_key",
		TimeoutMs:     5000,
		RetryAttempts: 3, // Should not retry on 4xx
		LogLevel:      "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	err := client.SendEvent(event)
	if err == nil {
		t.Error("Expected error for 400 response")
	}

	// Should only attempt once (no retries for client errors)
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt (no retry on 4xx), got %d", attemptCount)
	}
}

func TestSendEventAsync(t *testing.T) {
	receivedChan := make(chan bool, 1)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedChan <- true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test_key",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	client := NewClient(config)

	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: time.Now(),
		Data:      ConversationStartedData{},
	}

	// Should return immediately (async)
	client.SendEventAsync(event)

	// Wait for the async request to complete
	select {
	case <-receivedChan:
		// Success - server received the request
	case <-time.After(1 * time.Second):
		t.Error("Timeout: Expected server to receive async request")
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{400, true},
		{401, true},
		{404, true},
		{499, true},
		{500, false},
		{502, false},
		{200, false},
		{300, false},
	}

	for _, tt := range tests {
		err := &httpError{statusCode: tt.statusCode}
		result := isClientError(err)
		if result != tt.expected {
			t.Errorf("isClientError(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
		}
	}

	// Test with non-httpError
	if isClientError(http.ErrNotSupported) {
		t.Error("Expected false for non-httpError")
	}
}
