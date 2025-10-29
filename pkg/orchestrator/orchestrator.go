// Package orchestrator manages multi-agent conversations with different orchestration modes.
// It coordinates agent interactions, handles turn-taking, and manages message history.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/kevinelliott/agentpipe/internal/bridge"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/logger"
	"github.com/kevinelliott/agentpipe/pkg/metrics"
	"github.com/kevinelliott/agentpipe/pkg/middleware"
	"github.com/kevinelliott/agentpipe/pkg/ratelimit"
	"github.com/kevinelliott/agentpipe/pkg/utils"
)

// ConversationMode defines how agents take turns in a conversation.
type ConversationMode string

const (
	// ModeRoundRobin has agents take turns in a fixed circular order
	ModeRoundRobin ConversationMode = "round-robin"
	// ModeReactive randomly selects the next agent, but never the same agent twice in a row
	ModeReactive ConversationMode = "reactive"
	// ModeFreeForm allows all agents to respond if they want to participate
	ModeFreeForm ConversationMode = "free-form"
)

// OrchestratorConfig contains configuration for an Orchestrator instance.
type OrchestratorConfig struct {
	// Mode determines how agents take turns (round-robin, reactive, or free-form)
	Mode ConversationMode
	// TurnTimeout is the maximum time an agent has to respond
	TurnTimeout time.Duration
	// MaxTurns is the maximum number of conversation turns (0 = unlimited)
	MaxTurns int
	// ResponseDelay is the pause between agent responses
	ResponseDelay time.Duration
	// InitialPrompt is an optional starting prompt for the conversation
	InitialPrompt string
	// MaxRetries is the maximum number of retry attempts for failed agent responses (0 = no retries)
	MaxRetries int
	// RetryInitialDelay is the initial delay before the first retry
	RetryInitialDelay time.Duration
	// RetryMaxDelay is the maximum delay between retries
	RetryMaxDelay time.Duration
	// RetryMultiplier is the multiplier for exponential backoff (typically 2.0)
	RetryMultiplier float64
	// Summary defines conversation summary generation settings
	Summary config.SummaryConfig
}

// Orchestrator coordinates multi-agent conversations.
// It manages agent registration, turn-taking, message history, and logging.
// All methods are safe for concurrent use.
type Orchestrator struct {
	config            OrchestratorConfig
	agents            []agent.Agent
	messages          []agent.Message
	rateLimiters      map[string]*ratelimit.Limiter // per-agent rate limiters
	middlewareChain   *middleware.Chain             // message processing middleware
	mu                sync.RWMutex
	writer            io.Writer
	logger            *logger.ChatLogger
	currentTurnNumber int                     // tracks the current turn number for middleware context
	metrics           *metrics.Metrics        // Prometheus metrics for monitoring
	bridgeEmitter     bridge.BridgeEmitter    // optional streaming bridge for real-time updates
	conversationStart time.Time               // conversation start time for duration tracking
	commandInfo       *bridge.CommandInfo     // information about the command that started this conversation
	summary           *bridge.SummaryMetadata // conversation summary (populated after completion if enabled)
}

// NewOrchestrator creates a new Orchestrator with the given configuration.
// Default values are applied if TurnTimeout (30s) or ResponseDelay (1s) are zero.
// Retry defaults: MaxRetries=3, InitialDelay=1s, MaxDelay=30s, Multiplier=2.0.
// To disable retries, explicitly set all retry fields (at minimum RetryInitialDelay)
// The writer receives formatted conversation output for display (e.g., TUI).
func NewOrchestrator(config OrchestratorConfig, writer io.Writer) *Orchestrator {
	if config.TurnTimeout == 0 {
		config.TurnTimeout = 30 * time.Second
	}
	if config.ResponseDelay == 0 {
		config.ResponseDelay = 1 * time.Second
	}

	// Only apply retry defaults if retry config appears unset
	// Check if RetryInitialDelay is 0 - if so, assume retry config is not set
	if config.RetryInitialDelay == 0 && config.MaxRetries == 0 && config.RetryMaxDelay == 0 && config.RetryMultiplier == 0 {
		// Apply all retry defaults
		config.MaxRetries = 3
		config.RetryInitialDelay = 1 * time.Second
		config.RetryMaxDelay = 30 * time.Second
		config.RetryMultiplier = 2.0
	} else {
		// Retry config is being used, apply individual defaults for unset fields
		if config.RetryInitialDelay == 0 {
			config.RetryInitialDelay = 1 * time.Second
		}
		if config.RetryMaxDelay == 0 {
			config.RetryMaxDelay = 30 * time.Second
		}
		if config.RetryMultiplier == 0 {
			config.RetryMultiplier = 2.0
		}
		// Don't override MaxRetries if user set other retry fields
	}

	return &Orchestrator{
		config:            config,
		agents:            make([]agent.Agent, 0),
		messages:          make([]agent.Message, 0),
		rateLimiters:      make(map[string]*ratelimit.Limiter),
		middlewareChain:   middleware.NewChain(),
		writer:            writer,
		currentTurnNumber: 0,
	}
}

