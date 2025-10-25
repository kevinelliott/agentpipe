package adapters

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/internal/registry"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

type GroqAgent struct {
	agent.BaseAgent
	execPath string
}

func NewGroqAgent() agent.Agent {
	return &GroqAgent{}
}

func (g *GroqAgent) Initialize(config agent.AgentConfig) error {
	if err := g.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("groq agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("groq")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   g.ID,
			"agent_name": g.Name,
		}).WithError(err).Error("groq CLI not found in PATH")
		return fmt.Errorf("groq CLI not found: %w", err)
	}
	g.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   g.ID,
		"agent_name": g.Name,
		"exec_path":  path,
		"model":      g.Config.Model,
	}).Info("groq agent initialized successfully")

	return nil
}

func (g *GroqAgent) IsAvailable() bool {
	_, err := exec.LookPath("groq")
	return err == nil
}

func (g *GroqAgent) GetCLIVersion() string {
	return registry.GetInstalledVersion("groq")
}

func (g *GroqAgent) HealthCheck(ctx context.Context) error {
	if g.execPath == "" {
		log.WithField("agent_name", g.Name).Error("groq health check failed: not initialized")
		return fmt.Errorf("groq CLI not initialized")
	}

	log.WithField("agent_name", g.Name).Debug("starting groq health check")

	// Check if the Groq CLI binary exists and responds to --version
	cmd := exec.CommandContext(ctx, g.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try with -V flag if --version doesn't work
		log.WithField("agent_name", g.Name).Debug("--version check failed, trying -V")
		cmd = exec.CommandContext(ctx, g.execPath, "-V")
		output, err = cmd.CombinedOutput()

		if err != nil {
			// If both fail, the CLI is not properly installed
			log.WithField("agent_name", g.Name).WithError(err).Error("groq health check failed: CLI not responding")
			return fmt.Errorf("groq CLI not responding to --version or -V: %w", err)
		}
	}

	// Check if output contains version information
	outputStr := string(output)
	if len(outputStr) < 3 {
		log.WithFields(map[string]interface{}{
			"agent_name":    g.Name,
			"output_length": len(outputStr),
		}).Error("groq health check failed: output too short")
		return fmt.Errorf("groq CLI returned suspiciously short output")
	}

	log.WithField("agent_name", g.Name).Info("groq health check passed")
	return nil
}

func (g *GroqAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.Name,
		"message_count": len(messages),
	}).Debug("sending message to groq CLI")

	// Filter out this agent's own messages
	relevantMessages := g.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := g.buildPrompt(relevantMessages, true)

	// Build command args
	args := []string{}

	// Add temperature flag if specified and valid
	if g.Config.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.1f", g.Config.Temperature))
	}

	// Groq CLI takes prompt via stdin
	cmd := exec.CommandContext(ctx, g.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": g.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("groq execution failed with exit code")
			return "", fmt.Errorf("groq execution failed (exit code %d): %s", exitErr.ExitCode(), string(output))
		}
		log.WithFields(map[string]interface{}{
			"agent_name": g.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("groq execution failed")
		return "", fmt.Errorf("groq execution failed: %w\nOutput: %s", err, string(output))
	}

	// Clean up output - remove system messages and login prompts
	outputStr := string(output)
	cleanedOutput := g.cleanOutput(outputStr)

	log.WithFields(map[string]interface{}{
		"agent_name":    g.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("groq message sent successfully")

	return cleanedOutput, nil
}

func (g *GroqAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.Name,
		"message_count": len(messages),
	}).Debug("starting groq streaming message")

	// Filter out this agent's own messages
	relevantMessages := g.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := g.buildPrompt(relevantMessages, true)

	// Build command with temperature flag if specified
	args := []string{}
	if g.Config.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.1f", g.Config.Temperature))
	}

	// Groq CLI takes prompt via stdin
	cmd := exec.CommandContext(ctx, g.execPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithField("agent_name", g.Name).WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", g.Name).WithError(err).Error("failed to start groq process")
		return fmt.Errorf("failed to start groq: %w", err)
	}

	startTime := time.Now()
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		// Skip system messages and authentication prompts
		if g.shouldSkipLine(line) {
			continue
		}
		fmt.Fprintln(writer, line)
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		log.WithField("agent_name", g.Name).WithError(err).Error("error reading streaming output")
		return fmt.Errorf("error reading output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		log.WithField("agent_name", g.Name).WithError(err).Error("groq streaming execution failed")
		return fmt.Errorf("groq execution failed: %w", err)
	}

	duration := time.Since(startTime)
	log.WithFields(map[string]interface{}{
		"agent_name": g.Name,
		"duration":   duration.String(),
		"lines":      lineCount,
	}).Info("groq streaming message completed")

	return nil
}

// filterRelevantMessages filters out this agent's own messages
// We exclude this agent's own messages to avoid showing Groq what it already said
func (g *GroqAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip this agent's own messages
		if msg.AgentName == g.Name || msg.AgentID == g.ID {
			continue
		}
		// Include messages from other agents and system messages
		relevant = append(relevant, msg)
	}

	return relevant
}

func (g *GroqAgent) buildPrompt(messages []agent.Message, isInitialSession bool) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE (always first)
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", g.Name))

	if g.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(g.Config.Prompt)
		prompt.WriteString("\n")
	}
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n\n")

	// PART 2: CONVERSATION CONTEXT (after role is established)
	if len(messages) > 0 {
		// Deliver ALL existing messages including initial prompt and all conversation
		var initialPrompt string
		var otherMessages []agent.Message

		// IMPORTANT: Find the orchestrator's initial prompt (AgentID/AgentName = "host" or "system")
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
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.", g.Name))
		} else {
			prompt.WriteString(fmt.Sprintf("Now, as %s, respond to the conversation.", g.Name))
		}
	}

	return prompt.String()
}

// cleanOutput removes system messages, login prompts, and other noise from Groq output
func (g *GroqAgent) cleanOutput(output string) string {
	lines := strings.Split(output, "\n")
	cleanedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		if g.shouldSkipLine(line) {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}

// shouldSkipLine determines if a line should be filtered out from output
func (g *GroqAgent) shouldSkipLine(line string) bool {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return false // Keep empty lines for formatting
	}

	// Skip authentication/login messages
	if strings.Contains(line, "To authenticate") ||
		strings.Contains(line, "/login") ||
		strings.Contains(line, "GROQ_API_KEY") {
		return true
	}

	// Skip system initialization messages
	if strings.Contains(line, "Groq CLI") ||
		strings.Contains(line, "Loaded cached") {
		return true
	}

	return false
}

func init() {
	agent.RegisterFactory("groq", NewGroqAgent)
}
