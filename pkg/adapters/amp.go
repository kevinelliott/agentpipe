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
	ampStreamTimeout = 60 * time.Second
	ampReadDeadline  = 55 * time.Second
	ampHealthTimeout = 5 * time.Second
)

// AmpAgent represents the Amp coding agent adapter
type AmpAgent struct {
	agent.BaseAgent
	execPath       string
	threadID       string // Current Amp thread ID for conversation continuity
	lastMessageIdx int    // Index of last message sent to Amp (for incremental updates)
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
		"thread_id":     a.threadID,
		"last_msg_idx":  a.lastMessageIdx,
	}).Debug("sending message to amp CLI")

	// Get only new messages that haven't been sent to Amp yet
	// IMPORTANT: Filter out this agent's own messages since Amp maintains them in the thread
	newMessages := a.filterRelevantMessages(messages[a.lastMessageIdx:])
	if len(newMessages) == 0 {
		log.WithField("agent_name", a.Name).Debug("no new messages to send (all filtered)")
		return "", nil
	}

	var output string
	var err error
	startTime := time.Now()

	if a.threadID == "" {
		// Create a new thread with the initial conversation context
		// For initial thread, send ALL messages except this agent's own
		allRelevantMessages := a.filterRelevantMessages(messages)
		output, err = a.createThread(ctx, allRelevantMessages, newMessages)
	} else {
		// Continue existing thread with just the new messages from OTHER agents
		output, err = a.continueThread(ctx, newMessages)
	}

	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_name": a.Name,
			"duration":   duration.String(),
			"thread_id":  a.threadID,
		}).WithError(err).Error("amp message failed")
		return "", err
	}

	// Update the index of last sent message
	a.lastMessageIdx = len(messages)

	log.WithFields(map[string]interface{}{
		"agent_name":    a.Name,
		"duration":      duration.String(),
		"response_size": len(output),
		"thread_id":     a.threadID,
	}).Info("amp message sent successfully")

	return output, nil
}

// filterRelevantMessages filters out this agent's own messages
// Since Amp maintains thread context server-side, we should NOT send:
// 1. This agent's own responses (Amp already knows what it said)
// 2. Only send messages from OTHER agents and system messages
func (a *AmpAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip this agent's own messages - Amp already has them in the thread
		if msg.AgentName == a.Name || msg.AgentID == a.ID {
			continue
		}
		// Include messages from other agents and system messages
		relevant = append(relevant, msg)
	}

	return relevant
}

// createThread creates a new Amp thread with initial context
func (a *AmpAgent) createThread(ctx context.Context, allMessages, newMessages []agent.Message) (string, error) {
	// Count system messages to verify initial prompt is included
	systemMsgCount := 0
	for _, msg := range allMessages {
		if msg.Role == "system" || strings.ToLower(msg.AgentName) == "system" {
			systemMsgCount++
		}
	}

	log.WithFields(map[string]interface{}{
		"agent_name":        a.Name,
		"filtered_messages": len(allMessages),
		"system_messages":   systemMsgCount,
		"has_custom_prompt": a.Config.Prompt != "",
		"custom_prompt_len": len(a.Config.Prompt),
	}).Info("creating new amp thread with relevant conversation context")

	// Create an empty thread first, then send the initial request as thread continue
	// This avoids the issue of amp thread new not returning a response
	cmd := exec.CommandContext(ctx, a.execPath, "thread", "new")
	cmd.Stdin = strings.NewReader("") // Empty thread creation

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": a.Name,
				"exit_code":  exitErr.ExitCode(),
			}).WithError(err).Error("amp thread new failed")
			return "", fmt.Errorf("amp thread new failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		return "", fmt.Errorf("amp thread new failed: %w\nOutput: %s", err, string(output))
	}

	// Parse output to extract thread ID
	a.threadID = strings.TrimSpace(string(output))
	log.WithFields(map[string]interface{}{
		"agent_name": a.Name,
		"thread_id":  a.threadID,
	}).Info("amp thread created, sending initial request")

	// Build the prompt with proper structure
	// This includes:
	// 1. Agent's custom system prompt (a.Config.Prompt) - sent FIRST
	// 2. Initial orchestrator prompt - highlighted prominently
	// 3. Messages from OTHER agents (excluding this agent's own responses)
	// NOTE: allMessages is already filtered by caller to exclude this agent's messages
	prompt := a.buildPrompt(allMessages, true) // isInitialThread = true

	log.WithFields(map[string]interface{}{
		"agent_name":      a.Name,
		"message_count":   len(allMessages),
		"full_prompt_len": len(prompt),
	}).Debug("amp thread context prepared")

	// Log a preview of what we're sending for debugging
	if len(prompt) > 0 {
		previewLen := 500
		if len(prompt) < previewLen {
			previewLen = len(prompt)
		}
		log.WithFields(map[string]interface{}{
			"agent_name":   a.Name,
			"prompt_start": prompt[:previewLen],
		}).Debug("amp initial prompt preview")
	}

	// Now send the initial request as thread continue
	continueCmd := exec.CommandContext(ctx, a.execPath, "thread", "continue", a.threadID)
	continueCmd.Stdin = strings.NewReader(prompt)

	continueOutput, err := continueCmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": a.Name,
				"thread_id":  a.threadID,
				"exit_code":  exitErr.ExitCode(),
			}).WithError(err).Error("amp thread continue failed with initial request")
			return "", fmt.Errorf("amp thread continue failed (exit code %d): %s", exitErr.ExitCode(), string(continueOutput))
		}
		return "", fmt.Errorf("amp thread continue failed: %w\nOutput: %s", err, string(continueOutput))
	}

	return string(continueOutput), nil
}