// SetLogger sets the chat logger for the orchestrator.
// The logger receives all conversation messages for persistence.
func (o *Orchestrator) SetLogger(logger *logger.ChatLogger) {
	o.logger = logger
}

// SetMetrics sets the Prometheus metrics for the orchestrator.
// If metrics are set, the orchestrator will record metrics for all operations.
// This method is thread-safe.
func (o *Orchestrator) SetMetrics(m *metrics.Metrics) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.metrics = m
}

// GetMetrics returns the current metrics instance.
// Returns nil if metrics are not enabled.
// This method is thread-safe.
func (o *Orchestrator) GetMetrics() *metrics.Metrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.metrics
}

// SetBridgeEmitter sets the streaming bridge emitter for real-time conversation updates.
// If set, the orchestrator will emit events for conversation lifecycle and messages.
// This method is thread-safe.
func (o *Orchestrator) SetBridgeEmitter(emitter bridge.BridgeEmitter) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.bridgeEmitter = emitter
}

// SetCommandInfo sets the command information for this conversation.
// This captures the agentpipe command that was executed.
// This method is thread-safe.
func (o *Orchestrator) SetCommandInfo(info *bridge.CommandInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.commandInfo = info
}

// emitConversationCompleted emits the conversation.completed event if bridge is enabled.
// This helper method calculates the conversation statistics and duration.
func (o *Orchestrator) emitConversationCompleted(status string, summary *bridge.SummaryMetadata) {
	o.mu.RLock()
	bridgeEmitter := o.bridgeEmitter
	messageCount := len(o.messages)
	startTime := o.conversationStart
	o.mu.RUnlock()

	if bridgeEmitter == nil {
		return
	}

	// Calculate total metrics from all messages
	totalTokens := 0
	totalCost := 0.0
	for _, msg := range o.getMessages() {
		if msg.Metrics != nil {
			totalTokens += msg.Metrics.TotalTokens
			totalCost += msg.Metrics.Cost
		}
	}

	// Add summary metrics to totals if summary was generated
	if summary != nil {
		totalTokens += summary.TotalTokens
		totalCost += summary.Cost
	}

	duration := time.Since(startTime)

	bridgeEmitter.EmitConversationCompleted(
		status,
		messageCount,
		o.currentTurnNumber,
		totalTokens,
		totalCost,
		duration,
		summary,
	)
}

// emitConversationError emits the conversation.error event if bridge is enabled.
func (o *Orchestrator) emitConversationError(errorMsg, errorType, agentType string) {
	o.mu.RLock()
	bridgeEmitter := o.bridgeEmitter
	o.mu.RUnlock()

	if bridgeEmitter != nil {
		bridgeEmitter.EmitConversationError(errorMsg, errorType, agentType)
	}
}

