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

type GeminiAgent struct {
	agent.BaseAgent
	execPath string
}

func NewGeminiAgent() agent.Agent {
	return &GeminiAgent{}
}

func (g *GeminiAgent) Initialize(config agent.AgentConfig) error {
	if err := g.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("gemini agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("gemini")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   g.ID,
			"agent_name": g.Name,
		}).WithError(err).Error("gemini CLI not found in PATH")
		return fmt.Errorf("gemini CLI not found: %w", err)
	}
	g.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   g.ID,
		"agent_name": g.Name,
		"exec_path":  path,
		"model":      g.Config.Model,
	}).Info("gemini agent initialized successfully")

	return nil
}

func (g *GeminiAgent) IsAvailable() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (g *GeminiAgent) HealthCheck(ctx context.Context) error {
	if g.execPath == "" {
		log.WithField("agent_name", g.Name).Error("gemini health check failed: not initialized")
		return fmt.Errorf("gemini CLI not initialized")
	}

	log.WithField("agent_name", g.Name).Debug("starting gemini health check")

	// Gemini takes longer to start, so we'll just check if the binary exists
	// and can show help/version info
	cmd := exec.CommandContext(ctx, g.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Gemini might be interactive and not support --help well
		// Just check if we can execute it at all
		log.WithField("agent_name", g.Name).Debug("--help check failed, trying --version")
		testCmd := exec.Command(g.execPath, "--version")
		if err := testCmd.Start(); err != nil {
			log.WithField("agent_name", g.Name).WithError(err).Error("gemini health check failed: CLI not responding")
			return fmt.Errorf("gemini CLI cannot be executed: %w", err)
		}
		// Kill the process if it's still running
		if testCmd.Process != nil {
			_ = testCmd.Process.Kill()
			_ = testCmd.Wait() // Clean up the process
		}
		// If we can start it, consider it healthy
		log.WithField("agent_name", g.Name).Info("gemini health check passed")
		return nil
	}

	// Check if output looks like gemini help
	if len(output) < 50 {
		log.WithField("agent_name", g.Name).Error("gemini health check failed: suspiciously short help output")
		return fmt.Errorf("gemini CLI returned suspiciously short help output")
	}

	log.WithField("agent_name", g.Name).Info("gemini health check passed")
	return nil
}