// continueThread continues an existing Amp thread with new messages
func (a *AmpAgent) continueThread(ctx context.Context, newMessages []agent.Message) (string, error) {
	log.WithFields(map[string]interface{}{
		"agent_name":        a.Name,
		"thread_id":         a.threadID,
		"new_message_count": len(newMessages),
	}).Info("continuing amp thread with incremental messages only")

	// IMPORTANT: Only send NEW messages that haven't been sent yet
	// Amp maintains the full conversation context server-side in the thread
	// We only need to send the delta (new messages since last interaction)
	prompt := a.buildPrompt(newMessages, false) // isInitialThread = false

	// Continue thread: amp thread continue {thread_id}
	cmd := exec.CommandContext(ctx, a.execPath, "thread", "continue", a.threadID)
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": a.Name,
				"thread_id":  a.threadID,
				"exit_code":  exitErr.ExitCode(),
			}).WithError(err).Error("amp thread continue failed")
			return "", fmt.Errorf("amp thread continue failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		return "", fmt.Errorf("amp thread continue failed: %w\nOutput: %s", err, string(output))
	}

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
		"thread_id":     a.threadID,
		"last_msg_idx":  a.lastMessageIdx,
		"timeout":       ampStreamTimeout.String(),
	}).Debug("starting amp streaming message")

	// Get only new messages that haven't been sent to Amp yet
	// IMPORTANT: Filter out this agent's own messages since Amp maintains them in the thread
	newMessages := a.filterRelevantMessages(messages[a.lastMessageIdx:])
	if len(newMessages) == 0 {
		log.WithField("agent_name", a.Name).Debug("no new messages to stream (all filtered)")
		return nil
	}

	// Create a context with timeout for streaming
	streamCtx, cancel := context.WithTimeout(ctx, ampStreamTimeout)
	defer cancel()

	var cmd *exec.Cmd
	var prompt string

	if a.threadID == "" {
		// For initial thread, send ALL messages except this agent's own
		allRelevantMessages := a.filterRelevantMessages(messages)

		// Count system messages to verify initial prompt is included
		systemMsgCount := 0
		for _, msg := range allRelevantMessages {
			if msg.Role == "system" || strings.ToLower(msg.AgentName) == "system" {
				systemMsgCount++
			}
		}

		log.WithFields(map[string]interface{}{
			"agent_name":        a.Name,
			"total_messages":    len(messages),
			"filtered_messages": len(allRelevantMessages),
			"system_messages":   systemMsgCount,
			"has_custom_prompt": a.Config.Prompt != "",
			"mode":              "streaming",
		}).Info("creating new amp thread (streaming) with relevant conversation context")

		// IMPORTANT: Build prompt with proper structure
		// Agent setup and role come FIRST, then conversation context
		prompt = a.buildPrompt(allRelevantMessages, true) // isInitialThread = true

		log.WithFields(map[string]interface{}{
			"agent_name":      a.Name,
			"message_count":   len(allRelevantMessages),
			"full_prompt_len": len(prompt),
		}).Debug("amp streaming thread context prepared")

		// Log a preview of what we're sending for debugging
		if len(prompt) > 0 {
			previewLen := 300
			if len(prompt) < previewLen {
				previewLen = len(prompt)
			}
			log.WithFields(map[string]interface{}{
				"agent_name":   a.Name,
				"prompt_start": prompt[:previewLen],
			}).Debug("amp initial streaming prompt preview")
		}

		// Use --stream-json with thread new
		cmd = exec.CommandContext(streamCtx, a.execPath, "thread", "new", "--stream-json")
	} else {
		// Continue existing thread with just new messages
		log.WithFields(map[string]interface{}{
			"agent_name":        a.Name,
			"thread_id":         a.threadID,
			"new_message_count": len(newMessages),
			"mode":              "streaming-continue",
		}).Debug("continuing amp thread with new messages only")

		prompt = a.buildPrompt(newMessages, false) // isInitialThread = false
		// Use --stream-json with thread continue
		cmd = exec.CommandContext(streamCtx, a.execPath, "thread", "continue", a.threadID, "--stream-json")
	}

	cmd.Stdin = strings.NewReader(prompt)

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
	isFirstLine := a.threadID == "" // Track if we need to extract thread ID from first line

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

			// If this is a new thread, first line should be the thread ID
			if isFirstLine {
				// Try to extract thread ID from first line
				var threadInfo struct {
					ThreadID string `json:"thread_id"`
					ID       string `json:"id"`
				}
				if err := json.Unmarshal([]byte(line), &threadInfo); err == nil {
					if threadInfo.ThreadID != "" {
						a.threadID = threadInfo.ThreadID
						log.WithFields(map[string]interface{}{
							"agent_name": a.Name,
							"thread_id":  a.threadID,
						}).Info("amp thread created from streaming")
					} else if threadInfo.ID != "" {
						a.threadID = threadInfo.ID
						log.WithFields(map[string]interface{}{
							"agent_name": a.Name,
							"thread_id":  a.threadID,
						}).Info("amp thread created from streaming")
					}
				}
				isFirstLine = false
				// Don't write thread ID line to output, continue to next line
				continue
			}

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

	// Update the index of last sent message
	a.lastMessageIdx = len(messages)

	duration := time.Since(startTime)
	log.WithFields(map[string]interface{}{
		"agent_name":     a.Name,
		"duration":       duration.String(),
		"content_length": streamedContent.Len(),
		"thread_id":      a.threadID,
	}).Info("amp streaming message completed")

	return nil
}