// parseDualSummary extracts short and full summaries from a structured response.
// Expected format:
//
//	SHORT: [1-2 sentence summary]
//	FULL: [detailed summary]
//
// Returns the extracted summaries or an error if parsing fails.
func parseDualSummary(response string) (shortText, fullText string, err error) {
	lines := strings.Split(response, "\n")
	var short, full strings.Builder

	inShort, inFull := false, false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for SHORT marker
		if strings.HasPrefix(trimmed, "SHORT:") {
			inShort = true
			inFull = false
			// Extract content after "SHORT:"
			content := strings.TrimSpace(strings.TrimPrefix(trimmed, "SHORT:"))
			if content != "" {
				short.WriteString(content)
			}
			continue
		}

		// Check for FULL marker
		if strings.HasPrefix(trimmed, "FULL:") {
			inFull = true
			inShort = false
			// Extract content after "FULL:"
			content := strings.TrimSpace(strings.TrimPrefix(trimmed, "FULL:"))
			if content != "" {
				full.WriteString(content)
			}
			continue
		}

		// Append to current section
		if inShort && trimmed != "" {
			if short.Len() > 0 {
				short.WriteString(" ")
			}
			short.WriteString(trimmed)
		} else if inFull && trimmed != "" {
			if full.Len() > 0 {
				full.WriteString(" ")
			}
			full.WriteString(trimmed)
		}
	}

	shortText = strings.TrimSpace(short.String())
	fullText = strings.TrimSpace(full.String())

	// Validation
	if shortText == "" || fullText == "" {
		return "", "", fmt.Errorf("failed to parse dual summary format: short=%d chars, full=%d chars", len(shortText), len(fullText))
	}

	return shortText, fullText, nil
}

// generateSummary generates a summary of the conversation using the configured summary agent.
// Returns nil if summary is disabled or if generation fails.
func (o *Orchestrator) generateSummary(ctx context.Context) *bridge.SummaryMetadata {
	// Check if summary is enabled
	if !o.config.Summary.Enabled {
		return nil
	}

	// Get conversation messages
	messages := o.getMessages()
	if len(messages) == 0 {
		return nil
	}

	// Build conversation text for summary
	var conversationText strings.Builder
	for _, msg := range messages {
		// Skip system messages
		if msg.Role == "system" {
			continue
		}
		conversationText.WriteString(fmt.Sprintf("%s: %s\n\n", msg.AgentName, msg.Content))
	}

	if conversationText.Len() == 0 {
		return nil
	}

	// Create summary prompt for dual summaries
	summaryPrompt := fmt.Sprintf(`Please provide two summaries of the following conversation:

1. SHORT SUMMARY (1-2 sentences): A brief, high-level overview capturing the main topic and outcome.
2. FULL SUMMARY: A comprehensive summary including key points, insights, and conclusions.

Format your response EXACTLY as follows:
SHORT: [your 1-2 sentence summary here]
FULL: [your detailed summary here]

Do not include meta-commentary about the conversation structure (e.g., "This is a conversation between agents").

Conversation:
%s`, conversationText.String())

	// Create a temporary agent for summary generation
	summaryAgent, err := agent.CreateAgent(agent.AgentConfig{
		ID:   "summary-agent",
		Type: o.config.Summary.Agent,
		Name: "Summary",
	})

	if err != nil || summaryAgent == nil {
		log.WithField("agent_type", o.config.Summary.Agent).WithError(err).Warn("failed to create summary agent")
		return nil
	}

	// Initialize the summary agent
	err = summaryAgent.Initialize(agent.AgentConfig{
		ID:   "summary-agent",
		Type: o.config.Summary.Agent,
		Name: "Summary",
	})
	if err != nil {
		log.WithError(err).Warn("failed to initialize summary agent")
		return nil
	}

	// Create summary messages
	summaryMessages := []agent.Message{
		{
			AgentID:   "system",
			AgentName: "SYSTEM",
			Content:   summaryPrompt,
			Timestamp: time.Now().Unix(),
			Role:      "user",
		},
	}

	// Generate summary with a timeout
	summaryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Calculate input tokens from conversation text
	inputTokens := utils.EstimateTokens(conversationText.String())

	startTime := time.Now()
	response, err := summaryAgent.SendMessage(summaryCtx, summaryMessages)
	duration := time.Since(startTime)

	if err != nil {
		log.WithError(err).Warn("failed to generate conversation summary")
		return nil
	}

	// Parse dual summary from response
	shortSummary, fullSummary, parseErr := parseDualSummary(response)
	if parseErr != nil {
		log.WithError(parseErr).Warn("failed to parse dual summary format, using fallback")
		// Fallback: use entire response as full summary, extract first 1-2 sentences for short
		fullSummary = strings.TrimSpace(response)
		sentences := strings.Split(fullSummary, ".")
		if len(sentences) >= 2 {
			shortSummary = strings.TrimSpace(sentences[0] + ". " + sentences[1] + ".")
		} else if len(sentences) == 1 {
			shortSummary = strings.TrimSpace(sentences[0])
			if !strings.HasSuffix(shortSummary, ".") {
				shortSummary += "."
			}
		} else {
			shortSummary = fullSummary
		}
	}

	// Calculate metrics
	outputTokens := utils.EstimateTokens(response)
	totalTokens := inputTokens + outputTokens
	model := summaryAgent.GetModel()
	cost := utils.EstimateCost(model, inputTokens, outputTokens)

	summaryMetadata := &bridge.SummaryMetadata{
		ShortText:    shortSummary,
		Text:         fullSummary,
		AgentType:    o.config.Summary.Agent,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		Cost:         cost,
		DurationMs:   duration.Milliseconds(),
	}

	// Store summary in orchestrator for later access
	o.mu.Lock()
	o.summary = summaryMetadata
	o.mu.Unlock()

	return summaryMetadata
}

