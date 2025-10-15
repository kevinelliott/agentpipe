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

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

	// Claude CLI takes prompt via stdin, no command line args for prompt
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

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

	// Claude CLI takes prompt via stdin, no command line args for prompt
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

func (c *ClaudeAgent) formatConversation(messages []agent.Message) string {
	parts := make([]string, 0, len(messages))

	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}

	return strings.Join(parts, "\n")
}

func (c *ClaudeAgent) buildPrompt(conversation string) string {
	return BuildAgentPrompt(c.Name, c.Config.Prompt, conversation)
}

func init() {
	agent.RegisterFactory("claude", NewClaudeAgent)
}
