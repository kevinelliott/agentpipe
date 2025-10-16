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
	"github.com/kevinelliott/agentpipe/pkg/log"
)

const (
	// Cursor-specific timeout constants
	cursorStreamTimeout = 30 * time.Second
	cursorReadDeadline  = 25 * time.Second
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
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("cursor agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("cursor-agent")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   c.ID,
			"agent_name": c.Name,
		}).WithError(err).Error("cursor-agent CLI not found in PATH")
		return fmt.Errorf("cursor-agent CLI not found: %w", err)
	}
	c.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   c.ID,
		"agent_name": c.Name,
		"exec_path":  path,
		"model":      c.Config.Model,
	}).Info("cursor agent initialized successfully")

	return nil
}

func (c *CursorAgent) IsAvailable() bool {
	_, err := exec.LookPath("cursor-agent")
	return err == nil
}

func (c *CursorAgent) HealthCheck(ctx context.Context) error {
	if c.execPath == "" {
		log.WithField("agent_name", c.Name).Error("cursor health check failed: not initialized")
		return fmt.Errorf("cursor-agent CLI not initialized")
	}

	log.WithField("agent_name", c.Name).Debug("starting cursor health check")

	// Check if cursor-agent is available and authenticated
	cmd := exec.CommandContext(ctx, c.execPath, "status")
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	// Check if we need to login
	if strings.Contains(outputStr, "not logged in") || strings.Contains(outputStr, "Not authenticated") {
		log.WithField("agent_name", c.Name).Error("cursor health check failed: not authenticated")
		return fmt.Errorf("cursor-agent not authenticated - please run 'cursor-agent login'")
	}

	if err != nil {
		// If status command failed but gave us output, check what it says
		if len(outputStr) > 0 {
			// If it contains "Logged in" it's actually working
			if strings.Contains(outputStr, "Logged in") || strings.Contains(outputStr, "Login successful") {
				log.WithField("agent_name", c.Name).Info("cursor health check passed")
				return nil
			}
		}

		// Try with help flag as fallback
		log.WithField("agent_name", c.Name).Debug("status check failed, trying --help")
		cmd = exec.CommandContext(ctx, c.execPath, "--help")
		_, err = cmd.CombinedOutput()

		if err != nil {
			log.WithField("agent_name", c.Name).WithError(err).Error("cursor health check failed: CLI not responding")
			return fmt.Errorf("cursor-agent CLI not responding: %w", err)
		}
	}

	// Check if output indicates it's working
	if strings.Contains(outputStr, "Logged in") || strings.Contains(outputStr, "Login successful") || len(outputStr) > 10 {
		log.WithField("agent_name", c.Name).Info("cursor health check passed")
		return nil
	}

	log.WithField("agent_name", c.Name).Error("cursor health check failed: unknown status")
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

	log.WithFields(map[string]interface{}{
		"agent_name":    c.Name,
		"message_count": len(messages),
		"timeout":       cursorStreamTimeout.String(),
	}).Debug("starting cursor streaming message")

	// Filter out this agent's own messages
	relevantMessages := c.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := c.buildPrompt(relevantMessages, true)

	// Create a context with timeout for streaming
	// cursor-agent needs more time to respond (typically 10-15 seconds)
	streamCtx, cancel := context.WithTimeout(ctx, cursorStreamTimeout)
	defer cancel()

	// Use --print mode for streaming
	// cursor-agent reads prompt from stdin and outputs JSON stream
	cmd := exec.CommandContext(streamCtx, c.execPath, "--print")
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to create stderr pipe")
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", c.Name).WithError(err).Error("failed to start cursor-agent process")
		return fmt.Errorf("failed to start cursor-agent: %w", err)
	}

	// Read stderr in background to capture any errors
	var stderrBuf strings.Builder
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-streamCtx.Done():
				return
			default:
				stderrBuf.WriteString(scanner.Text())
				stderrBuf.WriteString("\n")
			}
		}
	}()

	hasOutput := false
	scanner := bufio.NewScanner(stdout)
	var streamedContent strings.Builder

	// Set a deadline for reading - use NewTimer so we can stop it
	readTimer := time.NewTimer(cursorReadDeadline)
	defer readTimer.Stop()

scanLoop:
	for scanner.Scan() {
		select {
		case <-readTimer.C:
			// Reading timeout - stop processing
			break scanLoop
		case <-streamCtx.Done():
			// Context canceled - stop processing
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
		log.WithField("agent_name", c.Name).WithError(err).Error("error reading cursor streaming output")
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
		log.WithFields(map[string]interface{}{
			"agent_name": c.Name,
			"stderr":     stderrOutput,
		}).Error("cursor produced no output")
		if stderrOutput != "" {
			return fmt.Errorf("cursor-agent produced no output. Stderr: %s", stderrOutput)
		}
		return fmt.Errorf("cursor-agent produced no output")
	}

	log.WithFields(map[string]interface{}{
		"agent_name":     c.Name,
		"content_length": streamedContent.Len(),
	}).Info("cursor streaming message completed")

	return nil
}

func (c *CursorAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		// Exclude this agent's own messages
		if msg.AgentName == c.Name || msg.AgentID == c.ID {
			continue
		}
		relevant = append(relevant, msg)
	}
	return relevant
}

func (c *CursorAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", c.Name))

	if c.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(c.Config.Prompt)
		prompt.WriteString("\n\n")
	}

	// PART 2: CONVERSATION CONTEXT
	if len(messages) > 0 {
		var initialPrompt string
		var otherMessages []agent.Message

		// Find the orchestrator's initial prompt (AgentID="system")
		// vs agent announcements (system messages from specific agents)
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentName == "System") && initialPrompt == "" {
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", c.Name))
		}
	}

	return prompt.String()
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