// AddMiddleware adds a middleware to the orchestrator's processing chain.
// Middleware is executed in the order it is added (first added = first executed).
// This method is thread-safe.
func (o *Orchestrator) AddMiddleware(m middleware.Middleware) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.middlewareChain.Add(m)

	log.WithField("middleware", m.Name()).Debug("middleware added to orchestrator")
}

// SetupDefaultMiddleware configures a sensible default middleware chain.
// This includes logging, metrics, validation, and error recovery.
func (o *Orchestrator) SetupDefaultMiddleware() {
	o.AddMiddleware(middleware.ErrorRecoveryMiddleware())
	o.AddMiddleware(middleware.LoggingMiddleware())
	o.AddMiddleware(middleware.MetricsMiddleware())
	o.AddMiddleware(middleware.EmptyContentValidationMiddleware())
	o.AddMiddleware(middleware.SanitizationMiddleware(false))
}

// AddAgent registers an agent with the orchestrator.
// The agent's announcement is added to the conversation history and logged.
// A rate limiter is created for the agent based on its configuration.
// This method is thread-safe.
func (o *Orchestrator) AddAgent(a agent.Agent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agents = append(o.agents, a)

	// Create rate limiter for this agent
	rateLimit := a.GetRateLimit()
	rateLimitBurst := a.GetRateLimitBurst()
	o.rateLimiters[a.GetID()] = ratelimit.NewLimiter(rateLimit, rateLimitBurst)

	log.WithFields(map[string]interface{}{
		"agent_id":   a.GetID(),
		"agent_name": a.GetName(),
		"agent_type": a.GetType(),
		"rate_limit": rateLimit,
		"burst":      rateLimitBurst,
	}).Info("agent added to orchestrator")

	announcement := agent.Message{
		AgentID:   a.GetID(),
		AgentName: a.GetName(),
		AgentType: a.GetType(),
		Content:   a.Announce(),
		Timestamp: time.Now().Unix(),
		Role:      "system",
	}
	o.messages = append(o.messages, announcement)

	// Log using the logger if available
	if o.logger != nil {
		o.logger.LogMessage(announcement)
	}
	// Always write to writer if available (for TUI)
	if o.writer != nil {
		fmt.Fprintf(o.writer, "\n[System] %s\n", announcement.Content)
	}
}

