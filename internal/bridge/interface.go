package bridge

import (
	"time"
)

// BridgeEmitter is the interface for emitting conversation events.
// Both the HTTP-based Emitter and the stdout-based StdoutEmitter implement this interface.
type BridgeEmitter interface {
	GetConversationID() string
	EmitConversationStarted(
		mode string,
		initialPrompt string,
		maxTurns int,
		participants []AgentParticipant,
		commandInfo *CommandInfo,
	)
	EmitMessageCreated(
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
	)
	EmitConversationCompleted(
		status string,
		totalMessages int,
		totalTurns int,
		totalTokens int,
		totalCost float64,
		duration time.Duration,
		summary *SummaryMetadata,
	)
	EmitConversationError(errorMessage string, errorType string, agentType string)
	Close() error
}
