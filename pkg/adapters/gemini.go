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

type GeminiAgent struct {
	agent.BaseAgent
	execPath string
}

func NewGeminiAgent() agent.Agent {
	return &GeminiAgent{}
}

func (g *GeminiAgent) Initialize(config agent.AgentConfig) error {
	if err := g.BaseAgent.Initialize(config); err != nil {
		return err
	}

	path, err := exec.LookPath("gemini")
	if err != nil {
		return fmt.Errorf("gemini CLI not found: %w", err)
	}
	g.execPath = path

	return nil
}

func (g *GeminiAgent) IsAvailable() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (g *GeminiAgent) HealthCheck(ctx context.Context) error {
	if g.execPath == "" {
		return fmt.Errorf("gemini CLI not initialized")
	}

	// Gemini takes longer to start, so we'll just check if the binary exists
	// and can show help/version info
	cmd := exec.CommandContext(ctx, g.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Gemini might be interactive and not support --help well
		// Just check if we can execute it at all
		testCmd := exec.Command(g.execPath, "--version")
		if err := testCmd.Start(); err != nil {
			return fmt.Errorf("gemini CLI cannot be executed: %w", err)
		}
		// Kill the process if it's still running
		if testCmd.Process != nil {
			_ = testCmd.Process.Kill()
			_ = testCmd.Wait() // Clean up the process
		}
		// If we can start it, consider it healthy
		return nil
	}

	// Check if output looks like gemini help
	if len(output) < 50 {
		return fmt.Errorf("gemini CLI returned suspiciously short help output")
	}

	return nil
}

func (g *GeminiAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	conversation := g.formatConversation(messages)
	prompt := g.buildPrompt(conversation)

	// Gemini CLI expects the prompt as a positional argument after any flags
	args := []string{}

	// Add model flag if specified
	if g.Config.Model != "" {
		args = append(args, "--model", g.Config.Model)
	}

	// Add the prompt as the last argument
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, g.execPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check for specific error patterns
		outputStr := string(output)
		if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "NOT_FOUND") {
			return "", fmt.Errorf("gemini model not found - check model name in config: %s", g.Config.Model)
		}
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "UNAUTHENTICATED") {
			return "", fmt.Errorf("gemini authentication failed - check API keys")
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			// Try to extract a meaningful error message
			if strings.Contains(outputStr, "error") {
				// Extract JSON error if present
				if start := strings.Index(outputStr, `"message":`); start != -1 {
					if end := strings.Index(outputStr[start:], `",`); end != -1 {
						errMsg := outputStr[start+11 : start+end-1]
						return "", fmt.Errorf("gemini API error: %s", errMsg)
					}
				}
			}
			return "", fmt.Errorf("gemini execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
		}
		return "", fmt.Errorf("gemini execution failed: %w\nOutput: %s", err, outputStr)
	}

	// Clean up output
	outputStr := string(output)

	// Remove common prefixes
	lines := strings.Split(outputStr, "\n")
	cleanedLines := []string{}
	for _, line := range lines {
		// Skip system messages
		if strings.Contains(line, "Loaded cached credentials") ||
			strings.Contains(line, "To authenticate") ||
			strings.HasPrefix(line, "Gemini CLI") {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n")), nil
}

func (g *GeminiAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	conversation := g.formatConversation(messages)
	prompt := g.buildPrompt(conversation)

	// Use stdin for the prompt
	cmd := exec.CommandContext(ctx, g.execPath)
	if g.Config.Model != "" {
		cmd = exec.CommandContext(ctx, g.execPath, "--model", g.Config.Model)
	}
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gemini: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	skipFirst := true
	for scanner.Scan() {
		line := scanner.Text()
		// Skip the "Loaded cached credentials" line
		if skipFirst && strings.Contains(line, "Loaded cached credentials") {
			skipFirst = false
			continue
		}
		fmt.Fprintln(writer, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("gemini execution failed: %w", err)
	}

	return nil
}

func (g *GeminiAgent) formatConversation(messages []agent.Message) string {
	parts := make([]string, 0, len(messages))

	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}

	return strings.Join(parts, "\n")
}

func (g *GeminiAgent) buildPrompt(conversation string) string {
	return BuildAgentPrompt(g.Name, g.Config.Prompt, conversation)
}

func init() {
	agent.RegisterFactory("gemini", NewGeminiAgent)
}
