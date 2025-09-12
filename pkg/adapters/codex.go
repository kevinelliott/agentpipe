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

type CodexAgent struct {
	agent.BaseAgent
	execPath string
}

func NewCodexAgent() agent.Agent {
	return &CodexAgent{}
}

func (c *CodexAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		return err
	}
	
	path, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("codex CLI not found: %w", err)
	}
	c.execPath = path
	
	return nil
}

func (c *CodexAgent) IsAvailable() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (c *CodexAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		return fmt.Errorf("codex CLI not initialized")
	}
	
	// Test with a simple version command
	cmd := exec.CommandContext(ctx, c.execPath, "--version")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		// Try help if version doesn't work
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("codex health check failed: %w", err)
		}
	}
	
	if len(output) == 0 {
		return fmt.Errorf("codex returned empty response")
	}
	
	return nil
}

func (c *CodexAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}
	
	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)
	
	args := []string{}
	
	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}
	
	// Add temperature if specified
	if c.Config.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", c.Config.Temperature))
	}
	
	// Add max tokens if specified
	if c.Config.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", c.Config.MaxTokens))
	}
	
	// Add the prompt
	args = append(args, prompt)
	
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "not found") {
			return "", fmt.Errorf("codex model not found - check model name in config: %s", c.Config.Model)
		}
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "unauthorized") {
			return "", fmt.Errorf("codex authentication failed - check API keys")
		}
		
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("codex execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		return "", fmt.Errorf("codex execution failed: %w\nOutput: %s", err, outputStr)
	}
	
	return strings.TrimSpace(string(output)), nil
}

func (c *CodexAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}
	
	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)
	
	args := []string{"--stream"}
	
	// Add model flag if specified
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}
	
	// Add temperature if specified
	if c.Config.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", c.Config.Temperature))
	}
	
	// Add max tokens if specified
	if c.Config.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", c.Config.MaxTokens))
	}
	
	// Add the prompt
	args = append(args, prompt)
	
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start codex: %w", err)
	}
	
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintln(writer, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}
	
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("codex execution failed: %w", err)
	}
	
	return nil
}

func (c *CodexAgent) formatConversation(messages []agent.Message) string {
	var parts []string
	
	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}
	
	return strings.Join(parts, "\n")
}

func (c *CodexAgent) buildPrompt(conversation string) string {
	var prompt strings.Builder
	
	if c.Config.Prompt != "" {
		prompt.WriteString(c.Config.Prompt)
		prompt.WriteString("\n\n")
	}
	
	prompt.WriteString("You are participating in a multi-agent conversation. ")
	prompt.WriteString(fmt.Sprintf("Your name is '%s'. ", c.Name))
	prompt.WriteString("Here is the conversation so far:\n\n")
	prompt.WriteString(conversation)
	prompt.WriteString("\n\nPlease provide your response:")
	
	return prompt.String()
}

func init() {
	agent.RegisterFactory("codex", NewCodexAgent)
}