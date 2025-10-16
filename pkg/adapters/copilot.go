package adapters

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

type CopilotAgent struct {
	agent.BaseAgent
	execPath string
}

func NewCopilotAgent() agent.Agent {
	return &CopilotAgent{}
}

func (c *CopilotAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("copilot agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("copilot")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   c.ID,
			"agent_name": c.Name,
		}).WithError(err).Error("copilot CLI not found in PATH")
		return fmt.Errorf("copilot CLI not found: %w", err)
	}
	c.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   c.ID,
		"agent_name": c.Name,
		"exec_path":  path,
		"model":      c.Config.Model,
	}).Info("copilot agent initialized successfully")

	return nil
}

func (c *CopilotAgent) IsAvailable() bool {
	_, err := exec.LookPath("copilot")
	return err == nil
}

func (c *CopilotAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		return fmt.Errorf("copilot CLI not initialized")
	}

	// Check if copilot is available and can show help
	cmd := exec.CommandContext(ctx, c.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try version command as fallback
		cmd = exec.CommandContext(ctx, c.execPath, "--version")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("copilot CLI not responding: %w", err)
		}
	}

	// Check if we got meaningful output
	if len(output) < 10 {
		return fmt.Errorf("copilot CLI returned suspiciously short output")
	}

	// Check if authentication is required
	outputStr := string(output)
	if strings.Contains(outputStr, "not authenticated") || strings.Contains(outputStr, "not logged in") {
		return fmt.Errorf("copilot not authenticated - please run 'copilot' and use '/login' command")
	}

	return nil
}

func (c *CopilotAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
	}).Debug("sending message to copilot CLI")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Use non-interactive mode with -p/--prompt flag
	args := []string{"-p", prompt}

	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Use --allow-all-tools for non-interactive execution
	// This prevents copilot from asking for confirmation
	args = append(args, "--allow-all-tools")

	cmd := exec.CommandContext(ctx, c.execPath, args...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "not authenticated") || strings.Contains(outputStr, "not logged in") {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("copilot authentication failed")
			return "", fmt.Errorf("copilot authentication failed - please run 'copilot' and use '/login' command")
		}
		if strings.Contains(outputStr, "subscription") {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("copilot subscription required")
			return "", fmt.Errorf("copilot subscription required - check your GitHub Copilot access")
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("copilot execution failed with exit code")
			return "", fmt.Errorf("copilot execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		log.WithFields(map[string]interface{}{
			"agent_name": c.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("copilot execution failed")
		return "", fmt.Errorf("copilot execution failed: %w\nOutput: %s", err, outputStr)
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("copilot message sent successfully")

	return strings.TrimSpace(string(output)), nil
}

func (c *CopilotAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Use non-interactive mode with -p/--prompt flag
	args := []string{"-p", prompt}

	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Use --allow-all-tools for non-interactive execution
	args = append(args, "--allow-all-tools")

	cmd := exec.CommandContext(ctx, c.execPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start copilot: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("copilot execution failed: %w", err)
	}

	return nil
}

func (c *CopilotAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
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

func (c *CopilotAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
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

		// Find the orchestrator's initial prompt (AgentID="system" or "host")
		// vs agent announcements (system messages from specific agents)
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
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

func init() {
	agent.RegisterFactory("copilot", NewCopilotAgent)
}
