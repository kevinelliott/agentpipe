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
	// Amp-specific timeout constants
	ampStreamTimeout  = 60 * time.Second
	ampReadDeadline   = 55 * time.Second
	ampHealthTimeout  = 5 * time.Second
)

// AmpAgent represents the Amp coding agent adapter
type AmpAgent struct {
	agent.BaseAgent
	execPath string
}

// NewAmpAgent creates a new Amp agent instance
func NewAmpAgent() agent.Agent {
	return &AmpAgent{}
}

// Initialize sets up the Amp agent with the provided configuration
func (a *AmpAgent) Initialize(config agent.AgentConfig) error {
	if err := a.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("amp agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("amp")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   a.ID,
			"agent_name": a.Name,
		}).WithError(err).Error("amp CLI not found in PATH")
		return fmt.Errorf("amp CLI not found: %w", err)
	}
	a.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   a.ID,
		"agent_name": a.Name,
		"exec_path":  path,
		"model":      a.Config.Model,
	}).Info("amp agent initialized successfully")

	return nil
}

// IsAvailable checks if the Amp CLI is available in the system PATH
func (a *AmpAgent) IsAvailable() bool {
	_, err := exec.LookPath("amp")
	return err == nil
}

// HealthCheck verifies that the Amp CLI is installed and functional
func (a *AmpAgent) HealthCheck(ctx context.Context) error {
	if a.execPath == "" {
		log.WithField("agent_name", a.Name).Error("amp health check failed: not initialized")
		return fmt.Errorf("amp CLI not initialized")
	}

	log.WithField("agent_name", a.Name).Debug("starting amp health check")

	// Create a context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, ampHealthTimeout)
	defer cancel()

	// Check if amp CLI responds to --help flag
	cmd := exec.CommandContext(healthCtx, a.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("amp health check failed: CLI not responding to --help")
		return fmt.Errorf("amp CLI not responding to --help: %w", err)
	}

	// Check if output contains expected content
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    a.Name,
			"output_length": len(outputStr),
		}).Error("amp health check failed: output too short")
		return fmt.Errorf("amp CLI returned suspiciously short output")
	}

	// Verify it's actually Amp by checking for key terms
	if !strings.Contains(strings.ToLower(outputStr), "amp") && !strings.Contains(strings.ToLower(outputStr), "execute") {
		log.WithField("agent_name", a.Name).Error("amp health check failed: output doesn't appear to be from Amp CLI")
		return fmt.Errorf("CLI at path doesn't appear to be Amp")
	}

	log.WithField("agent_name", a.Name).Info("amp health check passed")
	return nil
}

// SendMessage sends a message to the Amp CLI and returns the response
func (a *AmpAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"message_count": len(messages),
	}).Debug("sending message to amp CLI")

	conversation := a.formatConversation(messages)
	prompt := a.buildPrompt(conversation)

	// Use -x flag for execute mode
	cmd := exec.CommandContext(ctx, a.execPath, "-x", prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": a.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("amp execution failed with exit code")
			return "", fmt.Errorf("amp execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		log.WithFields(map[string]interface{}{
			"agent_name": a.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("amp execution failed")
		return "", fmt.Errorf("amp execution failed: %w\nOutput: %s", err, string(output))
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("amp message sent successfully")

	return string(output), nil
}

// StreamMessage sends a message to Amp CLI and streams the response
func (a *AmpAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"message_count": len(messages),
		"timeout":       ampStreamTimeout.String(),
	}).Debug("starting amp streaming message")

	conversation := a.formatConversation(messages)
	prompt := a.buildPrompt(conversation)

	// Create a context with timeout for streaming
	streamCtx, cancel := context.WithTimeout(ctx, ampStreamTimeout)
	defer cancel()

	// Use --stream-json and -x flags for streaming JSON output
	cmd := exec.CommandContext(streamCtx, a.execPath, "--stream-json", "-x", prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("failed to create stderr pipe")
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("failed to start amp process")
		return fmt.Errorf("failed to start amp: %w", err)
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

	startTime := time.Now()
	hasOutput := false
	scanner := bufio.NewScanner(stdout)
	var streamedContent strings.Builder

	// Set a deadline for reading
	readTimer := time.NewTimer(ampReadDeadline)
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

			// Parse the JSON line and extract text content
			if text := a.parseJSONLine(line); text != "" {
				_, _ = fmt.Fprint(writer, text)
				streamedContent.WriteString(text)
				hasOutput = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", a.Name).WithError(err).Error("error reading amp streaming output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		// Only log as error if we didn't get any output
		if !hasOutput {
			log.WithField("agent_name", a.Name).WithError(err).Error("amp streaming execution failed")
			return fmt.Errorf("amp execution failed: %w", err)
		}
		// If we got output, just log as debug (some CLIs exit with non-zero after Ctrl+C)
		log.WithField("agent_name", a.Name).WithError(err).Debug("amp process exited with error but produced output")
	}

	// Check if we got any output
	if !hasOutput {
		stderrOutput := stderrBuf.String()
		log.WithFields(map[string]interface{}{
			"agent_name": a.Name,
			"stderr":     stderrOutput,
		}).Error("amp produced no output")
		if stderrOutput != "" {
			return fmt.Errorf("amp produced no output. Stderr: %s", stderrOutput)
		}
		return fmt.Errorf("amp produced no output")
	}

	duration := time.Since(startTime)
	log.WithFields(map[string]interface{}{
		"agent_name":     a.Name,
		"duration":       duration.String(),
		"content_length": streamedContent.Len(),
	}).Info("amp streaming message completed")

	return nil
}

// formatConversation formats the conversation history for Amp
func (a *AmpAgent) formatConversation(messages []agent.Message) string {
	parts := make([]string, 0, len(messages))

	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}

	return strings.Join(parts, "\n")
}

// buildPrompt creates the final prompt for Amp
func (a *AmpAgent) buildPrompt(conversation string) string {
	return BuildAgentPrompt(a.Name, a.Config.Prompt, conversation)
}

// parseJSONLine parses a single JSON line from amp --stream-json output
func (a *AmpAgent) parseJSONLine(line string) string {
	if line == "" {
		return ""
	}

	// Amp's --stream-json format (need to verify exact structure)
	// Try common JSON streaming formats
	var msg struct {
		Type    string `json:"type"`
		Content string `json:"content"`
		Text    string `json:"text"`
		Message string `json:"message"`
		Delta   struct {
			Content string `json:"content"`
			Text    string `json:"text"`
		} `json:"delta"`
	}

	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		// If it's not JSON, treat it as plain text
		return line + "\n"
	}

	// Try different possible fields where content might be
	if msg.Content != "" {
		return msg.Content
	}
	if msg.Text != "" {
		return msg.Text
	}
	if msg.Message != "" {
		return msg.Message
	}
	if msg.Delta.Content != "" {
		return msg.Delta.Content
	}
	if msg.Delta.Text != "" {
		return msg.Delta.Text
	}

	return ""
}

func init() {
	agent.RegisterFactory("amp", NewAmpAgent)
}
