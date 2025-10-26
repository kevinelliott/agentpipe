package adapters

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/internal/providers"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/client"
	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/utils"
)

// OpenRouterAgent is an API-based agent that uses OpenRouter's unified API.
type OpenRouterAgent struct {
	agent.BaseAgent
	client *client.OpenAICompatClient
	apiKey string
}

// NewOpenRouterAgent creates a new OpenRouter agent instance.
func NewOpenRouterAgent() agent.Agent {
	return &OpenRouterAgent{}
}

// Initialize configures the OpenRouter agent with the provided configuration.
func (o *OpenRouterAgent) Initialize(config agent.AgentConfig) error {
	if err := o.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("openrouter agent base initialization failed")
		return err
	}

	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.WithFields(map[string]interface{}{
			"agent_id":   o.ID,
			"agent_name": o.Name,
		}).Error("OPENROUTER_API_KEY environment variable not set")
		return fmt.Errorf("OPENROUTER_API_KEY environment variable is required")
	}
	o.apiKey = apiKey

	// Validate model is configured
	if o.Config.Model == "" {
		log.WithFields(map[string]interface{}{
			"agent_id":   o.ID,
			"agent_name": o.Name,
		}).Error("model not specified in configuration")
		return fmt.Errorf("model must be specified for OpenRouter agent")
	}

	// Verify model exists in provider registry
	registry := providers.GetRegistry()
	modelInfo, provider, err := registry.GetModel(o.Config.Model)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   o.ID,
			"agent_name": o.Name,
			"model":      o.Config.Model,
		}).Warn("model not found in provider registry (cost estimates may be inaccurate)")
	} else {
		log.WithFields(map[string]interface{}{
			"agent_id":   o.ID,
			"agent_name": o.Name,
			"model":      modelInfo.ID,
			"provider":   provider.Name,
		}).Debug("model found in provider registry")
	}

	// Create HTTP client
	o.client = client.NewOpenAICompatClient("https://openrouter.ai/api/v1", apiKey)

	log.WithFields(map[string]interface{}{
		"agent_id":   o.ID,
		"agent_name": o.Name,
		"model":      o.Config.Model,
	}).Info("openrouter agent initialized successfully")

	return nil
}

// IsAvailable checks if the OpenRouter API is available (API key is set).
func (o *OpenRouterAgent) IsAvailable() bool {
	return os.Getenv("OPENROUTER_API_KEY") != ""
}

// GetCLIVersion returns a version string indicating this is an API-based agent.
func (o *OpenRouterAgent) GetCLIVersion() string {
	return "N/A (API)"
}

// HealthCheck performs a health check by making a test API request.
func (o *OpenRouterAgent) HealthCheck(ctx context.Context) error {
	if o.client == nil {
		log.WithField("agent_name", o.Name).Error("openrouter health check failed: not initialized")
		return fmt.Errorf("openrouter agent not initialized")
	}

	log.WithField("agent_name", o.Name).Debug("starting openrouter health check")

	// Make a minimal test request
	req := client.ChatCompletionRequest{
		Model: o.Config.Model,
		Messages: []client.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
		MaxTokens: intPtr(1),
	}

	_, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.WithField("agent_name", o.Name).WithError(err).Error("openrouter health check failed")
		return fmt.Errorf("openrouter API health check failed: %w", err)
	}

	log.WithField("agent_name", o.Name).Info("openrouter health check passed")
	return nil
}

