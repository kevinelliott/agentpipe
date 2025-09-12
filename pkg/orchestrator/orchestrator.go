package orchestrator

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/logger"
	"github.com/kevinelliott/agentpipe/pkg/utils"
)

type ConversationMode string

const (
	ModeRoundRobin ConversationMode = "round-robin"
	ModeReactive   ConversationMode = "reactive"
	ModeFreeForm   ConversationMode = "free-form"
)

type OrchestratorConfig struct {
	Mode          ConversationMode
	TurnTimeout   time.Duration
	MaxTurns      int
	ResponseDelay time.Duration
	InitialPrompt string
}

type Orchestrator struct {
	config   OrchestratorConfig
	agents   []agent.Agent
	messages []agent.Message
	mu       sync.RWMutex
	writer   io.Writer
	logger   *logger.ChatLogger
}

func NewOrchestrator(config OrchestratorConfig, writer io.Writer) *Orchestrator {
	if config.TurnTimeout == 0 {
		config.TurnTimeout = 30 * time.Second
	}
	if config.ResponseDelay == 0 {
		config.ResponseDelay = 1 * time.Second
	}

	return &Orchestrator{
		config:   config,
		agents:   make([]agent.Agent, 0),
		messages: make([]agent.Message, 0),
		writer:   writer,
	}
}

func (o *Orchestrator) SetLogger(logger *logger.ChatLogger) {
	o.logger = logger
}

func (o *Orchestrator) AddAgent(a agent.Agent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agents = append(o.agents, a)

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

func (o *Orchestrator) Start(ctx context.Context) error {
	if len(o.agents) == 0 {
		return fmt.Errorf("no agents configured")
	}

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
	messages := o.getMessages()

	timeoutCtx, cancel := context.WithTimeout(ctx, o.config.TurnTimeout)
	defer cancel()

	// Track timing
	startTime := time.Now()

	// Calculate input tokens from conversation history
	inputText := ""
	for _, msg := range messages {
		inputText += msg.Content + " "
	}
	inputTokens := utils.EstimateTokens(inputText)

	// Get the response
	response, err := a.SendMessage(timeoutCtx, messages)
	if err != nil {
		return err
	}

	// Calculate metrics
	duration := time.Since(startTime)
	outputTokens := utils.EstimateTokens(response)
	totalTokens := inputTokens + outputTokens

	// Get model from agent config (if available)
	model := a.GetType() // Default to type, but ideally get from config

	// Calculate estimated cost
	cost := utils.EstimateCost(model, inputTokens, outputTokens)

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

func (o *Orchestrator) getMessages() []agent.Message {
	o.mu.RLock()
	defer o.mu.RUnlock()

	messages := make([]agent.Message, len(o.messages))
	copy(messages, o.messages)
	return messages
}

func (o *Orchestrator) selectNextAgent(lastSpeaker string) agent.Agent {
	availableAgents := make([]agent.Agent, 0)
	for _, a := range o.agents {
		if a.GetID() != lastSpeaker {
			availableAgents = append(availableAgents, a)
		}
	}

	if len(availableAgents) == 0 {
		return nil
	}

	return availableAgents[time.Now().UnixNano()%int64(len(availableAgents))]
}

func shouldRespond(messages []agent.Message, a agent.Agent) bool {
	if len(messages) == 0 {
		return true
	}

	lastMessage := messages[len(messages)-1]
	return lastMessage.AgentID != a.GetID()
}

func (o *Orchestrator) GetMessages() []agent.Message {
	return o.getMessages()
}
