// Package orchestrator manages multi-agent conversations with different orchestration modes.
// It coordinates agent interactions, handles turn-taking, and manages message history.
package orchestrator

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/logger"
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
}

// Orchestrator coordinates multi-agent conversations.
// It manages agent registration, turn-taking, message history, and logging.
// All methods are safe for concurrent use.
type Orchestrator struct {
	config       OrchestratorConfig
	agents       []agent.Agent
	messages     []agent.Message
	rateLimiters map[string]*ratelimit.Limiter // per-agent rate limiters
	mu           sync.RWMutex
	writer       io.Writer
	logger       *logger.ChatLogger
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
		config:       config,
		agents:       make([]agent.Agent, 0),
		messages:     make([]agent.Message, 0),
		rateLimiters: make(map[string]*ratelimit.Limiter),
		writer:       writer,
	}
}

// SetLogger sets the chat logger for the orchestrator.
// The logger receives all conversation messages for persistence.
func (o *Orchestrator) SetLogger(logger *logger.ChatLogger) {
	o.logger = logger
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

	log.WithFields(map[string]interface{}{
		"mode":       o.config.Mode,
		"max_turns":  o.config.MaxTurns,
		"agents":     len(o.agents),
		"has_prompt": o.config.InitialPrompt != "",
	}).Info("starting conversation")

	if o.config.InitialPrompt != "" {
		initialMsg := agent.Message{
			AgentID:   "system",
			AgentName: "System",
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
			fmt.Fprintf(o.writer, "\n[System] %s\n", initialMsg.Content)
		}
	}

	switch o.config.Mode {
	case ModeRoundRobin:
		return o.runRoundRobin(ctx)
	case ModeReactive:
		return o.runReactive(ctx)
	case ModeFreeForm:
		return o.runFreeForm(ctx)
	default:
		log.WithField("mode", o.config.Mode).Error("unknown conversation mode")
		return fmt.Errorf("unknown conversation mode: %s", o.config.Mode)
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
			delay := o.calculateBackoffDelay(attempt)
			log.WithFields(map[string]interface{}{
				"agent_name": a.GetName(),
				"attempt":    attempt,
				"max_retries": o.config.MaxRetries,
				"delay":      delay.String(),
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
			"agent_name": a.GetName(),
			"attempt":    attempt + 1,
			"max_retries": o.config.MaxRetries + 1,
		}).WithError(lastErr).Warn("agent request attempt failed")
	}

	// If all retries failed, return the last error
	if lastErr != nil {
		log.WithFields(map[string]interface{}{
			"agent_name": a.GetName(),
			"attempts":   o.config.MaxRetries + 1,
		}).WithError(lastErr).Error("all agent request attempts failed")
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

	// Store the message in history with metrics
	msg := agent.Message{
		AgentID:   a.GetID(),
		AgentName: a.GetName(),
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

	o.mu.Lock()
	o.messages = append(o.messages, msg)
	o.mu.Unlock()

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