// Start begins the multi-agent conversation using the configured orchestration mode.
// It returns an error if no agents are registered or if the orchestration mode is invalid.
// The conversation continues until MaxTurns is reached, the context is canceled, or an error occurs.
// This method blocks until the conversation completes.
func (o *Orchestrator) Start(ctx context.Context) error {
	if len(o.agents) == 0 {
		log.Error("conversation start failed: no agents configured")
		return fmt.Errorf("no agents configured")
	}

	// Increment active conversations metric
	if o.metrics != nil {
		o.metrics.IncrementActiveConversations()
		defer o.metrics.DecrementActiveConversations()
	}

	log.WithFields(map[string]interface{}{
		"mode":       o.config.Mode,
		"max_turns":  o.config.MaxTurns,
		"agents":     len(o.agents),
		"has_prompt": o.config.InitialPrompt != "",
	}).Info("starting conversation")

	// Record conversation start time for duration tracking
	o.conversationStart = time.Now()

	// Track return error to determine status
	var runErr error

	// Emit conversation.completed and close bridge when function returns
	defer func() {
		// Determine status based on context cancellation or error
		status := "completed"

		// Check if context was canceled
		select {
		case <-ctx.Done():
			status = "interrupted"
		default:
			// Also check if the error indicates cancellation
			if runErr != nil && (errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded)) {
				status = "interrupted"
			}
		}

		// Generate summary if enabled
		// Use background context since original ctx may be canceled
		summary := o.generateSummary(context.Background())

		o.emitConversationCompleted(status, summary)

		// Close bridge emitter to flush events and close event store
		o.mu.RLock()
		bridgeEmitter := o.bridgeEmitter
		o.mu.RUnlock()
		if bridgeEmitter != nil {
			_ = bridgeEmitter.Close()
		}
	}()

	// Emit conversation.started event if bridge is enabled
	o.mu.RLock()
	bridgeEmitter := o.bridgeEmitter
	o.mu.RUnlock()

	if bridgeEmitter != nil {
		// Build agent participants list
		participants := make([]bridge.AgentParticipant, 0, len(o.agents))
		for _, a := range o.agents {
			participants = append(participants, bridge.AgentParticipant{
				AgentID:    a.GetID(),
				AgentType:  a.GetType(),
				Model:      a.GetModel(),
				Name:       a.GetName(),
				Prompt:     a.GetPrompt(),
				CLIVersion: a.GetCLIVersion(),
			})
		}

		bridgeEmitter.EmitConversationStarted(
			string(o.config.Mode),
			o.config.InitialPrompt,
			o.config.MaxTurns,
			participants,
			o.commandInfo,
		)
	}

	if o.config.InitialPrompt != "" {
		initialMsg := agent.Message{
			AgentID:   "host",
			AgentName: "HOST",
			Content:   o.config.InitialPrompt,
			Timestamp: time.Now().Unix(),
			Role:      "system",
		}
		o.mu.Lock()
		o.messages = append(o.messages, initialMsg)
		o.mu.Unlock()

		// Log using the logger if available
		if o.logger != nil {
			o.logger.LogMessage(initialMsg)
		}
		// Always write to writer if available (for TUI)
		if o.writer != nil {
			fmt.Fprintf(o.writer, "\n[HOST] %s\n", initialMsg.Content)
		}
	}

	switch o.config.Mode {
	case ModeRoundRobin:
		runErr = o.runRoundRobin(ctx)
		return runErr
	case ModeReactive:
		runErr = o.runReactive(ctx)
		return runErr
	case ModeFreeForm:
		runErr = o.runFreeForm(ctx)
		return runErr
	default:
		log.WithField("mode", o.config.Mode).Error("unknown conversation mode")
		errMsg := fmt.Sprintf("unknown conversation mode: %s", o.config.Mode)
		o.emitConversationError(errMsg, "configuration", "orchestrator")
		runErr = fmt.Errorf("unknown conversation mode: %s", o.config.Mode)
		return runErr
	}
}

func (o *Orchestrator) runRoundRobin(ctx context.Context) error {
	turns := 0
	agentIndex := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if o.config.MaxTurns > 0 && turns >= o.config.MaxTurns {
			endMsg := "Maximum turns reached. Conversation ended."
			if o.logger != nil {
				o.logger.LogSystem(endMsg)
			}
			if o.writer != nil {
				fmt.Fprintln(o.writer, "\n[System] "+endMsg)
			}
			break
		}

		currentAgent := o.agents[agentIndex]

		if err := o.getAgentResponse(ctx, currentAgent); err != nil {
			if o.logger != nil {
				o.logger.LogError(currentAgent.GetName(), err)
				o.logger.LogSystem("Continuing conversation with remaining agents...")
			}
			if o.writer != nil {
				fmt.Fprintf(o.writer, "\n[Error] Agent %s failed: %v\n", currentAgent.GetName(), err)
				fmt.Fprintf(o.writer, "[Info] Continuing conversation with remaining agents...\n")
			}
		}

		time.Sleep(o.config.ResponseDelay)

		agentIndex = (agentIndex + 1) % len(o.agents)
		if agentIndex == 0 {
			turns++
		}
	}

	return nil
}

