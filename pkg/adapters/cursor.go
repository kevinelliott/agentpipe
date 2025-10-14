package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

type CursorAgent struct {
	agent.BaseAgent
	execPath string
}

func NewCursorAgent() agent.Agent {
	return &CursorAgent{}
}

func (c *CursorAgent) Initialize(config agent.AgentConfig) error {
	if err := c.BaseAgent.Initialize(config); err != nil {
		return err
	}

	path, err := exec.LookPath("cursor-agent")
	if err != nil {
		return fmt.Errorf("cursor-agent CLI not found: %w", err)
	}
	c.execPath = path

	return nil
}

func (c *CursorAgent) IsAvailable() bool {
	_, err := exec.LookPath("cursor-agent")
	return err == nil
}

func (c *CursorAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		return fmt.Errorf("cursor-agent CLI not initialized")
	}

	// Check if cursor-agent is available and authenticated
	cmd := exec.CommandContext(ctx, c.execPath, "status")
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	// Check if we need to login
	if strings.Contains(outputStr, "not logged in") || strings.Contains(outputStr, "Not authenticated") {
		return fmt.Errorf("cursor-agent not authenticated - please run 'cursor-agent login'")
	}

	if err != nil {
		// If status command failed but gave us output, check what it says
		if len(outputStr) > 0 {
			// If it contains "Logged in" it's actually working
			if strings.Contains(outputStr, "Logged in") || strings.Contains(outputStr, "Login successful") {
				return nil
			}
		}

		// Try with help flag as fallback
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		_, err = cmd.CombinedOutput()

		if err != nil {
			return fmt.Errorf("cursor-agent CLI not responding: %w", err)
		}
	}

	// Check if output indicates it's working
	if strings.Contains(outputStr, "Logged in") || strings.Contains(outputStr, "Login successful") || len(outputStr) > 10 {
		return nil
	}

	return fmt.Errorf("cursor-agent CLI health check failed")
}

func (c *CursorAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Use StreamMessage to handle the response properly
	var result strings.Builder
	err := c.StreamMessage(ctx, messages, &result)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func (c *CursorAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	conversation := c.formatConversation(messages)
	prompt := c.buildPrompt(conversation)

	// Create a context with timeout for streaming
	// cursor-agent needs more time to respond (typically 10-15 seconds)
	streamCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Use --print mode for streaming
	// cursor-agent reads prompt from stdin and outputs JSON stream
	cmd := exec.CommandContext(streamCtx, c.execPath, "--print")
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cursor-agent: %w", err)
	}

	// Read stderr in background to capture any errors
	var stderrBuf strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text())
			stderrBuf.WriteString("\n")
		}
	}()

	hasOutput := false
	scanner := bufio.NewScanner(stdout)
	var streamedContent strings.Builder

	// Set a deadline for reading
	readDeadline := time.After(25 * time.Second)

scanLoop:
	for scanner.Scan() {
		select {
		case <-readDeadline:
			// Reading timeout - stop processing
			break scanLoop
		default:
			line := scanner.Text()

			// Check for result message which signals completion
			if result := c.parseResultLine(line); result != "" {
				// If we get a complete result, only use it if we haven't streamed content
				if streamedContent.Len() == 0 {
					_, _ = fmt.Fprint(writer, result)
				}
				hasOutput = true
				break scanLoop
			}

			// Otherwise stream assistant messages
			if text := c.parseJSONLine(line); text != "" {
				_, _ = fmt.Fprint(writer, text)
				streamedContent.WriteString(text)
				hasOutput = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// Kill the process before returning error
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		return fmt.Errorf("error reading output: %w", err)
	}

	// Kill the process if it's still running (cursor-agent doesn't terminate on its own)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() // Clean up the process
	}

	// Check if we got any output
	if !hasOutput {
		stderrOutput := stderrBuf.String()
		if stderrOutput != "" {
			return fmt.Errorf("cursor-agent produced no output. Stderr: %s", stderrOutput)
		}
		return fmt.Errorf("cursor-agent produced no output")
	}

	return nil
}

func (c *CursorAgent) formatConversation(messages []agent.Message) string {
	parts := make([]string, 0, len(messages))

	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}

	return strings.Join(parts, "\n")
}

func (c *CursorAgent) buildPrompt(conversation string) string {
	return BuildAgentPrompt(c.Name, c.Config.Prompt, conversation)
}

// parseResultLine checks for a result message which contains the complete response
func (c *CursorAgent) parseResultLine(line string) string {
	var result struct {
		Type   string `json:"type"`
		Result string `json:"result"`
	}

	if err := json.Unmarshal([]byte(line), &result); err != nil {
		return ""
	}

	if result.Type == "result" {
		return result.Result
	}

	return ""
}

// parseJSONLine parses a single JSON line from cursor-agent output
func (c *CursorAgent) parseJSONLine(line string) string {
	if line == "" {
		return ""
	}

	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	}

	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return ""
	}

	// Only process assistant messages
	if msg.Type != "assistant" {
		return ""
	}

	// Extract text from content
	for _, content := range msg.Message.Content {
		if content.Type == "text" {
			return content.Text
		}
	}

	return ""
}

func init() {
	agent.RegisterFactory("cursor", NewCursorAgent)
}
