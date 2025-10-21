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

type FactoryAgent struct {
	agent.BaseAgent
	execPath string
}

func NewFactoryAgent() agent.Agent {
	return &FactoryAgent{}
}

func (f *FactoryAgent) Initialize(config agent.AgentConfig) error {
	if err := f.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("factory agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("droid")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   f.ID,
			"agent_name": f.Name,
		}).WithError(err).Error("droid CLI not found in PATH")
		return fmt.Errorf("droid CLI not found: %w", err)
	}
	f.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   f.ID,
		"agent_name": f.Name,
		"exec_path":  path,
		"model":      f.Config.Model,
	}).Info("factory agent initialized successfully")

	return nil
}

func (f *FactoryAgent) IsAvailable() bool {
	_, err := exec.LookPath("droid")
	return err == nil
}

func (f *FactoryAgent) GetCLIVersion() string {
	return registry.GetInstalledVersion("droid")
}

func (f *FactoryAgent) HealthCheck(ctx context.Context) error {
	if f.execPath == "" {
		log.WithField("agent_name", f.Name).Error("factory health check failed: not initialized")
		return fmt.Errorf("droid CLI not initialized")
	}

	log.WithField("agent_name", f.Name).Debug("starting factory health check")

	// Check if the binary exists and is executable using --help
	cmd := exec.CommandContext(ctx, f.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try with --version if --help doesn't work
		log.WithField("agent_name", f.Name).Debug("--help check failed, trying --version")
		cmd = exec.CommandContext(ctx, f.execPath, "--version")
		output, err = cmd.CombinedOutput()

		if err != nil {
			// If both fail, the CLI is not properly installed
			log.WithField("agent_name", f.Name).WithError(err).Error("factory health check failed: CLI not responding")
			return fmt.Errorf("droid CLI not responding to --help or --version: %w", err)
		}
	}

	// Check if output contains something that indicates it's Factory/Droid
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    f.Name,
			"output_length": len(outputStr),
		}).Error("factory health check failed: output too short")
		return fmt.Errorf("droid CLI returned suspiciously short output")
	}

	log.WithField("agent_name", f.Name).Info("factory health check passed")
	return nil
}

func (f *FactoryAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    f.Name,
		"message_count": len(messages),
	}).Debug("sending message to droid CLI")

	// Filter out this agent's own messages
	relevantMessages := f.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := f.buildPrompt(relevantMessages, true)

	// Build command args for droid exec (non-interactive mode)
	args := []string{"exec"}

	// Add model flag if specified
	if f.Config.Model != "" {
		args = append(args, "--model", f.Config.Model)
	}

	// Set autonomy level to high for multi-agent conversations
	// This allows the agent to perform operations without permission prompts
	args = append(args, "--auto", "high")

	// Add the prompt as the last argument
	args = append(args, prompt)

	// Execute droid exec command
	cmd := exec.CommandContext(ctx, f.execPath, args...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": f.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("factory execution failed with exit code")
			return "", fmt.Errorf("droid execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		log.WithFields(map[string]interface{}{
			"agent_name": f.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("factory execution failed")
		return "", fmt.Errorf("droid execution failed: %w\nOutput: %s", err, string(output))
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    f.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("factory message sent successfully")

	return strings.TrimSpace(string(output)), nil
}

func (f *FactoryAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    f.Name,
		"message_count": len(messages),
	}).Debug("starting factory streaming message")

	// Filter out this agent's own messages
	relevantMessages := f.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := f.buildPrompt(relevantMessages, true)

	// Build command args for droid exec
	args := []string{"exec"}

	// Add model flag if specified
	if f.Config.Model != "" {
		args = append(args, "--model", f.Config.Model)
	}

	// Set autonomy level to high
	args = append(args, "--auto", "high")

	// Add the prompt
	args = append(args, prompt)

	// Execute droid exec command
	cmd := exec.CommandContext(ctx, f.execPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", f.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", f.Name).WithError(err).Error("failed to start factory process")
		return fmt.Errorf("failed to start droid: %w", err)
	}

	startTime := time.Now()
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", f.Name).WithError(err).Error("error reading streaming output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		log.WithField("agent_name", f.Name).WithError(err).Error("factory streaming execution failed")
		return fmt.Errorf("droid execution failed: %w", err)
	}

	duration := time.Since(startTime)
	log.WithFields(map[string]interface{}{
		"agent_name": f.Name,
		"duration":   duration.String(),
		"lines":      lineCount,
	}).Info("factory streaming message completed")

	return nil
}

// filterRelevantMessages filters out this agent's own messages
// We exclude this agent's own messages to avoid showing Factory what it already said
func (f *FactoryAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip this agent's own messages
		if msg.AgentName == f.Name || msg.AgentID == f.ID {
			continue
		}
		// Include messages from other agents and system messages
		relevant = append(relevant, msg)
	}

	return relevant
}

func (f *FactoryAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE (always first)
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", f.Name))

	if f.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(f.Config.Prompt)
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.", f.Name))
		} else {
			prompt.WriteString(fmt.Sprintf("Now, as %s, respond to the conversation.", f.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("factory", NewFactoryAgent)
}
