package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewEmitter(t *testing.T) {
	config := &Config{
		Enabled:       true,
		URL:           "https://example.com",
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "info",
	}

	version := "0.2.4"
	emitter := NewEmitter(config, version)

	if emitter == nil {
		t.Fatal("Expected emitter to be created")
	}

	if emitter.client == nil {
		t.Error("Expected client to be initialized")
	}

	// Conversation ID should be a valid UUID
	if emitter.conversationID == "" {
		t.Error("Expected conversation ID to be set")
	}

	if !strings.Contains(emitter.conversationID, "-") {
		t.Error("Expected conversation ID to be UUID format")
	}

	// Sequence number should start at 0
	if emitter.sequenceNumber != 0 {
		t.Errorf("Expected sequence number=0, got %d", emitter.sequenceNumber)
	}

	// System info should be collected
	if emitter.systemInfo.AgentPipeVersion != version {
		t.Errorf("Expected AgentPipeVersion=%s, got %s", version, emitter.systemInfo.AgentPipeVersion)
	}
}

func TestGetConversationID(t *testing.T) {
	config := &Config{
		Enabled: true,
		URL:     "https://example.com",
		APIKey:  "sk_test",
	}

	emitter := NewEmitter(config, "0.2.4")
	convID := emitter.GetConversationID()

	if convID == "" {
		t.Error("Expected conversation ID to be returned")
	}

	if convID != emitter.conversationID {
		t.Error("Expected GetConversationID to return internal conversation ID")
	}
}

