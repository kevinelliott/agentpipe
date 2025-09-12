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
		return err
	}

	path, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found: %w", err)
	}
	c.execPath = path

	return nil
}

func (c *ClaudeAgent) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		return fmt.Errorf("claude CLI not initialized")
	}

	// For Claude, we'll just check if the binary exists and is executable
	// The actual prompt test might hang if it's waiting for API keys or other config
	cmd := exec.CommandContext(ctx, c.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try with help flag if version doesn't work
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		output, err = cmd.CombinedOutput()

		if err != nil {
			// If both fail, the CLI is not properly installed
			return fmt.Errorf("claude CLI not responding to --version or --help: %w", err)
		}
	}

	// Check if output contains something that indicates it's Claude
	outputStr := string(output)
	if len(outputStr) < 10 {
		return fmt.Errorf("claude CLI returned suspiciously short output")
	}

	return nil
}

func (c *ClaudeAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

	// Claude CLI takes prompt via stdin, no command line args for prompt
	cmd := exec.CommandContext(ctx, c.execPath)
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		return "", fmt.Errorf("claude execution failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

func (c *ClaudeAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

	// Claude CLI takes prompt via stdin, no command line args for prompt
	cmd := exec.CommandContext(ctx, c.execPath)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude execution failed: %w", err)
	}

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
	var prompt strings.Builder

	prompt.WriteString("You are participating in a multi-agent conversation. ")
	prompt.WriteString(fmt.Sprintf("Your name is '%s'. ", c.Name))

	if c.Config.Prompt != "" {
		prompt.WriteString(c.Config.Prompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Here is the conversation so far:\n\n")
	prompt.WriteString(conversation)
	prompt.WriteString("\n\nContinue the conversation naturally as ")
	prompt.WriteString(c.Name)
	prompt.WriteString(". Build on what was just said without repeating previous points. Don't announce that you're joining - just respond directly:")

	return prompt.String()
}

func init() {
	agent.RegisterFactory("claude", NewClaudeAgent)
}
