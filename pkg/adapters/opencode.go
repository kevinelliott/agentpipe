package adapters

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/internal/registry"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

type OpenCodeAgent struct {
	agent.BaseAgent
	execPath string
}

func NewOpenCodeAgent() agent.Agent {
	return &OpenCodeAgent{}
}

func (o *OpenCodeAgent) Initialize(config agent.AgentConfig) error {
	if err := o.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("opencode agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("opencode")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   o.ID,
			"agent_name": o.Name,
		}).WithError(err).Error("opencode CLI not found in PATH")
		return fmt.Errorf("opencode CLI not found: %w", err)
	}
	o.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   o.ID,
		"agent_name": o.Name,
		"exec_path":  path,
		"model":      o.Config.Model,
	}).Info("opencode agent initialized successfully")

	return nil
}

func (o *OpenCodeAgent) IsAvailable() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

func (o *OpenCodeAgent) GetCLIVersion() string {
	return registry.GetInstalledVersion("opencode")
}

func (o *OpenCodeAgent) HealthCheck(ctx context.Context) error {
	if o.execPath == "" {
		log.WithField("agent_name", o.Name).Error("opencode health check failed: not initialized")
		return fmt.Errorf("opencode CLI not initialized")
	}

	log.WithField("agent_name", o.Name).Debug("starting opencode health check")

	// Test with a simple version command
	cmd := exec.CommandContext(ctx, o.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try help if version doesn't work
		log.WithField("agent_name", o.Name).Debug("--version check failed, trying --help")
		cmd = exec.CommandContext(ctx, o.execPath, "--help")
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.WithField("agent_name", o.Name).WithError(err).Error("opencode health check failed: CLI not responding")
			return fmt.Errorf("opencode health check failed: %w", err)
		}
	}

	if len(output) == 0 {
		log.WithField("agent_name", o.Name).Error("opencode health check failed: empty response")
		return fmt.Errorf("opencode returned empty response")
	}

	log.WithField("agent_name", o.Name).Info("opencode health check passed")
	return nil
}

func (o *OpenCodeAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    o.Name,
		"message_count": len(messages),
	}).Debug("sending message to opencode CLI")

	// Filter out this agent's own messages
	relevantMessages := o.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := o.buildPrompt(relevantMessages, true)

	// Log prompt preview for debugging
	promptPreview := prompt
	if len(promptPreview) > 300 {
		promptPreview = promptPreview[:300] + "..."
	}
	log.WithFields(map[string]interface{}{
		"agent_name":     o.Name,
		"prompt_preview": promptPreview,
	}).Debug("opencode prompt preview")

	// Build command args - use 'run' subcommand for non-interactive mode
	args := []string{"run"}

	// Add quiet flag to disable spinner
	args = append(args, "--quiet")

	// Add the prompt as the final argument
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, o.execPath, args...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "not found") {
			log.WithFields(map[string]interface{}{
				"agent_name": o.Name,
				"model":      o.Config.Model,
				"duration":   duration.String(),
			}).WithError(err).Error("opencode model not found")
			return "", fmt.Errorf("opencode model not found - check model name in config: %s", o.Config.Model)
		}
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "unauthorized") || strings.Contains(outputStr, "authentication") {
			log.WithFields(map[string]interface{}{
				"agent_name": o.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("opencode authentication failed")
			return "", fmt.Errorf("opencode authentication failed - run 'opencode auth login' to authenticate")
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": o.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("opencode execution failed with exit code")
			return "", fmt.Errorf("opencode execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		log.WithFields(map[string]interface{}{
			"agent_name": o.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("opencode execution failed")
		return "", fmt.Errorf("opencode execution failed: %w\nOutput: %s", err, outputStr)
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    o.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("opencode message sent successfully")

	return strings.TrimSpace(string(output)), nil
}

func (o *OpenCodeAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		// Exclude this agent's own messages
		if msg.AgentName == o.Name || msg.AgentID == o.ID {
			continue
		}
		relevant = append(relevant, msg)
	}
	return relevant
}

func (o *OpenCodeAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", o.Name))

	if o.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(o.Config.Prompt)
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", o.Name))
		}
	}

	return prompt.String()
}

func (o *OpenCodeAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	// Filter out this agent's own messages
	relevantMessages := o.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := o.buildPrompt(relevantMessages, true)

	// Build command args - use 'run' subcommand for non-interactive mode
	args := []string{"run"}

	// Add quiet flag to disable spinner
	args = append(args, "--quiet")

	// Add the prompt
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, o.execPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode execution failed: %w", err)
	}

	return nil
}

func init() {
	agent.RegisterFactory("opencode", NewOpenCodeAgent)
}
