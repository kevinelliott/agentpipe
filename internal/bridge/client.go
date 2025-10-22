package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client is an HTTP client for sending streaming events to AgentPipe Web
type Client struct {
	config           *Config
	httpClient       *http.Client
	suppressWarnings bool // Set to true after first failure to avoid spamming warnings
}

// NewClient creates a new bridge client with the given configuration
func NewClient(config *Config) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		},
		suppressWarnings: false,
	}
}

// getEndpointURL returns the full API endpoint URL by appending /api/ingest to the base URL
func (c *Client) getEndpointURL() string {
	return c.config.URL + "/api/ingest"
}

// SendEvent sends an event to the streaming endpoint with retry logic
// Returns an error if all retry attempts fail, but logs errors instead of failing the conversation
func (c *Client) SendEvent(event *Event) error {
	if !c.config.Enabled {
		return nil // Silently skip if streaming is disabled
	}

	// Validate that we have an API key
	if c.config.APIKey == "" {
		if c.config.LogLevel == "debug" {
			fmt.Fprintln(os.Stderr, "Debug: Streaming enabled but no API key configured")
		}
		return fmt.Errorf("streaming enabled but no API key configured")
	}

	// Serialize event to JSON
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			// Safe conversion: attempt is always > 0 here, so attempt-1 >= 0
			//nolint:gosec // G115: Safe conversion - attempt > 0 guarantees attempt-1 >= 0
			exponent := uint(attempt - 1)
			backoff := time.Duration(1<<exponent) * time.Second
			time.Sleep(backoff)

			if c.config.LogLevel == "debug" {
				fmt.Fprintf(os.Stderr, "Debug: Retry attempt %d/%d after %v\n",
					attempt, c.config.RetryAttempts, backoff)
			}
		}

		err := c.sendRequest(body)
		if err == nil {
			if c.config.LogLevel == "debug" {
				fmt.Fprintf(os.Stderr, "Debug: Successfully sent %s event\n", event.Type)
			}
			return nil // Success
		}

		lastErr = err

		// Don't retry on client errors (4xx), only on network/server errors
		if isClientError(err) {
			break
		}
	}

	// Log error but don't fail the conversation
	if !c.suppressWarnings {
		// Show a user-friendly warning only once
		fmt.Fprintln(os.Stderr, "\n⚠️  Bridge streaming unavailable - conversation will continue normally")
		fmt.Fprintln(os.Stderr, "   (Events will be saved locally and can be uploaded later)")
		c.suppressWarnings = true
	}

	// Log detailed error at debug level only
	if c.config.LogLevel == "debug" {
		fmt.Fprintf(os.Stderr, "Debug: Failed to stream event after %d attempts: %v\n",
			c.config.RetryAttempts+1, lastErr)
	}

	return lastErr
}

// sendRequest performs a single HTTP request to send an event
func (c *Client) sendRequest(body []byte) error {
	url := c.getEndpointURL()

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Success codes
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return nil
	}

	// Read error response for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	return &httpError{
		statusCode: resp.StatusCode,
		message:    string(bodyBytes),
	}
}

// SendEventAsync sends an event asynchronously in a goroutine (non-blocking)
// Errors are logged at debug level but do not block or fail the conversation
func (c *Client) SendEventAsync(event *Event) {
	go func() {
		if err := c.SendEvent(event); err != nil {
			// Log at debug level only to avoid cluttering output
			if c.config.LogLevel == "debug" {
				fmt.Fprintf(os.Stderr, "Debug: Async stream event error: %v\n", err)
			}
		}
	}()
}

// httpError represents an HTTP error response
type httpError struct {
	statusCode int
	message    string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.statusCode, e.message)
}

// isClientError returns true if the error is a 4xx client error (should not retry)
func isClientError(err error) bool {
	if httpErr, ok := err.(*httpError); ok {
		return httpErr.statusCode >= 400 && httpErr.statusCode < 500
	}
	return false
}
