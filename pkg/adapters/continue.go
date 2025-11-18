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

type ContinueAgent struct {
	agent.BaseAgent
	execPath string
}

func NewContinueAgent() agent.Agent {
	return &ContinueAgent{}
}

func (c *ContinueAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("continue agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("cn")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   c.ID,
			"agent_name": c.Name,
		}).WithError(err).Error("continue CLI not found in PATH")
		return fmt.Errorf("continue CLI not found: %w", err)
	}
	c.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   c.ID,
		"agent_name": c.Name,
		"exec_path":  path,
		"model":      c.Config.Model,
	}).Info("continue agent initialized successfully")

	return nil
}

func (c *ContinueAgent) IsAvailable() bool {
	_, err := exec.LookPath("cn")
	return err == nil
}

func (c *ContinueAgent) GetCLIVersion() string {
	return registry.GetInstalledVersion("cn")
}

func (c *ContinueAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		log.WithField("agent_name", c.Name).Error("continue health check failed: not initialized")
		return fmt.Errorf("continue CLI not initialized")
	}

	log.WithField("agent_name", c.Name).Debug("starting continue health check")

	// Check if the binary exists and is executable
	cmd := exec.CommandContext(ctx, c.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try with help flag if version doesn't work
		log.WithField("agent_name", c.Name).Debug("--version check failed, trying --help")
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		output, err = cmd.CombinedOutput()

		if err != nil {
			// If both fail, the CLI is not properly installed
			log.WithField("agent_name", c.Name).WithError(err).Error("continue health check failed: CLI not responding")
			return fmt.Errorf("continue CLI not responding to --version or --help: %w", err)
		}
	}

	// Check if output contains something that indicates it's Continue CLI
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    c.Name,
			"output_length": len(outputStr),
		}).Error("continue health check failed: output too short")
		return fmt.Errorf("continue CLI returned suspiciously short output")
	}

	log.WithField("agent_name", c.Name).Info("continue health check passed")
	return nil
}

func (c *ContinueAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
	}).Debug("sending message to continue CLI")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Build command args
	args := []string{"-p", prompt}

	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Add silent flag to strip <think></think> tags
	args = append(args, "--silent")

	// Continue CLI uses -p flag with prompt as argument
	cmd := exec.CommandContext(ctx, c.execPath, args...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("continue execution failed with exit code")
			return "", fmt.Errorf("continue execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		log.WithFields(map[string]interface{}{
			"agent_name": c.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("continue execution failed")
		return "", fmt.Errorf("continue execution failed: %w\nOutput: %s", err, string(output))
	}

	response := string(output)

	// Filter out any status messages or metadata (similar to Gemini adapter)
	response = c.filterStatusMessages(response)

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"duration":      duration.String(),
		"response_size": len(response),
	}).Info("continue message sent successfully")

	return response, nil
}

func (c *ContinueAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
	}).Debug("starting continue streaming message")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Build command args
	args := []string{"-p", prompt}

	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Add silent flag to strip <think></think> tags
	args = append(args, "--silent")

	// Continue CLI uses -p flag with prompt as argument
	cmd := exec.CommandContext(ctx, c.execPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to start continue process")
		return fmt.Errorf("failed to start continue: %w", err)
	}

	// Stream output line by line
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and status messages
		if line == "" || c.isStatusMessage(line) {
			continue
		}

		// Write line to output
		if _, err := fmt.Fprintln(writer, line); err != nil {
			log.WithField("agent_name", c.Name).WithError(err).Error("failed to write to output")
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("error reading continue output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": c.Name,
				"exit_code":  exitErr.ExitCode(),
			}).WithError(err).Error("continue streaming failed with exit code")
			return fmt.Errorf("continue execution failed (exit code %d)", exitErr.ExitCode())
		}
		log.WithField("agent_name", c.Name).WithError(err).Error("continue streaming failed")
		return fmt.Errorf("continue execution failed: %w", err)
	}

	log.WithField("agent_name", c.Name).Info("continue streaming completed successfully")
	return nil
}

// filterStatusMessages removes Continue CLI status messages from output
func (c *ContinueAgent) filterStatusMessages(output string) string {
	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))

	for _, line := range lines {
		// Skip empty lines and status messages
		if line == "" || c.isStatusMessage(line) {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

// isStatusMessage checks if a line is a Continue CLI status message
func (c *ContinueAgent) isStatusMessage(line string) bool {
	// Filter out common Continue CLI status messages
	// These patterns may need adjustment based on actual Continue CLI output
	statusPrefixes := []string{
		"Loading",
		"Initializing",
		"Connecting",
		"[INFO]",
		"[DEBUG]",
		"Session ID:",
	}

	for _, prefix := range statusPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// filterRelevantMessages filters out the agent's own messages
func (c *ContinueAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
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

// buildPrompt constructs a structured prompt with three parts: identity, context, and instruction
func (c *ContinueAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
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
	agent.RegisterFactory("continue", NewContinueAgent)
}