func (o *Orchestrator) runReactive(ctx context.Context) error {
	turns := 0
	lastSpeaker := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if o.config.MaxTurns > 0 && turns >= o.config.MaxTurns {
			endMsg := "Maximum turns reached. Conversation ended."
			if o.logger != nil {
				o.logger.LogSystem(endMsg)
			}
			if o.writer != nil {
				fmt.Fprintln(o.writer, "\n[System] "+endMsg)
			}
			break
		}

		nextAgent := o.selectNextAgent(lastSpeaker)
		if nextAgent == nil {
			time.Sleep(o.config.ResponseDelay)
			continue
		}

		if err := o.getAgentResponse(ctx, nextAgent); err != nil {
			if o.writer != nil {
				fmt.Fprintf(o.writer, "\n[Error] Agent %s failed: %v\n", nextAgent.GetName(), err)
			}
		} else {
			lastSpeaker = nextAgent.GetID()
			turns++
		}

		time.Sleep(o.config.ResponseDelay)
	}

	return nil
}

func (o *Orchestrator) runFreeForm(ctx context.Context) error {
	turns := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if o.config.MaxTurns > 0 && turns >= o.config.MaxTurns {
			endMsg := "Maximum turns reached. Conversation ended."
			if o.logger != nil {
				o.logger.LogSystem(endMsg)
			}
			if o.writer != nil {
				fmt.Fprintln(o.writer, "\n[System] "+endMsg)
			}
			break
		}

		for _, a := range o.agents {
			if shouldRespond(o.getMessages(), a) {
				if err := o.getAgentResponse(ctx, a); err != nil {
					if o.writer != nil {
						fmt.Fprintf(o.writer, "\n[Error] Agent %s failed: %v\n", a.GetName(), err)
					}
				} else {
					turns++
				}
				time.Sleep(o.config.ResponseDelay)
			}
		}
	}

	return nil
}

