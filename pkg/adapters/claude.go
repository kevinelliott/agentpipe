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

type ClaudeAgent struct {
	agent.BaseAgent
	execPath string
}

func NewClaudeAgent() agent.Agent {
	return &ClaudeAgent{}
}

func (c *ClaudeAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("claude agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("claude")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   c.ID,
			"agent_name": c.Name,
		}).WithError(err).Error("claude CLI not found in PATH")
		return fmt.Errorf("claude CLI not found: %w", err)
	}
	c.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   c.ID,
		"agent_name": c.Name,
		"exec_path":  path,
		"model":      c.Config.Model,
	}).Info("claude agent initialized successfully")

	return nil
}

func (c *ClaudeAgent) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		log.WithField("agent_name", c.Name).Error("claude health check failed: not initialized")
		return fmt.Errorf("claude CLI not initialized")
	}

	log.WithField("agent_name", c.Name).Debug("starting claude health check")

	// For Claude, we'll just check if the binary exists and is executable
	// The actual prompt test might hang if it's waiting for API keys or other config
	cmd := exec.CommandContext(ctx, c.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try with help flag if version doesn't work
		log.WithField("agent_name", c.Name).Debug("--version check failed, trying --help")
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		output, err = cmd.CombinedOutput()

		if err != nil {
			// If both fail, the CLI is not properly installed
			log.WithField("agent_name", c.Name).WithError(err).Error("claude health check failed: CLI not responding")
			return fmt.Errorf("claude CLI not responding to --version or --help: %w", err)
		}
	}

	// Check if output contains something that indicates it's Claude
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    c.Name,
			"output_length": len(outputStr),
		}).Error("claude health check failed: output too short")
		return fmt.Errorf("claude CLI returned suspiciously short output")
	}

	log.WithField("agent_name", c.Name).Info("claude health check passed")
	return nil
}

func (c *ClaudeAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
	}).Debug("sending message to claude CLI")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Claude CLI takes prompt via stdin
	cmd := exec.CommandContext(ctx, c.execPath)
	cmd.Stdin = strings.NewReader(prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("claude execution failed with exit code")
			return "", fmt.Errorf("claude execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		log.WithFields(map[string]interface{}{
			"agent_name": c.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w\nOutput: %s", err, string(output))
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("claude message sent successfully")

	return string(output), nil
}

func (c *ClaudeAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
	}).Debug("starting claude streaming message")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Claude CLI takes prompt via stdin
	cmd := exec.CommandContext(ctx, c.execPath)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to start claude process")
		return fmt.Errorf("failed to start claude: %w", err)
	}

	startTime := time.Now()
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("error reading streaming output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("claude streaming execution failed")
		return fmt.Errorf("claude execution failed: %w", err)
	}

	duration := time.Since(startTime)
	log.WithFields(map[string]interface{}{
		"agent_name": c.Name,
		"duration":   duration.String(),
		"lines":      lineCount,
	}).Info("claude streaming message completed")

	return nil
}

// filterRelevantMessages filters out this agent's own messages
// We exclude this agent's own messages to avoid showing Claude what it already said
func (c *ClaudeAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip this agent's own messages
		if msg.AgentName == c.Name || msg.AgentID == c.ID {
			continue
		}
		// Include messages from other agents and system messages
		relevant = append(relevant, msg)
	}

	return relevant
}

func (c *ClaudeAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE (always first)
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", c.Name))

	if c.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(c.Config.Prompt)
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.", c.Name))
		} else {
			prompt.WriteString(fmt.Sprintf("Now, as %s, respond to the conversation.", c.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("claude", NewClaudeAgent)
}
