package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StdoutEmitter emits events to stdout as JSON lines (JSONL format)
// This is useful for programmatic consumption and CI/CD pipelines
type StdoutEmitter struct {
	conversationID string
	sequenceNum    int
	mu             sync.Mutex
	version        string
}

// NewStdoutEmitter creates a new stdout emitter
func NewStdoutEmitter(version string) *StdoutEmitter {
	emitter := &StdoutEmitter{
		conversationID: uuid.New().String(),
		sequenceNum:    0,
		version:        version,
	}

	// Emit bridge.connected event immediately
	emitter.emitBridgeConnected()

	return emitter
}

// GetConversationID returns the conversation ID
func (e *StdoutEmitter) GetConversationID() string {
	return e.conversationID
}

// emitEvent writes an event as JSON to stdout
func (e *StdoutEmitter) emitEvent(event Event) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Write to stdout with newline for JSONL format
	fmt.Fprintln(os.Stdout, string(jsonData))
	return nil
}

// emitBridgeConnected emits the bridge.connected event with system info
func (e *StdoutEmitter) emitBridgeConnected() {
	sysInfo := CollectSystemInfo(e.version)

	event := Event{
		Type:      EventBridgeConnected,
		Timestamp: UTCTime{Time: time.Now()},
		Data: BridgeConnectedData{
			SystemInfo:  sysInfo,
			ConnectedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	_ = e.emitEvent(event) // Ignore error for initialization event
}

// EmitConversationStarted emits a conversation.started event
func (e *StdoutEmitter) EmitConversationStarted(
	mode string,
	initialPrompt string,
	maxTurns int,
	participants []AgentParticipant,
	commandInfo *CommandInfo,
) {
	data := ConversationStartedData{
		ConversationID: e.conversationID,
		Mode:           mode,
		InitialPrompt:  initialPrompt,
		MaxTurns:       maxTurns,
		Participants:   participants,
		SystemInfo:     CollectSystemInfo(e.version),
		Command:        commandInfo,
	}

	event := Event{
		Type:      EventConversationStarted,
		Timestamp: UTCTime{Time: time.Now()},
		Data:      data,
	}

	_ = e.emitEvent(event)
}

// EmitMessageCreated emits a message.created event
func (e *StdoutEmitter) EmitMessageCreated(
	agentID string,
	agentType string,
	agentName string,
	content string,
	model string,
	turnNumber int,
	tokensUsed int,
	inputTokens int,
	outputTokens int,
	cost float64,
	duration time.Duration,
) {
	e.mu.Lock()
	e.sequenceNum++
	seqNum := e.sequenceNum
	e.mu.Unlock()

	data := MessageCreatedData{
		ConversationID: e.conversationID,
		MessageID:      uuid.New().String(),
		AgentID:        agentID,
		AgentType:      agentType,
		AgentName:      agentName,
		Content:        content,
		SequenceNumber: seqNum,
		TurnNumber:     turnNumber,
		TokensUsed:     tokensUsed,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		Cost:           cost,
		Model:          model,
		DurationMs:     duration.Milliseconds(),
	}

	event := Event{
		Type:      EventMessageCreated,
		Timestamp: UTCTime{Time: time.Now()},
		Data:      data,
	}

	_ = e.emitEvent(event)
}

// Close is a no-op for StdoutEmitter (no resources to clean up)
func (e *StdoutEmitter) Close() error {
	return nil
}

// EmitConversationCompleted emits a conversation.completed event
func (e *StdoutEmitter) EmitConversationCompleted(
	status string,
	totalMessages int,
	totalTurns int,
	totalTokens int,
	totalCost float64,
	duration time.Duration,
	summary *SummaryMetadata,
) {
	data := ConversationCompletedData{
		ConversationID:  e.conversationID,
		Status:          status,
		TotalMessages:   totalMessages,
		TotalTurns:      totalTurns,
		TotalTokens:     totalTokens,
		TotalCost:       totalCost,
		DurationSeconds: duration.Seconds(),
		Summary:         summary,
	}

	event := Event{
		Type:      EventConversationCompleted,
		Timestamp: UTCTime{Time: time.Now()},
		Data:      data,
	}

	_ = e.emitEvent(event)
}

// EmitConversationError emits a conversation.error event
func (e *StdoutEmitter) EmitConversationError(errorMessage string, errorType string, agentType string) {
	data := ConversationErrorData{
		ConversationID: e.conversationID,
		ErrorMessage:   errorMessage,
		ErrorType:      errorType,
		AgentType:      agentType,
	}

	event := Event{
		Type:      EventConversationError,
		Timestamp: UTCTime{Time: time.Now()},
		Data:      data,
	}

	_ = e.emitEvent(event)
}

// EmitLogEntry emits a log.entry event for log messages
func (e *StdoutEmitter) EmitLogEntry(
	level string,
	agentID string,
	agentName string,
	agentType string,
	content string,
	role string,
	metrics *LogEntryMetrics,
	metadata map[string]interface{},
) {
	data := LogEntryData{
		ConversationID: e.conversationID,
		Level:          level,
		AgentID:        agentID,
		AgentName:      agentName,
		AgentType:      agentType,
		Content:        content,
		Role:           role,
		Metrics:        metrics,
		Metadata:       metadata,
	}

	event := Event{
		Type:      EventLogEntry,
		Timestamp: UTCTime{Time: time.Now()},
		Data:      data,
	}

	_ = e.emitEvent(event)
}