func (g *GeminiAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.Name,
		"message_count": len(messages),
	}).Debug("sending message to gemini CLI")

	// Filter out this agent's own messages
	relevantMessages := g.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := g.buildPrompt(relevantMessages, true)

	// Build command args
	args := []string{}

	// Add model flag if specified
	if g.Config.Model != "" {
		args = append(args, "--model", g.Config.Model)
	}

	// Use stdin for the prompt to avoid terminal detection issues
	cmd := exec.CommandContext(ctx, g.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Convert output to string for analysis
	outputStr := string(output)

	// Check if we have valid output even if there was an error
	// Gemini CLI sometimes produces output but doesn't exit cleanly
	hasValidOutput := len(outputStr) > 0 && !strings.Contains(outputStr, "404") &&
		!strings.Contains(outputStr, "NOT_FOUND") && !strings.Contains(outputStr, "401") &&
		!strings.Contains(outputStr, "UNAUTHENTICATED")

	if err != nil {
		// If we have valid output, accept it even with process errors
		// This handles cases where Gemini CLI doesn't exit cleanly but produces valid responses
		if hasValidOutput {
			log.WithFields(map[string]interface{}{
				"agent_name": g.Name,
				"duration":   duration.String(),
				"exit_error": err.Error(),
			}).Debug("gemini had exit error but produced valid output, accepting response")
			// Continue to output processing below
		} else {
			// No valid output, treat as real error
			if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "NOT_FOUND") {
				log.WithFields(map[string]interface{}{
					"agent_name": g.Name,
					"model":      g.Config.Model,
					"duration":   duration.String(),
				}).WithError(err).Error("gemini model not found")
				return "", fmt.Errorf("gemini model not found - check model name in config: %s", g.Config.Model)
			}
			if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "UNAUTHENTICATED") {
				log.WithFields(map[string]interface{}{
					"agent_name": g.Name,
					"duration":   duration.String(),
				}).WithError(err).Error("gemini authentication failed")
				return "", fmt.Errorf("gemini authentication failed - check API keys")
			}

			if exitErr, ok := err.(*exec.ExitError); ok {
				// Try to extract a meaningful error message
				if strings.Contains(outputStr, "error") {
					// Extract JSON error if present
					if start := strings.Index(outputStr, `"message":`); start != -1 {
						if end := strings.Index(outputStr[start:], `",`); end != -1 {
							errMsg := outputStr[start+11 : start+end-1]
							log.WithFields(map[string]interface{}{
								"agent_name": g.Name,
								"duration":   duration.String(),
								"error_msg":  errMsg,
							}).Error("gemini API error")
							return "", fmt.Errorf("gemini API error: %s", errMsg)
						}
					}
				}
				log.WithFields(map[string]interface{}{
					"agent_name": g.Name,
					"exit_code":  exitErr.ExitCode(),
					"duration":   duration.String(),
				}).WithError(err).Error("gemini execution failed with exit code")
				return "", fmt.Errorf("gemini execution failed (exit code %d): %s", exitErr.ExitCode(), outputStr)
			}
			log.WithFields(map[string]interface{}{
				"agent_name": g.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("gemini execution failed")
			return "", fmt.Errorf("gemini execution failed: %w\nOutput: %s", err, outputStr)
		}
	}

	// Clean up output (outputStr already defined above)
	// Remove common prefixes and error traces
	lines := strings.Split(outputStr, "\n")
	cleanedLines := []string{}
	inErrorTrace := false

	for _, line := range lines {
		// Skip system messages
		if strings.Contains(line, "Loaded cached credentials") ||
			strings.Contains(line, "To authenticate") ||
			strings.HasPrefix(line, "Gemini CLI") {
			continue
		}

		// Detect start of error trace or stack trace
		if strings.Contains(line, "Attempt") && strings.Contains(line, "failed with status") {
			inErrorTrace = true
			continue
		}
		if strings.Contains(line, "GaxiosError:") || strings.Contains(line, "at Gaxios._request") {
			inErrorTrace = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "at ") || strings.HasPrefix(strings.TrimSpace(line), "at async") {
			inErrorTrace = true
			continue
		}

		// Skip lines that are part of error objects or config dumps
		if inErrorTrace {
			// Check if we've reached the end of the error trace
			// Error traces typically end with empty lines or actual content
			if strings.TrimSpace(line) == "" {
				continue // Skip empty lines within trace
			}
			// Check if this looks like actual response content (not trace)
			// Real content typically doesn't start with special chars like '{', '[', or indentation
			if !strings.HasPrefix(strings.TrimSpace(line), "{") &&
				!strings.HasPrefix(strings.TrimSpace(line), "[") &&
				!strings.HasPrefix(strings.TrimSpace(line), "}") &&
				!strings.HasPrefix(strings.TrimSpace(line), "]") &&
				!strings.Contains(line, "config:") &&
				!strings.Contains(line, "response:") &&
				!strings.Contains(line, "Symbol(") &&
				len(strings.TrimSpace(line)) > 20 { // Real content is typically longer
				// This looks like actual content, exit error trace mode
				inErrorTrace = false
			} else {
				continue // Still in error trace, skip
			}
		}

		cleanedLines = append(cleanedLines, line)
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("gemini message sent successfully")

	return strings.TrimSpace(strings.Join(cleanedLines, "\n")), nil
}

func (g *GeminiAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	// Filter out this agent's own messages
	relevantMessages := g.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := g.buildPrompt(relevantMessages, true)

	// Build command with model flag if specified
	args := []string{}
	if g.Config.Model != "" {
		args = append(args, "--model", g.Config.Model)
	}

	// Use stdin for the prompt
	cmd := exec.CommandContext(ctx, g.execPath, args...)
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

func (g *GeminiAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		// Exclude this agent's own messages
		if msg.AgentName == g.Name || msg.AgentID == g.ID {
			continue
		}
		relevant = append(relevant, msg)
	}
	return relevant
}

func (g *GeminiAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", g.Name))

	if g.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(g.Config.Prompt)
		prompt.WriteString("\n\n")
	}

	// PART 2: CONVERSATION CONTEXT
	if len(messages) > 0 {
		var initialPrompt string
		var otherMessages []agent.Message

		// Find the orchestrator's initial prompt (AgentID="system" or "host")
		// vs agent announcements (system messages from specific agents)
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", g.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("gemini", NewGeminiAgent)
}