// buildPrompt creates the final prompt for Amp with explicit context
// For initial threads, we need to send setup BEFORE conversation to avoid confusion
func (a *AmpAgent) buildPrompt(messages []agent.Message, isInitialThread bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE (always first)
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", a.Name))

	// Include custom system prompt if provided
	if a.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(a.Config.Prompt)
		prompt.WriteString("\n")
	}
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n\n")

	// PART 2: CONVERSATION CONTEXT (after role is established)
	if isInitialThread && len(messages) > 0 {
		// When agent comes online for the first time, deliver ALL existing messages
		// This includes: initial system prompt, any other system messages, and all agent messages
		var initialPrompt string
		var otherMessages []agent.Message

		// IMPORTANT: Find the orchestrator's initial prompt (AgentID/AgentName = "system" or "host")
		// Agent announcements are also system messages, but they come from specific agents
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
				// This is the orchestrator's initial prompt - show it prominently
				initialPrompt = msg.Content
			} else {
				// ALL other messages (agent announcements, other system messages, agent responses)
				otherMessages = append(otherMessages, msg)
			}
		}

		// Show the initial prompt as a DIRECT INSTRUCTION
		if initialPrompt != "" {
			prompt.WriteString("YOUR TASK - PLEASE RESPOND TO THIS:\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n")
			prompt.WriteString(initialPrompt)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n\n")
		}

		// Then show ALL remaining conversation (system messages + agent messages)
		if len(otherMessages) > 0 {
			prompt.WriteString("CONVERSATION SO FAR:\n")
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n")
			for _, msg := range otherMessages {
				timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
				// Include role indicator for system messages to make them clear
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.", a.Name))
		} else {
			prompt.WriteString(fmt.Sprintf("Now, as %s, respond to the conversation.", a.Name))
		}
	} else if len(messages) > 0 {
		// For continuation, simpler format - just show new messages
		prompt.WriteString("NEW MESSAGES:\n")
		prompt.WriteString(strings.Repeat("-", 60))
		prompt.WriteString("\n")
		for _, msg := range messages {
			timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
			if msg.Role == "system" {
				prompt.WriteString(fmt.Sprintf("[%s] SYSTEM: %s\n", timestamp, msg.Content))
			} else {
				prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.AgentName, msg.Content))
			}
		}
		prompt.WriteString(strings.Repeat("-", 60))
		prompt.WriteString("\n\n")
		prompt.WriteString(fmt.Sprintf("Continue the conversation as %s.", a.Name))
	} else {
		prompt.WriteString(fmt.Sprintf("Start the conversation as %s.", a.Name))
	}

	return prompt.String()
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
