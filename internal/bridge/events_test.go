package bridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEventJSONSerialization(t *testing.T) {
	// Test that all event types serialize to JSON correctly
	sysInfo := SystemInfo{
		AgentPipeVersion: "0.2.4",
		OS:               "darwin",
		OSVersion:        "macOS 14.1",
		GoVersion:        "go1.24.0",
		Architecture:     "arm64",
	}

	agents := []AgentParticipant{
		{
			AgentType:  "claude",
			Model:      "claude-sonnet-4",
			Name:       "Claude",
			Prompt:     "You are a helpful assistant",
			CLIVersion: "1.2.0",
		},
	}

	// Test conversation.started event
	startedEvent := &Event{
		Type:      EventConversationStarted,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationStartedData{
			ConversationID: "test-conv-123",
			Mode:           "round-robin",
			InitialPrompt:  "Hello agents",
			MaxTurns:       10,
			Agents:         agents,
			SystemInfo:     sysInfo,
		},
	}

	data, err := json.Marshal(startedEvent)
	if err != nil {
		t.Fatalf("Failed to marshal conversation.started event: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["type"] != string(EventConversationStarted) {
		t.Errorf("Expected type=%s, got %v", EventConversationStarted, parsed["type"])
	}

	if _, ok := parsed["timestamp"]; !ok {
		t.Error("Expected timestamp field to be present")
	}

	dataMap, ok := parsed["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be an object")
	}

	if dataMap["conversation_id"] != "test-conv-123" {
		t.Errorf("Expected conversation_id=test-conv-123, got %v", dataMap["conversation_id"])
	}

	// Verify system_info is present and has all required fields
	systemInfoMap, ok := dataMap["system_info"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected system_info to be an object")
	}

	requiredSysInfoFields := []string{"agentpipe_version", "os", "os_version", "go_version", "architecture"}
	for _, field := range requiredSysInfoFields {
		if _, hasField := systemInfoMap[field]; !hasField {
			t.Errorf("Expected system_info.%s to be present", field)
		}
	}

	// Verify agents array is present and has cli_version
	agentsArray, ok := dataMap["agents"].([]interface{})
	if !ok || len(agentsArray) == 0 {
		t.Fatal("Expected agents to be a non-empty array")
	}

	agentMap, ok := agentsArray[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected first agent to be an object")
	}

	if agentMap["cli_version"] != "1.2.0" {
		t.Errorf("Expected cli_version=1.2.0, got %v", agentMap["cli_version"])
	}
}

