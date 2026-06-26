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

// RequestyAgent is an API-based agent that uses Requesty's unified API.
type RequestyAgent struct {
	agent.BaseAgent
	client *client.OpenAICompatClient
	apiKey string
}

// NewRequestyAgent creates a new Requesty agent instance.
func NewRequestyAgent() agent.Agent {
	return &RequestyAgent{}
}

// Initialize configures the Requesty agent with the provided configuration.
func (r *RequestyAgent) Initialize(config agent.AgentConfig) error {
	if err := r.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("requesty agent base initialization failed")
		return err
	}

	// Get API key from environment
	apiKey := os.Getenv("REQUESTY_API_KEY")
	if apiKey == "" {
		log.WithFields(map[string]interface{}{
			"agent_id":   r.ID,
			"agent_name": r.Name,
		}).Error("REQUESTY_API_KEY environment variable not set")
		return fmt.Errorf("REQUESTY_API_KEY environment variable is required")
	}
	r.apiKey = apiKey

	// Validate model is configured
	if r.Config.Model == "" {
		log.WithFields(map[string]interface{}{
			"agent_id":   r.ID,
			"agent_name": r.Name,
		}).Error("model not specified in configuration")
		return fmt.Errorf("model must be specified for Requesty agent")
	}

	// Verify model exists in provider registry
	registry := providers.GetRegistry()
	modelInfo, provider, err := registry.GetModel(r.Config.Model)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   r.ID,
			"agent_name": r.Name,
			"model":      r.Config.Model,
		}).Warn("model not found in provider registry (cost estimates may be inaccurate)")
	} else {
		log.WithFields(map[string]interface{}{
			"agent_id":   r.ID,
			"agent_name": r.Name,
			"model":      modelInfo.ID,
			"provider":   provider.Name,
		}).Debug("model found in provider registry")
	}

	// Create HTTP client
	r.client = client.NewOpenAICompatClient("https://router.requesty.ai/v1", apiKey)

	log.WithFields(map[string]interface{}{
		"agent_id":   r.ID,
		"agent_name": r.Name,
		"model":      r.Config.Model,
	}).Info("requesty agent initialized successfully")

	return nil
}

// IsAvailable checks if the Requesty API is available (API key is set).
func (r *RequestyAgent) IsAvailable() bool {
	return os.Getenv("REQUESTY_API_KEY") != ""
}

// GetCLIVersion returns a version string indicating this is an API-based agent.
func (r *RequestyAgent) GetCLIVersion() string {
	return "N/A (API)"
}

// HealthCheck performs a health check by making a test API request.
func (r *RequestyAgent) HealthCheck(ctx context.Context) error {
	if r.client == nil {
		log.WithField("agent_name", r.Name).Error("requesty health check failed: not initialized")
		return fmt.Errorf("requesty agent not initialized")
	}

	log.WithField("agent_name", r.Name).Debug("starting requesty health check")

	// Make a minimal test request
	req := client.ChatCompletionRequest{
		Model: r.Config.Model,
		Messages: []client.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
		MaxTokens: intPtr(1),
	}

	_, err := r.client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.WithField("agent_name", r.Name).WithError(err).Error("requesty health check failed")
		return fmt.Errorf("requesty API health check failed: %w", err)
	}

	log.WithField("agent_name", r.Name).Info("requesty health check passed")
	return nil
}

// SendMessage sends a message to Requesty and returns the response.
func (r *RequestyAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    r.Name,
		"message_count": len(messages),
		"model":         r.Config.Model,
	}).Debug("sending message to requesty")

	// Build conversation history
	apiMessages := r.buildConversationHistory(messages)

	// Build request
	req := client.ChatCompletionRequest{
		Model:    r.Config.Model,
		Messages: apiMessages,
	}

	if r.Config.Temperature > 0 {
		req.Temperature = &r.Config.Temperature
	}

	if r.Config.MaxTokens > 0 {
		req.MaxTokens = &r.Config.MaxTokens
	}

	// Send request
	startTime := time.Now()
	resp, err := r.client.CreateChatCompletion(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_name": r.Name,
			"duration":   duration.String(),
			"model":      r.Config.Model,
		}).WithError(err).Error("requesty request failed")
		return "", fmt.Errorf("requesty request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		log.WithField("agent_name", r.Name).Error("requesty returned no choices")
		return "", fmt.Errorf("no response from requesty")
	}

	content := resp.Choices[0].Message.Content

	// Log metrics
	if resp.Usage != nil {
		cost := utils.EstimateCost(r.Config.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		log.WithFields(map[string]interface{}{
			"agent_name":        r.Name,
			"duration":          duration.String(),
			"model":             resp.Model,
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
			"cost":              fmt.Sprintf("$%.4f", cost),
		}).Info("requesty message sent successfully")
	} else {
		log.WithFields(map[string]interface{}{
			"agent_name": r.Name,
			"duration":   duration.String(),
			"model":      resp.Model,
		}).Info("requesty message sent successfully")
	}

	return strings.TrimSpace(content), nil
}

// StreamMessage sends a message to Requesty and streams the response.
func (r *RequestyAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    r.Name,
		"message_count": len(messages),
		"model":         r.Config.Model,
	}).Debug("starting requesty streaming message")

	// Build conversation history
	apiMessages := r.buildConversationHistory(messages)

	// Build request
	req := client.ChatCompletionRequest{
		Model:    r.Config.Model,
		Messages: apiMessages,
	}

	if r.Config.Temperature > 0 {
		req.Temperature = &r.Config.Temperature
	}

	if r.Config.MaxTokens > 0 {
		req.MaxTokens = &r.Config.MaxTokens
	}

	// Send streaming request
	startTime := time.Now()
	usage, err := r.client.CreateChatCompletionStream(ctx, req, writer)
	duration := time.Since(startTime)

	if err != nil {
		log.WithField("agent_name", r.Name).WithError(err).Error("requesty streaming failed")
		return fmt.Errorf("requesty streaming failed: %w", err)
	}

	// Log metrics
	if usage != nil {
		cost := utils.EstimateCost(r.Config.Model, usage.PromptTokens, usage.CompletionTokens)
		log.WithFields(map[string]interface{}{
			"agent_name":        r.Name,
			"duration":          duration.String(),
			"model":             r.Config.Model,
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
			"cost":              fmt.Sprintf("$%.4f", cost),
		}).Info("requesty streaming message completed")
	} else {
		log.WithFields(map[string]interface{}{
			"agent_name": r.Name,
			"duration":   duration.String(),
			"model":      r.Config.Model,
		}).Info("requesty streaming message completed")
	}

	return nil
}

// buildConversationHistory converts AgentPipe messages to OpenAI API format.
func (r *RequestyAgent) buildConversationHistory(messages []agent.Message) []client.ChatCompletionMessage {
	apiMessages := make([]client.ChatCompletionMessage, 0)

	// Add system prompt if configured
	if r.Config.Prompt != "" {
		apiMessages = append(apiMessages, client.ChatCompletionMessage{
			Role:    "system",
			Content: r.Config.Prompt,
		})
	}

	// Convert conversation messages
	for _, msg := range messages {
		// Skip this agent's own messages to avoid confusion
		if msg.AgentName == r.Name || msg.AgentID == r.ID {
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

func init() {
	agent.RegisterFactory("requesty", NewRequestyAgent)
}
