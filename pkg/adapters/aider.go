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

type AiderAgent struct {
	agent.BaseAgent
	execPath string
}

func NewAiderAgent() agent.Agent {
	return &AiderAgent{}
}

func (a *AiderAgent) Initialize(config agent.AgentConfig) error {
	if err := a.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("aider agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("aider")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   a.ID,
			"agent_name": a.Name,
		}).WithError(err).Error("aider CLI not found in PATH")
		return fmt.Errorf("aider CLI not found: %w", err)
	}
	a.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   a.ID,
		"agent_name": a.Name,
		"exec_path":  path,
		"model":      a.Config.Model,
	}).Info("aider agent initialized successfully")

	return nil
}

func (a *AiderAgent) IsAvailable() bool {
	_, err := exec.LookPath("aider")
	return err == nil
}

func (a *AiderAgent) GetCLIVersion() string {
	return registry.GetInstalledVersion("aider")
}

func (a *AiderAgent) HealthCheck(ctx context.Context) error {
	if a.execPath == "" {
		log.WithField("agent_name", a.Name).Error("aider health check failed: not initialized")
		return fmt.Errorf("aider CLI not initialized")
	}

	log.WithField("agent_name", a.Name).Debug("starting aider health check")

	// Check if the binary exists and is executable using --version
	cmd := exec.CommandContext(ctx, a.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try with --help if --version doesn't work
		log.WithField("agent_name", a.Name).Debug("--version check failed, trying --help")
		cmd = exec.CommandContext(ctx, a.execPath, "--help")
		output, err = cmd.CombinedOutput()

		if err != nil {
			// If both fail, the CLI is not properly installed
			log.WithField("agent_name", a.Name).WithError(err).Error("aider health check failed: CLI not responding")
			return fmt.Errorf("aider CLI not responding to --version or --help: %w", err)
		}
	}

	// Check if output contains something that indicates it's Aider
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    a.Name,
			"output_length": len(outputStr),
		}).Error("aider health check failed: output too short")
		return fmt.Errorf("aider CLI returned suspiciously short output")
	}

	log.WithField("agent_name", a.Name).Info("aider health check passed")
	return nil
}

func (a *AiderAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"message_count": len(messages),
	}).Debug("sending message to aider CLI")

	// Filter out this agent's own messages
	relevantMessages := a.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := a.buildPrompt(relevantMessages, true)

	// Build command args - Aider uses --message for non-interactive mode
	args := []string{
		"--yes",        // Auto-confirm changes
		"--no-git",     // Don't use git (we're in a conversation, not editing files)
		"--no-stream",  // Don't stream output for non-interactive mode
		"--message", prompt,
	}

	// Add model flag if specified
	if a.Config.Model != "" {
		args = append([]string{"--model", a.Config.Model}, args...)
	}

	// Execute aider command
	cmd := exec.CommandContext(ctx, a.execPath, args...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": a.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("aider execution failed with exit code")
			return "", fmt.Errorf("aider execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		log.WithFields(map[string]interface{}{
			"agent_name": a.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("aider execution failed")
		return "", fmt.Errorf("aider execution failed: %w\nOutput: %s", err, string(output))
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("aider message sent successfully")

	return strings.TrimSpace(string(output)), nil
}

func (a *AiderAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"message_count": len(messages),
	}).Debug("starting aider streaming message")

	// Filter out this agent's own messages
	relevantMessages := a.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := a.buildPrompt(relevantMessages, true)

	// Build command args for streaming mode
	args := []string{
		"--yes",       // Auto-confirm changes
		"--no-git",    // Don't use git
		"--message", prompt,
	}

	// Add model flag if specified
	if a.Config.Model != "" {
		args = append([]string{"--model", a.Config.Model}, args...)
	}

	// Execute aider command
	cmd := exec.CommandContext(ctx, a.execPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("failed to start aider process")
		return fmt.Errorf("failed to start aider: %w", err)
	}

	startTime := time.Now()
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("error reading streaming output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("aider streaming execution failed")
		return fmt.Errorf("aider execution failed: %w", err)
	}

	duration := time.Since(startTime)
	log.WithFields(map[string]interface{}{
		"agent_name": a.Name,
		"duration":   duration.String(),
		"lines":      lineCount,
	}).Info("aider streaming message completed")

	return nil
}

// filterRelevantMessages filters out this agent's own messages
func (a *AiderAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip this agent's own messages
		if msg.AgentName == a.Name || msg.AgentID == a.ID {
			continue
		}
		// Include messages from other agents and system messages
		relevant = append(relevant, msg)
	}

	return relevant
}

func (a *AiderAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE (always first)
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", a.Name))

	if a.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(a.Config.Prompt)
		prompt.WriteString("\n")
	}
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n\n")

	// PART 2: CONVERSATION CONTEXT (after role is established)
	if len(messages) > 0 {
		// Deliver ALL existing messages including initial prompt and all conversation
		var initialPrompt string
		var otherMessages []agent.Message

		// IMPORTANT: Find the orchestrator's initial prompt (AgentID/AgentName = "host" or "system")
		// Agent announcements are also system messages, but they come from specific agents
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
				// This is the orchestrator's initial prompt - show it prominently
				initialPrompt = msg.Content
			} else {
				// ALL other messages (agent announcements, other system messages, agent responses)
				otherMessages = append(otherMessages, msg)
			}
		}

		// Show the initial prompt as a DIRECT INSTRUCTION
		if initialPrompt != "" {
			prompt.WriteString("YOUR TASK - PLEASE RESPOND TO THIS:\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n")
			prompt.WriteString(initialPrompt)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n\n")
		}

		// Then show ALL remaining conversation (system messages + agent messages)
		if len(otherMessages) > 0 {
			prompt.WriteString("CONVERSATION SO FAR:\n")
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n")
			for _, msg := range otherMessages {
				timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
				// Include role indicator for system messages to make them clear
				if msg.Role == "system" {
					prompt.WriteString(fmt.Sprintf("[%s] SYSTEM: %s\n", timestamp, msg.Content))
				} else {
					prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.AgentName, msg.Content))
				}
			}
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n\n")
		}

		if initialPrompt != "" {
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.", a.Name))
		} else {
			prompt.WriteString(fmt.Sprintf("Now, as %s, respond to the conversation.", a.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("aider", NewAiderAgent)
}