// SendMessage sends a message to OpenRouter and returns the response.
func (o *OpenRouterAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    o.Name,
		"message_count": len(messages),
		"model":         o.Config.Model,
	}).Debug("sending message to openrouter")

	// Build conversation history
	apiMessages := o.buildConversationHistory(messages)

	// Build request
	req := client.ChatCompletionRequest{
		Model:    o.Config.Model,
		Messages: apiMessages,
	}

	if o.Config.Temperature > 0 {
		req.Temperature = &o.Config.Temperature
	}

	if o.Config.MaxTokens > 0 {
		req.MaxTokens = &o.Config.MaxTokens
	}

	// Send request
	startTime := time.Now()
	resp, err := o.client.CreateChatCompletion(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_name": o.Name,
			"duration":   duration.String(),
			"model":      o.Config.Model,
		}).WithError(err).Error("openrouter request failed")
		return "", fmt.Errorf("openrouter request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		log.WithField("agent_name", o.Name).Error("openrouter returned no choices")
		return "", fmt.Errorf("no response from openrouter")
	}

	content := resp.Choices[0].Message.Content

	// Log metrics
	if resp.Usage != nil {
		cost := utils.EstimateCost(o.Config.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		log.WithFields(map[string]interface{}{
			"agent_name":        o.Name,
			"duration":          duration.String(),
			"model":             resp.Model,
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
			"cost":              fmt.Sprintf("$%.4f", cost),
		}).Info("openrouter message sent successfully")
	} else {
		log.WithFields(map[string]interface{}{
			"agent_name": o.Name,
			"duration":   duration.String(),
			"model":      resp.Model,
		}).Info("openrouter message sent successfully")
	}

	return strings.TrimSpace(content), nil
}

// StreamMessage sends a message to OpenRouter and streams the response.
func (o *OpenRouterAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    o.Name,
		"message_count": len(messages),
		"model":         o.Config.Model,
	}).Debug("starting openrouter streaming message")

	// Build conversation history
	apiMessages := o.buildConversationHistory(messages)

	// Build request
	req := client.ChatCompletionRequest{
		Model:    o.Config.Model,
		Messages: apiMessages,
	}

	if o.Config.Temperature > 0 {
		req.Temperature = &o.Config.Temperature
	}

	if o.Config.MaxTokens > 0 {
		req.MaxTokens = &o.Config.MaxTokens
	}

	// Send streaming request
	startTime := time.Now()
	usage, err := o.client.CreateChatCompletionStream(ctx, req, writer)
	duration := time.Since(startTime)

	if err != nil {
		log.WithField("agent_name", o.Name).WithError(err).Error("openrouter streaming failed")
		return fmt.Errorf("openrouter streaming failed: %w", err)
	}

	// Log metrics
	if usage != nil {
		cost := utils.EstimateCost(o.Config.Model, usage.PromptTokens, usage.CompletionTokens)
		log.WithFields(map[string]interface{}{
			"agent_name":        o.Name,
			"duration":          duration.String(),
			"model":             o.Config.Model,
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
			"cost":              fmt.Sprintf("$%.4f", cost),
		}).Info("openrouter streaming message completed")
	} else {
		log.WithFields(map[string]interface{}{
			"agent_name": o.Name,
			"duration":   duration.String(),
			"model":      o.Config.Model,
		}).Info("openrouter streaming message completed")
	}

	return nil
}

// buildConversationHistory converts AgentPipe messages to OpenAI API format.
func (o *OpenRouterAgent) buildConversationHistory(messages []agent.Message) []client.ChatCompletionMessage {
	apiMessages := make([]client.ChatCompletionMessage, 0)

	// Add system prompt if configured
	if o.Config.Prompt != "" {
		apiMessages = append(apiMessages, client.ChatCompletionMessage{
			Role:    "system",
			Content: o.Config.Prompt,
		})
	}

	// Convert conversation messages
	for _, msg := range messages {
		// Skip this agent's own messages to avoid confusion
		if msg.AgentName == o.Name || msg.AgentID == o.ID {
			continue
		}

		var role string
		var content string

		switch msg.Role {
		case "system":
			// System messages (orchestrator prompts, announcements)
			role = "user" // Most APIs don't support multiple system messages, so use user role
			content = fmt.Sprintf("[System] %s", msg.Content)

		case "user":
			role = "user"
			content = msg.Content

		case "agent":
			role = "user" // Treat other agents' messages as user messages
			content = fmt.Sprintf("%s: %s", msg.AgentName, msg.Content)

		default:
			// Unknown role, skip
			continue
		}

		apiMessages = append(apiMessages, client.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}

	return apiMessages
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int {
	return &i
}

func init() {
	agent.RegisterFactory("openrouter", NewOpenRouterAgent)
}