func TestEmitConversationStarted(t *testing.T) {
	receivedEvents := make(chan *Event, 10)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedEvents <- &event
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	emitter := NewEmitter(config, "0.2.4")

	agents := []AgentParticipant{
		{
			AgentType:  "claude",
			Model:      "claude-sonnet-4",
			Name:       "Claude",
			CLIVersion: "1.2.0",
		},
	}

	emitter.EmitConversationStarted("round-robin", "Hello agents", 10, agents, nil)

	// Collect both events (bridge.connected and conversation.started)
	events := collectEvents(t, receivedEvents, 2)

	// First event should be bridge.connected
	if events[0].Type != EventBridgeConnected {
		t.Errorf("Expected first event type=%s, got %s", EventBridgeConnected, events[0].Type)
	}

	// Second event should be conversation.started
	event := events[1]
	if event.Type != EventConversationStarted {
		t.Errorf("Expected second event type=%s, got %s", EventConversationStarted, event.Type)
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["mode"] != "round-robin" {
		t.Errorf("Expected mode=round-robin, got %v", data["mode"])
	}

	if data["initial_prompt"] != "Hello agents" {
		t.Errorf("Expected initial_prompt='Hello agents', got %v", data["initial_prompt"])
	}

	// Verify system_info is present
	if _, ok := data["system_info"]; !ok {
		t.Error("Expected system_info to be present in conversation.started event")
	}
}

func TestEmitMessageCreated(t *testing.T) {
	receivedEvents := make(chan *Event, 10)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedEvents <- &event
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	emitter := NewEmitter(config, "0.2.4")

	// Emit two messages to test sequence numbering
	emitter.EmitMessageCreated("claude-0", "claude", "Claude", "Hello", "claude-sonnet-4", 1, 100, 50, 50, 0.001, 1234*time.Millisecond)
	emitter.EmitMessageCreated("gemini-0", "gemini", "Gemini", "Hi", "gemini-pro", 1, 80, 40, 40, 0.0008, 987*time.Millisecond)

	// Collect all three events (bridge.connected + two messages)
	events := collectEvents(t, receivedEvents, 3)

	// First event should be bridge.connected
	if events[0].Type != EventBridgeConnected {
		t.Errorf("Expected first event type=%s, got %s", EventBridgeConnected, events[0].Type)
	}

	// Verify the two message events by sequence number
	messageEvents := events[1:]
	for _, event := range messageEvents {
		verifyMessageEvent(t, event)
	}
}

// Helper to collect multiple events with timeout
func collectEvents(t *testing.T, ch chan *Event, count int) []*Event {
	t.Helper()
	var events []*Event
	for i := 0; i < count; i++ {
		select {
		case event := <-ch:
			events = append(events, event)
		case <-time.After(2 * time.Second):
			t.Fatalf("Timeout waiting for event %d", i+1)
		}
	}
	return events
}

// Helper to verify a message.created event
func verifyMessageEvent(t *testing.T, event *Event) {
	t.Helper()

	if event.Type != EventMessageCreated {
		t.Errorf("Expected type=%s, got %s", EventMessageCreated, event.Type)
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	seqNum := int(data["sequence_number"].(float64))
	if seqNum == 1 {
		if data["content"] != "Hello" {
			t.Errorf("Expected content='Hello' for seq 1, got %v", data["content"])
		}
		if data["agent_name"] != "Claude" {
			t.Errorf("Expected agent_name='Claude' for seq 1, got %v", data["agent_name"])
		}
	} else if seqNum == 2 {
		if data["content"] != "Hi" {
			t.Errorf("Expected content='Hi' for seq 2, got %v", data["content"])
		}
		if data["agent_name"] != "Gemini" {
			t.Errorf("Expected agent_name='Gemini' for seq 2, got %v", data["agent_name"])
		}
	} else {
		t.Errorf("Unexpected sequence number: %d", seqNum)
	}

	// Verify message_id is a UUID
	messageID, ok := data["message_id"].(string)
	if !ok || messageID == "" {
		t.Error("Expected message_id to be a non-empty string")
	}
}

func TestEmitConversationCompleted(t *testing.T) {
	receivedEvents := make(chan *Event, 10)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedEvents <- &event
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	emitter := NewEmitter(config, "0.2.4")

	summary := &SummaryMetadata{
		ShortSummary: "Test summary.",
		Summary:      "Test summary of the conversation",
		AgentType:    "gemini",
		Model:        "gemini-2.0-flash",
		InputTokens:  2500,
		OutputTokens: 150,
		TotalTokens:  2650,
		Cost:         0.002,
		DurationMs:   1200,
	}

	emitter.EmitConversationCompleted("completed", 20, 10, 3000, 0.03, 300*time.Second, summary)

	// Collect both events (bridge.connected and conversation.completed)
	events := collectEvents(t, receivedEvents, 2)

	// First event should be bridge.connected
	if events[0].Type != EventBridgeConnected {
		t.Errorf("Expected first event type=%s, got %s", EventBridgeConnected, events[0].Type)
	}

	// Second event should be conversation.completed
	event := events[1]
	if event.Type != EventConversationCompleted {
		t.Errorf("Expected second event type=%s, got %s", EventConversationCompleted, event.Type)
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["status"] != "completed" {
		t.Errorf("Expected status=completed, got %v", data["status"])
	}

	if data["total_messages"].(float64) != 20 {
		t.Errorf("Expected total_messages=20, got %v", data["total_messages"])
	}

	if data["duration_seconds"].(float64) != 300.0 {
		t.Errorf("Expected duration_seconds=300.0, got %v", data["duration_seconds"])
	}
}

func TestEmitConversationError(t *testing.T) {
	receivedEvents := make(chan *Event, 10)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedEvents <- &event
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		LogLevel:      "debug",
	}

	emitter := NewEmitter(config, "0.2.4")

	emitter.EmitConversationError("API rate limit exceeded", "rate_limit", "claude")

	// Collect both events (bridge.connected and conversation.error)
	events := collectEvents(t, receivedEvents, 2)

	// First event should be bridge.connected
	if events[0].Type != EventBridgeConnected {
		t.Errorf("Expected first event type=%s, got %s", EventBridgeConnected, events[0].Type)
	}

	// Second event should be conversation.error
	event := events[1]
	if event.Type != EventConversationError {
		t.Errorf("Expected second event type=%s, got %s", EventConversationError, event.Type)
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["error_message"] != "API rate limit exceeded" {
		t.Errorf("Expected error_message='API rate limit exceeded', got %v", data["error_message"])
	}

	if data["error_type"] != "rate_limit" {
		t.Errorf("Expected error_type=rate_limit, got %v", data["error_type"])
	}

	if data["agent_type"] != "claude" {
		t.Errorf("Expected agent_type=claude, got %v", data["agent_type"])
	}
}

func TestSequenceNumbering(t *testing.T) {
	config := &Config{
		Enabled: false, // Disabled to avoid network calls
		URL:     "https://example.com",
		APIKey:  "sk_test",
	}

	emitter := NewEmitter(config, "0.2.4")

	// Initial sequence number should be 0
	if emitter.sequenceNumber != 0 {
		t.Errorf("Expected initial sequence_number=0, got %d", emitter.sequenceNumber)
	}

	// After first message, should be 1
	emitter.EmitMessageCreated("claude-0", "claude", "Claude", "msg1", "model", 1, 100, 50, 50, 0.001, 1*time.Second)
	if emitter.sequenceNumber != 1 {
		t.Errorf("Expected sequence_number=1 after first message, got %d", emitter.sequenceNumber)
	}

	// After second message, should be 2
	emitter.EmitMessageCreated("gemini-0", "gemini", "Gemini", "msg2", "model", 1, 100, 50, 50, 0.001, 1*time.Second)
	if emitter.sequenceNumber != 2 {
		t.Errorf("Expected sequence_number=2 after second message, got %d", emitter.sequenceNumber)
	}

	// After third message, should be 3
	emitter.EmitMessageCreated("claude-1", "claude", "Claude", "msg3", "model", 2, 100, 50, 50, 0.001, 1*time.Second)
	if emitter.sequenceNumber != 3 {
		t.Errorf("Expected sequence_number=3 after third message, got %d", emitter.sequenceNumber)
	}
}

func TestUniqueConversationIDs(t *testing.T) {
	config := &Config{
		Enabled: true,
		URL:     "https://example.com",
		APIKey:  "sk_test",
	}

	// Create multiple emitters
	emitter1 := NewEmitter(config, "0.2.4")
	emitter2 := NewEmitter(config, "0.2.4")
	emitter3 := NewEmitter(config, "0.2.4")

	// Each should have a unique conversation ID
	if emitter1.conversationID == emitter2.conversationID {
		t.Error("Expected unique conversation IDs for different emitters")
	}

	if emitter1.conversationID == emitter3.conversationID {
		t.Error("Expected unique conversation IDs for different emitters")
	}

	if emitter2.conversationID == emitter3.conversationID {
		t.Error("Expected unique conversation IDs for different emitters")
	}
}

func TestBridgeConnectedEvent(t *testing.T) {
	// Track received events
	receivedEvents := []Event{}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedEvents = append(receivedEvents, event)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Enabled:       true,
		URL:           server.URL,
		APIKey:        "sk_test",
		TimeoutMs:     5000,
		RetryAttempts: 0,
		LogLevel:      "debug",
	}

	// Creating the emitter should automatically send bridge.connected event
	version := "0.3.8-test"
	_ = NewEmitter(config, version)

	// Verify we received the bridge.connected event
	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(receivedEvents))
	}

	event := receivedEvents[0]
	if event.Type != EventBridgeConnected {
		t.Errorf("Expected event type=%s, got %s", EventBridgeConnected, event.Type)
	}

	// Verify data structure
	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	// Verify system_info is present
	systemInfoMap, ok := dataMap["system_info"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected system_info to be present")
	}

	if systemInfoMap["agentpipe_version"] != version {
		t.Errorf("Expected agentpipe_version=%s, got %v", version, systemInfoMap["agentpipe_version"])
	}

	// Verify connected_at is present
	connectedAt, ok := dataMap["connected_at"].(string)
	if !ok {
		t.Fatal("Expected connected_at to be a string")
	}

	if connectedAt == "" {
		t.Error("Expected connected_at to be non-empty")
	}
}
