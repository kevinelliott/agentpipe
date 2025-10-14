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

type CopilotAgent struct {
	agent.BaseAgent
	execPath string
}

func NewCopilotAgent() agent.Agent {
	return &CopilotAgent{}
}

func (c *CopilotAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		return err
	}

	path, err := exec.LookPath("copilot")
	if err != nil {
		return fmt.Errorf("copilot CLI not found: %w", err)
	}
	c.execPath = path

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

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

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

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "not authenticated") || strings.Contains(outputStr, "not logged in") {
			return "", fmt.Errorf("copilot authentication failed - please run 'copilot' and use '/login' command")
		}
		if strings.Contains(outputStr, "subscription") {
			return "", fmt.Errorf("copilot subscription required - check your GitHub Copilot access")
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("copilot execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		return "", fmt.Errorf("copilot execution failed: %w\nOutput: %s", err, outputStr)
	}

	return strings.TrimSpace(string(output)), nil
}

func (c *CopilotAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

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

func (c *CopilotAgent) formatConversation(messages []agent.Message) string {
	parts := make([]string, 0, len(messages))

	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}

	return strings.Join(parts, "\n")
}

func (c *CopilotAgent) buildPrompt(conversation string) string {
	return BuildAgentPrompt(c.Name, c.Config.Prompt, conversation)
}

func init() {
	agent.RegisterFactory("copilot", NewCopilotAgent)
}
