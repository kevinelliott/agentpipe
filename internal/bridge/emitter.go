package bridge

import (
	"time"

	"github.com/google/uuid"
)

// Emitter provides high-level methods for emitting streaming events
type Emitter struct {
	client         *Client
	conversationID string
	sequenceNumber int
	systemInfo     SystemInfo
}

// NewEmitter creates a new event emitter for a conversation
// Automatically sends a bridge.connected event to announce the connection
func NewEmitter(config *Config, agentpipeVersion string) *Emitter {
	emitter := &Emitter{
		client:         NewClient(config),
		conversationID: uuid.New().String(),
		sequenceNumber: 0,
		systemInfo:     CollectSystemInfo(agentpipeVersion),
	}

	// Emit bridge.connected event to announce the connection
	emitter.emitBridgeConnected()

	return emitter
}

// GetConversationID returns the conversation ID for this emitter
func (e *Emitter) GetConversationID() string {
	return e.conversationID
}

// EmitConversationStarted emits a conversation.started event
func (e *Emitter) EmitConversationStarted(
	mode string,
	initialPrompt string,
	maxTurns int,
	agents []AgentParticipant,
) {
	event := &Event{
		Type:      EventConversationStarted,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationStartedData{
			ConversationID: e.conversationID,
			Mode:           mode,
			InitialPrompt:  initialPrompt,
			MaxTurns:       maxTurns,
			Participants:   agents,
			SystemInfo:     e.systemInfo,
		},
	}
	e.client.SendEventAsync(event)
}

// EmitMessageCreated emits a message.created event
func (e *Emitter) EmitMessageCreated(
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
	e.sequenceNumber++
	event := &Event{
		Type:      EventMessageCreated,
		Timestamp: UTCTime{time.Now()},
		Data: MessageCreatedData{
			ConversationID: e.conversationID,
			MessageID:      uuid.New().String(),
			AgentType:      agentType,
			AgentName:      agentName,
			Content:        content,
			SequenceNumber: e.sequenceNumber,
			TurnNumber:     turnNumber,
			TokensUsed:     tokensUsed,
			InputTokens:    inputTokens,
			OutputTokens:   outputTokens,
			Cost:           cost,
			Model:          model,
			DurationMs:     duration.Milliseconds(),
		},
	}
	e.client.SendEventAsync(event)
}

// EmitConversationCompleted emits a conversation.completed event
// Uses synchronous send to ensure the event is fully sent before program exit
func (e *Emitter) EmitConversationCompleted(
	status string,
	totalMessages int,
	totalTurns int,
	totalTokens int,
	totalCost float64,
	duration time.Duration,
) {
	event := &Event{
		Type:      EventConversationCompleted,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationCompletedData{
			ConversationID:  e.conversationID,
			Status:          status,
			TotalMessages:   totalMessages,
			TotalTurns:      totalTurns,
			TotalTokens:     totalTokens,
			TotalCost:       totalCost,
			DurationSeconds: duration.Seconds(),
		},
	}
	// Use synchronous send for completion event to ensure it's sent before program exit
	_ = e.client.SendEvent(event)
}

// EmitConversationError emits a conversation.error event
// Uses synchronous send to ensure the event is fully sent before program exit
func (e *Emitter) EmitConversationError(
	errorMessage string,
	errorType string,
	agentType string,
) {
	event := &Event{
		Type:      EventConversationError,
		Timestamp: UTCTime{time.Now()},
		Data: ConversationErrorData{
			ConversationID: e.conversationID,
			ErrorMessage:   errorMessage,
			ErrorType:      errorType,
			AgentType:      agentType,
		},
	}
	// Use synchronous send for error event to ensure it's sent before program exit
	_ = e.client.SendEvent(event)
}

// emitBridgeConnected emits a bridge.connected event to announce the connection
// This is called automatically when the emitter is created
func (e *Emitter) emitBridgeConnected() {
	event := &Event{
		Type:      EventBridgeConnected,
		Timestamp: UTCTime{time.Now()},
		Data: BridgeConnectedData{
			SystemInfo:  e.systemInfo,
			ConnectedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
	// Use synchronous send to ensure connection is announced before proceeding
	_ = e.client.SendEvent(event)
}