func TestMessageCreatedEvent(t *testing.T) {
	event := &Event{
		Type:      EventMessageCreated,
		Timestamp: UTCTime{time.Now()},
		Data: MessageCreatedData{
			ConversationID: "test-conv-123",
			MessageID:      "msg-456",
			AgentType:      "claude",
			AgentName:      "Claude",
			Content:        "Hello, world!",
			SequenceNumber: 1,
			TurnNumber:     1,
			TokensUsed:     100,
			InputTokens:    50,
			OutputTokens:   50,
			Cost:           0.001,
			Model:          "claude-sonnet-4",
			DurationMs:     1234,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal message.created event: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["type"] != string(EventMessageCreated) {
		t.Errorf("Expected type=%s, got %v", EventMessageCreated, parsed["type"])
	}

	dataMap := parsed["data"].(map[string]interface{})
	if dataMap["message_id"] != "msg-456" {
		t.Errorf("Expected message_id=msg-456, got %v", dataMap["message_id"])
	}

	if dataMap["content"] != "Hello, world!" {
		t.Errorf("Expected content='Hello, world!', got %v", dataMap["content"])
	}

	// Verify numeric fields
	if dataMap["sequence_number"].(float64) != 1 {
		t.Errorf("Expected sequence_number=1, got %v", dataMap["sequence_number"])
	}

	if dataMap["duration_ms"].(float64) != 1234 {
		t.Errorf("Expected duration_ms=1234, got %v", dataMap["duration_ms"])
	}
}

func TestConversationCompletedEvent(t *testing.T) {
	event := &Event{
		Type:      EventConversationCompleted,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationCompletedData{
			ConversationID:  "test-conv-123",
			Status:          "completed",
			TotalMessages:   20,
			TotalTurns:      10,
			TotalTokens:     3000,
			TotalCost:       0.03,
			DurationSeconds: 300.5,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal conversation.completed event: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["type"] != string(EventConversationCompleted) {
		t.Errorf("Expected type=%s, got %v", EventConversationCompleted, parsed["type"])
	}

	dataMap := parsed["data"].(map[string]interface{})
	if dataMap["status"] != "completed" {
		t.Errorf("Expected status=completed, got %v", dataMap["status"])
	}

	if dataMap["total_messages"].(float64) != 20 {
		t.Errorf("Expected total_messages=20, got %v", dataMap["total_messages"])
	}

	if dataMap["duration_seconds"].(float64) != 300.5 {
		t.Errorf("Expected duration_seconds=300.5, got %v", dataMap["duration_seconds"])
	}
}

func TestConversationErrorEvent(t *testing.T) {
	event := &Event{
		Type:      EventConversationError,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationErrorData{
			ConversationID: "test-conv-123",
			ErrorMessage:   "API rate limit exceeded",
			ErrorType:      "rate_limit",
			AgentType:      "claude",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal conversation.error event: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["type"] != string(EventConversationError) {
		t.Errorf("Expected type=%s, got %v", EventConversationError, parsed["type"])
	}

	dataMap := parsed["data"].(map[string]interface{})
	if dataMap["error_message"] != "API rate limit exceeded" {
		t.Errorf("Expected error_message='API rate limit exceeded', got %v", dataMap["error_message"])
	}

	if dataMap["error_type"] != "rate_limit" {
		t.Errorf("Expected error_type=rate_limit, got %v", dataMap["error_type"])
	}

	if dataMap["agent_type"] != "claude" {
		t.Errorf("Expected agent_type=claude, got %v", dataMap["agent_type"])
	}
}

func TestBridgeTestEvent(t *testing.T) {
	sysInfo := SystemInfo{
		AgentPipeVersion: "0.3.3",
		OS:               "darwin",
		OSVersion:        "macOS 14.1",
		GoVersion:        "go1.24.0",
		Architecture:     "arm64",
	}

	event := &Event{
		Type:      EventBridgeTest,
		Timestamp: UTCTime{time.Now()},
		Data: BridgeTestData{
			Message:    "Bridge connection test",
			SystemInfo: sysInfo,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal bridge.test event: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["type"] != string(EventBridgeTest) {
		t.Errorf("Expected type=%s, got %v", EventBridgeTest, parsed["type"])
	}

	dataMap := parsed["data"].(map[string]interface{})
	if dataMap["message"] != "Bridge connection test" {
		t.Errorf("Expected message='Bridge connection test', got %v", dataMap["message"])
	}

	// Verify system_info is present
	systemInfoMap, ok := dataMap["system_info"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected system_info to be an object")
	}

	if systemInfoMap["agentpipe_version"] != "0.3.3" {
		t.Errorf("Expected agentpipe_version=0.3.3, got %v", systemInfoMap["agentpipe_version"])
	}
}

func TestTimestampFormat(t *testing.T) {
	// Test that timestamps are in ISO 8601 format with Z suffix
	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationStartedData{
			ConversationID: "test",
			Mode:           "round-robin",
			InitialPrompt:  "test",
			Agents:         []AgentParticipant{},
			SystemInfo:     SystemInfo{},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	jsonStr := string(data)

	// Verify timestamp is in ISO 8601 format (contains 'T' and 'Z' or timezone offset)
	if !strings.Contains(jsonStr, "\"timestamp\":\"") {
		t.Error("Expected timestamp field in JSON")
	}

	// The timestamp should be in RFC3339 format (ISO 8601)
	var parsed map[string]interface{}
	if err = json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	timestampStr, ok := parsed["timestamp"].(string)
	if !ok {
		t.Fatal("Expected timestamp to be a string")
	}

	// Verify timestamp ends with 'Z' for UTC
	if !strings.HasSuffix(timestampStr, "Z") {
		t.Errorf("Expected timestamp to end with 'Z', got: %s", timestampStr)
	}

	// Try parsing it back to verify it's valid RFC3339
	_, err = time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		t.Errorf("Failed to parse timestamp as RFC3339: %v (timestamp: %s)", err, timestampStr)
	}
}

func TestOmitemptyFields(t *testing.T) {
	// Test that omitempty fields are actually omitted when empty
	event := &Event{
		Type:      EventMessageCreated,
		Timestamp: UTCTime{time.Now()},
		Data: MessageCreatedData{
			ConversationID: "test-conv",
			MessageID:      "msg-123",
			AgentType:      "claude",
			Content:        "Hello",
			// Omit optional fields
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	jsonStr := string(data)

	// Optional fields should not be present when zero values
	if strings.Contains(jsonStr, "\"sequence_number\":0") {
		t.Error("Expected sequence_number:0 to be omitted")
	}

	if strings.Contains(jsonStr, "\"tokens_used\":0") {
		t.Error("Expected tokens_used:0 to be omitted")
	}

	if strings.Contains(jsonStr, "\"cost\":0") {
		t.Error("Expected cost:0 to be omitted")
	}
}
