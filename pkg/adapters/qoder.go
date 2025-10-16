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

type QoderAgent struct {
	agent.BaseAgent
	execPath string
}

func NewQoderAgent() agent.Agent {
	return &QoderAgent{}
}

func (q *QoderAgent) Initialize(config agent.AgentConfig) error {
	if err := q.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("qoder agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("qodercli")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   q.ID,
			"agent_name": q.Name,
		}).WithError(err).Error("qodercli not found in PATH")
		return fmt.Errorf("qodercli not found: %w", err)
	}
	q.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":  q.ID,
		"agent_name": q.Name,
		"exec_path": path,
		"model":     q.Config.Model,
	}).Info("qoder agent initialized successfully")

	return nil
}

func (q *QoderAgent) IsAvailable() bool {
	_, err := exec.LookPath("qodercli")
	return err == nil
}

func (q *QoderAgent) HealthCheck(ctx context.Context) error {
	if q.execPath == "" {
		log.WithField("agent_name", q.Name).Error("qoder health check failed: not initialized")
		return fmt.Errorf("qodercli not initialized")
	}

	log.WithField("agent_name", q.Name).Debug("starting qoder health check")

	// Test with help flag
	cmd := exec.CommandContext(ctx, q.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.WithField("agent_name", q.Name).WithError(err).Error("qoder health check failed: CLI not responding")
		return fmt.Errorf("qodercli health check failed: %w", err)
	}

	// Check if output looks like qoder help
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    q.Name,
			"output_length": len(outputStr),
		}).Error("qoder health check failed: output too short")
		return fmt.Errorf("qodercli returned suspiciously short output")
	}

	// Verify it's actually Qoder by checking for key terms
	if !strings.Contains(strings.ToLower(outputStr), "qoder") && !strings.Contains(strings.ToLower(outputStr), "print") {
		log.WithField("agent_name", q.Name).Error("qoder health check failed: output doesn't appear to be from qodercli")
		return fmt.Errorf("CLI at path doesn't appear to be qodercli")
	}

	log.WithField("agent_name", q.Name).Info("qoder health check passed")
	return nil
}

func (q *QoderAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    q.Name,
		"message_count": len(messages),
	}).Debug("sending message to qodercli")

	// Filter out this agent's own messages
	relevantMessages := q.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := q.buildPrompt(relevantMessages, true)

	// Build command args for non-interactive mode
	args := []string{"--print"} // Non-interactive print mode

	// Add model flag if specified
	if q.Config.Model != "" {
		args = append(args, "--model", q.Config.Model)
	}

	// Use --yolo to skip permission prompts for non-interactive use
	args = append(args, "--yolo")

	// Use text output format for easier parsing
	args = append(args, "--output-format", "text")

	// Use stdin for the prompt
	cmd := exec.CommandContext(ctx, q.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "not logged in") || strings.Contains(outputStr, "authentication") {
			log.WithFields(map[string]interface{}{
				"agent_name": q.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("qoder authentication failed")
			return "", fmt.Errorf("qodercli not authenticated - please run 'qodercli' and use '/login' command")
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": q.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("qoder execution failed with exit code")
			return "", fmt.Errorf("qodercli execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		log.WithFields(map[string]interface{}{
			"agent_name": q.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("qoder execution failed")
		return "", fmt.Errorf("qodercli execution failed: %w\nOutput: %s", err, string(output))
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    q.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("qoder message sent successfully")

	return strings.TrimSpace(string(output)), nil
}

func (q *QoderAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    q.Name,
		"message_count": len(messages),
	}).Debug("starting qoder streaming message")

	// Filter out this agent's own messages
	relevantMessages := q.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := q.buildPrompt(relevantMessages, true)

	// Build command args for streaming mode
	args := []string{"--print"}

	// Add model flag if specified
	if q.Config.Model != "" {
		args = append(args, "--model", q.Config.Model)
	}

	// Use --yolo to skip permission prompts
	args = append(args, "--yolo")

	// Use stream-json format for real-time output
	args = append(args, "--output-format", "stream-json")

	// Use stdin for the prompt
	cmd := exec.CommandContext(ctx, q.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", q.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", q.Name).WithError(err).Error("failed to start qoder process")
		return fmt.Errorf("failed to start qodercli: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", q.Name).WithError(err).Error("error reading qoder streaming output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		log.WithField("agent_name", q.Name).WithError(err).Error("qoder streaming execution failed")
		return fmt.Errorf("qodercli execution failed: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"agent_name": q.Name,
		"lines":      lineCount,
	}).Info("qoder streaming message completed")

	return nil
}

func (q *QoderAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
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

func (q *QoderAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
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

		// Find orchestrator's initial prompt (HOST) vs agent announcements (SYSTEM)
		// HOST = orchestrator's initial task/prompt (AgentID="host", AgentName="HOST")
		// SYSTEM = agent join announcements and other system messages
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
				initialPrompt = msg.Content
			} else {
				otherMessages = append(otherMessages, msg)
			}
		}

		// Show initial task prominently
		if initialPrompt != "" {
			prompt.WriteString("YOUR TASK - PLEASE RESPOND TO THIS:\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n")
			prompt.WriteString(initialPrompt)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n\n")
		}

		// Show conversation history
		if len(otherMessages) > 0 {
			prompt.WriteString("CONVERSATION SO FAR:\n")
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n")
			for _, msg := range otherMessages {
				timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", q.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("qoder", NewQoderAgent)
}
