package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

type CodexAgent struct {
	agent.BaseAgent
	execPath string
}

func NewCodexAgent() agent.Agent {
	return &CodexAgent{}
}

func (c *CodexAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("codex agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("codex")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   c.ID,
			"agent_name": c.Name,
		}).WithError(err).Error("codex CLI not found in PATH")
		return fmt.Errorf("codex CLI not found: %w", err)
	}
	c.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   c.ID,
		"agent_name": c.Name,
		"exec_path":  path,
		"model":      c.Config.Model,
	}).Info("codex agent initialized successfully")

	return nil
}

func (c *CodexAgent) IsAvailable() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (c *CodexAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		log.WithField("agent_name", c.Name).Error("codex health check failed: not initialized")
		return fmt.Errorf("codex CLI not initialized")
	}

	log.WithField("agent_name", c.Name).Debug("starting codex health check")

	// Test with a simple version command
	cmd := exec.CommandContext(ctx, c.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try help if version doesn't work
		log.WithField("agent_name", c.Name).Debug("--version check failed, trying --help")
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.WithField("agent_name", c.Name).WithError(err).Error("codex health check failed: CLI not responding")
			return fmt.Errorf("codex health check failed: %w", err)
		}
	}

	if len(output) == 0 {
		log.WithField("agent_name", c.Name).Error("codex health check failed: empty response")
		return fmt.Errorf("codex returned empty response")
	}

	log.WithField("agent_name", c.Name).Info("codex health check passed")
	return nil
}

func (c *CodexAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
	}).Debug("sending message to codex CLI")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Build command args - use 'exec' subcommand for non-interactive mode
	args := []string{"exec"}

	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Skip approval prompts for non-interactive use
	args = append(args, "--dangerously-bypass-approvals-and-sandbox")

	// Use JSON output for cleaner parsing
	args = append(args, "--json")

	// Use "-" to read prompt from stdin
	args = append(args, "-")

	// Use stdin for the prompt
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "not found") {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"model":      c.Config.Model,
				"duration":   duration.String(),
			}).WithError(err).Error("codex model not found")
			return "", fmt.Errorf("codex model not found - check model name in config: %s", c.Config.Model)
		}
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "unauthorized") {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("codex authentication failed")
			return "", fmt.Errorf("codex authentication failed - check API keys")
		}
		if strings.Contains(outputStr, "terminal") || strings.Contains(outputStr, "tty") {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("codex terminal detection issue")
			return "", fmt.Errorf("codex requires non-interactive mode: %s", outputStr)
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("codex execution failed with exit code")
			return "", fmt.Errorf("codex execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		log.WithFields(map[string]interface{}{
			"agent_name": c.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("codex execution failed")
		return "", fmt.Errorf("codex execution failed: %w\nOutput: %s", err, outputStr)
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("codex message sent successfully")

	// Parse JSON output and extract agent message
	response := c.parseJSONOutput(string(output))
	return strings.TrimSpace(response), nil
}

func (c *CodexAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		// Exclude this agent's own messages
		if msg.AgentName == c.Name || msg.AgentID == c.ID {
			continue
		}
		relevant = append(relevant, msg)
	}
	return relevant
}

func (c *CodexAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", c.Name))

	if c.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(c.Config.Prompt)
		prompt.WriteString("\n\n")
	}

	// PART 2: CONVERSATION CONTEXT
	if len(messages) > 0 {
		var initialPrompt string
		var otherMessages []agent.Message

		// Find the orchestrator's initial prompt (AgentID="system")
		// vs agent announcements (system messages from specific agents)
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentName == "System") && initialPrompt == "" {
				// This is the orchestrator's initial prompt - show it prominently
				initialPrompt = msg.Content
			} else {
				// ALL other messages (agent announcements, other system messages, agent responses)
				otherMessages = append(otherMessages, msg)
			}
		}

		// PART 2: Show initial topic prominently as DIRECT TASK
		if initialPrompt != "" {
			prompt.WriteString("YOUR TASK - PLEASE RESPOND TO THIS:\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n")
			prompt.WriteString(initialPrompt)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n\n")
		}

		// PART 3: Show conversation history
		if len(otherMessages) > 0 {
			prompt.WriteString("CONVERSATION SO FAR:\n")
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n")
			for _, msg := range otherMessages {
				timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
				if msg.Role == "system" {
					// Agent announcements come through as system messages
					prompt.WriteString(fmt.Sprintf("[%s] SYSTEM: %s\n", timestamp, msg.Content))
				} else {
					prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.AgentName, msg.Content))
				}
			}
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n\n")
		}

		// Add closing instruction if we showed the initial task
		if initialPrompt != "" {
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", c.Name))
		}
	}

	return prompt.String()
}

func (c *CodexAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Build command args - use 'exec' subcommand for non-interactive mode
	args := []string{"exec"}

	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Skip approval prompts for non-interactive use
	args = append(args, "--dangerously-bypass-approvals-and-sandbox")

	// Use JSON output for streaming
	args = append(args, "--json")

	// Use "-" to read prompt from stdin
	args = append(args, "-")

	// Use stdin for the prompt
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start codex: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("codex execution failed: %w", err)
	}

	return nil
}

// parseJSONOutput parses Codex's JSON output and extracts the agent message text
func (c *CodexAgent) parseJSONOutput(output string) string {
	// Codex --json mode outputs multiple JSON lines
	// We need to find item.completed events with type="agent_message"
	lines := strings.Split(output, "\n")
	var messageText strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as JSON
		var event struct {
			Type string `json:"type"`
			Item struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
		}

		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// If it's not JSON, it might be plain text - include it
			if !strings.HasPrefix(line, "{") {
				if messageText.Len() > 0 {
					messageText.WriteString("\n")
				}
				messageText.WriteString(line)
			}
			continue
		}

		// Extract agent_message text
		if event.Type == "item.completed" && event.Item.Type == "agent_message" {
			if messageText.Len() > 0 {
				messageText.WriteString("\n\n")
			}
			messageText.WriteString(event.Item.Text)
		}
	}

	// If we didn't find any agent_message items, return the raw output
	if messageText.Len() == 0 {
		return output
	}

	return messageText.String()
}

func init() {
	agent.RegisterFactory("codex", NewCodexAgent)
}
