package adapters

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

type QwenAgent struct {
	agent.BaseAgent
	execPath string
}

func NewQwenAgent() agent.Agent {
	return &QwenAgent{}
}

func (q *QwenAgent) Initialize(config agent.AgentConfig) error {
	if err := q.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("qwen agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("qwen")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   q.ID,
			"agent_name": q.Name,
		}).WithError(err).Error("qwen CLI not found in PATH")
		return fmt.Errorf("qwen CLI not found: %w", err)
	}
	q.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   q.ID,
		"agent_name": q.Name,
		"exec_path":  path,
		"model":      q.Config.Model,
	}).Info("qwen agent initialized successfully")

	return nil
}

func (q *QwenAgent) IsAvailable() bool {
	_, err := exec.LookPath("qwen")
	return err == nil
}

func (q *QwenAgent) HealthCheck(ctx context.Context) error {
	if q.execPath == "" {
		return fmt.Errorf("qwen CLI not initialized")
	}

	// Test with version or help command instead of a prompt
	cmd := exec.CommandContext(ctx, q.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try help if version doesn't work
		cmd = exec.CommandContext(ctx, q.execPath, "--help")
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Some CLIs might not support flags, just check if we can execute it
			testCmd := exec.Command(q.execPath)
			if err := testCmd.Start(); err != nil {
				return fmt.Errorf("qwen CLI cannot be executed: %w", err)
			}
			// Kill the process if it's still running
			if testCmd.Process != nil {
				_ = testCmd.Process.Kill()
				_ = testCmd.Wait() // Clean up the process
			}
			// If we can start it, consider it healthy
			return nil
		}
	}

	if len(output) == 0 {
		// Empty output is OK for version/help commands
		return nil
	}

	return nil
}

func (q *QwenAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    q.Name,
		"message_count": len(messages),
	}).Debug("sending message to qwen CLI")

	// Filter out this agent's own messages
	relevantMessages := q.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := q.buildPrompt(relevantMessages, true)

	// Qwen uses -p/--prompt for non-interactive mode
	args := []string{}
	if q.Config.Model != "" {
		args = append(args, "--model", q.Config.Model)
	}
	// Note: qwen CLI doesn't seem to support temperature/max-tokens flags based on --help output
	args = append(args, "--prompt", prompt)

	cmd := exec.CommandContext(ctx, q.execPath, args...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_name": q.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("qwen execution failed")
		return "", fmt.Errorf("qwen execution failed: %w, output: %s", err, string(output))
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    q.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("qwen message sent successfully")

	return strings.TrimSpace(string(output)), nil
}

func (q *QwenAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	// Filter out this agent's own messages
	relevantMessages := q.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := q.buildPrompt(relevantMessages, true)

	// Qwen uses -p/--prompt for non-interactive mode
	// Note: Streaming might not be directly supported, fallback to regular execution
	args := []string{}
	if q.Config.Model != "" {
		args = append(args, "--model", q.Config.Model)
	}
	args = append(args, "--prompt", prompt)

	cmd := exec.CommandContext(ctx, q.execPath, args...)

	// For now, just execute and write the output since qwen may not support streaming
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qwen execution failed: %w", err)
	}

	fmt.Fprintln(writer, strings.TrimSpace(string(output)))
	return nil
}

func (q *QwenAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		// Exclude this agent's own messages
		if msg.AgentName == q.Name || msg.AgentID == q.ID {
			continue
		}
		relevant = append(relevant, msg)
	}
	return relevant
}

func (q *QwenAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", q.Name))

	if q.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(q.Config.Prompt)
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", q.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("qwen", NewQwenAgent)
}