func (o *Orchestrator) getAgentResponse(ctx context.Context, a agent.Agent) error {
	// Apply rate limiting before attempting to get response
	o.mu.RLock()
	limiter := o.rateLimiters[a.GetID()]
	o.mu.RUnlock()

	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			// Record rate limit hit metric
			if o.metrics != nil {
				o.metrics.RecordRateLimitHit(a.GetName())
			}

			log.WithFields(map[string]interface{}{
				"agent_id":   a.GetID(),
				"agent_name": a.GetName(),
			}).WithError(err).Error("rate limit wait failed")
			return fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	messages := o.getMessages()

	// Calculate input tokens from conversation history (once, outside retry loop)
	var inputBuilder strings.Builder
	for _, msg := range messages {
		inputBuilder.WriteString(msg.Content)
		inputBuilder.WriteString(" ")
	}
	inputTokens := utils.EstimateTokens(inputBuilder.String())

	log.WithFields(map[string]interface{}{
		"agent_id":     a.GetID(),
		"agent_name":   a.GetName(),
		"input_tokens": inputTokens,
		"max_retries":  o.config.MaxRetries,
	}).Debug("requesting agent response")

	// Retry loop with exponential backoff
	var lastErr error
	var response string
	var startTime time.Time

	for attempt := 0; attempt <= o.config.MaxRetries; attempt++ {
		// Apply exponential backoff delay before retry (skip on first attempt)
		if attempt > 0 {
			// Record retry attempt metric
			if o.metrics != nil {
				o.metrics.RecordRetryAttempt(a.GetName(), a.GetType())
			}

			delay := o.calculateBackoffDelay(attempt)
			log.WithFields(map[string]interface{}{
				"agent_name":  a.GetName(),
				"attempt":     attempt,
				"max_retries": o.config.MaxRetries,
				"delay":       delay.String(),
			}).Warn("retrying agent request after failure")
			if o.writer != nil {
				fmt.Fprintf(o.writer, "[Retry] Waiting %v before retry %d/%d for %s...\n",
					delay, attempt, o.config.MaxRetries, a.GetName())
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, o.config.TurnTimeout)
		startTime = time.Now()

		// Attempt to get response
		response, lastErr = a.SendMessage(timeoutCtx, messages)
		cancel()

		if lastErr == nil {
			// Success! Break out of retry loop
			log.WithFields(map[string]interface{}{
				"agent_name": a.GetName(),
				"attempt":    attempt + 1,
				"duration":   time.Since(startTime).String(),
			}).Debug("agent response received")
			break
		}

		// Log retry attempt
		if o.logger != nil {
			o.logger.LogError(a.GetName(), fmt.Errorf("attempt %d/%d failed: %w", attempt+1, o.config.MaxRetries+1, lastErr))
		}
		if o.writer != nil && attempt < o.config.MaxRetries {
			fmt.Fprintf(o.writer, "[Error] Agent %s attempt %d/%d failed: %v\n",
				a.GetName(), attempt+1, o.config.MaxRetries+1, lastErr)
		}

		log.WithFields(map[string]interface{}{
			"agent_name":  a.GetName(),
			"attempt":     attempt + 1,
			"max_retries": o.config.MaxRetries + 1,
		}).WithError(lastErr).Warn("agent request attempt failed")
	}

	// If all retries failed, return the last error
	if lastErr != nil {
		log.WithFields(map[string]interface{}{
			"agent_name": a.GetName(),
			"attempts":   o.config.MaxRetries + 1,
		}).WithError(lastErr).Error("all agent request attempts failed")

		// Determine error type
		errorType := "unknown"
		if strings.Contains(lastErr.Error(), "timeout") || strings.Contains(lastErr.Error(), "deadline") {
			errorType = "timeout"
		} else if strings.Contains(lastErr.Error(), "rate limit") {
			errorType = "rate_limit"
		}

		// Record error metric
		if o.metrics != nil {
			o.metrics.RecordAgentError(a.GetName(), a.GetType(), errorType)
			o.metrics.RecordAgentRequest(a.GetName(), a.GetType(), "error")
		}

		// Emit conversation.error event
		o.emitConversationError(lastErr.Error(), errorType, a.GetType())

		return lastErr
	}

	// Calculate metrics
	duration := time.Since(startTime)
	outputTokens := utils.EstimateTokens(response)
	totalTokens := inputTokens + outputTokens

	// Get model from agent
	model := a.GetModel()

	// Calculate estimated cost
	cost := utils.EstimateCost(model, inputTokens, outputTokens)

	log.WithFields(map[string]interface{}{
		"agent_name":    a.GetName(),
		"model":         model,
		"duration_ms":   duration.Milliseconds(),
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"total_tokens":  totalTokens,
		"cost":          cost,
	}).Info("agent response successful")

	// Record metrics
	if o.metrics != nil {
		o.metrics.RecordAgentRequest(a.GetName(), a.GetType(), "success")
		o.metrics.RecordAgentDuration(a.GetName(), a.GetType(), duration.Seconds())
		o.metrics.RecordAgentTokens(a.GetName(), a.GetType(), "input", inputTokens)
		o.metrics.RecordAgentTokens(a.GetName(), a.GetType(), "output", outputTokens)
		o.metrics.RecordAgentCost(a.GetName(), a.GetType(), model, cost)
		o.metrics.RecordMessageSize(a.GetName(), "input", len(inputBuilder.String()))
		o.metrics.RecordMessageSize(a.GetName(), "output", len(response))
		o.metrics.RecordConversationTurn(string(o.config.Mode))
	}

	// Store the message in history with metrics
	msg := agent.Message{
		AgentID:   a.GetID(),
		AgentName: a.GetName(),
		AgentType: a.GetType(),
		Content:   response,
		Timestamp: time.Now().Unix(),
		Role:      "agent",
		Metrics: &agent.ResponseMetrics{
			Duration:     duration,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			Model:        model,
			Cost:         cost,
		},
	}

	// Process message through middleware chain
	o.mu.RLock()
	chain := o.middlewareChain
	turnNumber := o.currentTurnNumber
	o.mu.RUnlock()

	if chain != nil && chain.Len() > 0 {
		middlewareCtx := &middleware.MessageContext{
			Ctx:        ctx,
			AgentID:    a.GetID(),
			AgentName:  a.GetName(),
			TurnNumber: turnNumber,
			Metadata:   make(map[string]interface{}),
		}

		processedMsg, err := chain.Process(middlewareCtx, &msg)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"agent_name": a.GetName(),
				"turn":       turnNumber,
			}).WithError(err).Error("middleware processing failed")
			return fmt.Errorf("middleware processing failed: %w", err)
		}

		// Use the processed message
		if processedMsg != nil {
			msg = *processedMsg
		}
	}

	o.mu.Lock()
	o.messages = append(o.messages, msg)
	currentTurn := o.currentTurnNumber
	o.currentTurnNumber++
	bridgeEmitter := o.bridgeEmitter
	o.mu.Unlock()

	// Emit message.created event if bridge is enabled
	if bridgeEmitter != nil {
		bridgeEmitter.EmitMessageCreated(
			a.GetID(),
			a.GetType(),
			a.GetName(),
			response,
			model,
			currentTurn,
			totalTokens,
			inputTokens,
			outputTokens,
			cost,
			duration,
		)
	}

	// Display the response
	if o.logger != nil {
		o.logger.LogMessage(msg)
	}
	// Always write to writer if available (for TUI)
	if o.writer != nil {
		// Include metrics in a special format if available
		if msg.Metrics != nil {
			fmt.Fprintf(o.writer, "\n[%s|%dms|%dt|%.4f] %s\n",
				a.GetName(),
				msg.Metrics.Duration.Milliseconds(),
				msg.Metrics.TotalTokens,
				msg.Metrics.Cost,
				response)
		} else {
			fmt.Fprintf(o.writer, "\n[%s] %s\n", a.GetName(), response)
		}
	}

	return nil
}

// calculateBackoffDelay computes the delay for the given retry attempt using exponential backoff.
// The delay grows exponentially: InitialDelay * (Multiplier ^ attempt), capped at MaxDelay.
func (o *Orchestrator) calculateBackoffDelay(attempt int) time.Duration {
	// Calculate exponential backoff: initialDelay * multiplier^attempt
	delay := float64(o.config.RetryInitialDelay) * math.Pow(o.config.RetryMultiplier, float64(attempt))

	// Cap at maximum delay
	if delay > float64(o.config.RetryMaxDelay) {
		delay = float64(o.config.RetryMaxDelay)
	}

	return time.Duration(delay)
}

func (o *Orchestrator) getMessages() []agent.Message {
	o.mu.RLock()
	defer o.mu.RUnlock()

	messages := make([]agent.Message, len(o.messages))
	copy(messages, o.messages)
	return messages
}

func (o *Orchestrator) selectNextAgent(lastSpeaker string) agent.Agent {
	// Count available agents (excluding last speaker)
	availableCount := 0
	for _, a := range o.agents {
		if a.GetID() != lastSpeaker {
			availableCount++
		}
	}

	if availableCount == 0 {
		return nil
	}

	// Select a random index among available agents
	targetIndex := rand.Intn(availableCount)

	// Find the agent at that index
	currentIndex := 0
	for _, a := range o.agents {
		if a.GetID() != lastSpeaker {
			if currentIndex == targetIndex {
				return a
			}
			currentIndex++
		}
	}

	return nil
}

func shouldRespond(messages []agent.Message, a agent.Agent) bool {
	if len(messages) == 0 {
		return true
	}

	lastMessage := messages[len(messages)-1]
	return lastMessage.AgentID != a.GetID()
}

// GetMessages returns a copy of all messages in the conversation history.
// The returned slice is a copy and can be safely modified without affecting the orchestrator's state.
// This method is thread-safe.
func (o *Orchestrator) GetMessages() []agent.Message {
	return o.getMessages()
}

// GetSummary returns the conversation summary if one was generated.
// Returns nil if summary generation was disabled or hasn't been completed yet.
// This method is thread-safe.
func (o *Orchestrator) GetSummary() *bridge.SummaryMetadata {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.summary
}
