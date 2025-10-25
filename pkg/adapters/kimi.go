package adapters

import (
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

type KimiAgent struct {
	agent.BaseAgent
	execPath string
}

func NewKimiAgent() agent.Agent {
	return &KimiAgent{}
}

func (k *KimiAgent) Initialize(config agent.AgentConfig) error {
	if err := k.BaseAgent.Initialize(config); err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   config.ID,
			"agent_name": config.Name,
		}).WithError(err).Error("kimi agent base initialization failed")
		return err
	}

	path, err := exec.LookPath("kimi")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   k.ID,
			"agent_name": k.Name,
		}).WithError(err).Error("kimi not found in PATH")
		return fmt.Errorf("kimi not found: %w", err)
	}
	k.execPath = path

	log.WithFields(map[string]interface{}{
		"agent_id":   k.ID,
		"agent_name": k.Name,
		"exec_path":  path,
		"model":      k.Config.Model,
	}).Info("kimi agent initialized successfully")

	return nil
}

func (k *KimiAgent) IsAvailable() bool {
	_, err := exec.LookPath("kimi")
	return err == nil
}

func (k *KimiAgent) GetCLIVersion() string {
	return registry.GetInstalledVersion("kimi")
}

func (k *KimiAgent) HealthCheck(ctx context.Context) error {
	if k.execPath == "" {
		log.WithField("agent_name", k.Name).Error("kimi health check failed: not initialized")
		return fmt.Errorf("kimi not initialized")
	}

	log.WithField("agent_name", k.Name).Debug("starting kimi health check")

	// Test with help flag
	cmd := exec.CommandContext(ctx, k.execPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.WithField("agent_name", k.Name).WithError(err).Error("kimi health check failed: CLI not responding")
		return fmt.Errorf("kimi health check failed: %w", err)
	}

	// Check if output looks like kimi help
	outputStr := string(output)
	if len(outputStr) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    k.Name,
			"output_length": len(outputStr),
		}).Error("kimi health check failed: output too short")
		return fmt.Errorf("kimi returned suspiciously short output")
	}

	// Verify it's actually Kimi by checking for key terms
	if !strings.Contains(strings.ToLower(outputStr), "kimi") && !strings.Contains(strings.ToLower(outputStr), "moonshot") {
		log.WithField("agent_name", k.Name).Error("kimi health check failed: output doesn't appear to be from kimi")
		return fmt.Errorf("CLI at path doesn't appear to be kimi")
	}

	log.WithField("agent_name", k.Name).Info("kimi health check passed")
	return nil
}

func (k *KimiAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    k.Name,
		"message_count": len(messages),
	}).Debug("sending message to kimi")

	// Filter out this agent's own messages
	relevantMessages := k.filterRelevantMessages(messages)

	// Build prompt with structured format
	prompt := k.buildPrompt(relevantMessages)

	// Kimi is an interactive tool without a traditional non-interactive API mode
	// However, we can attempt to send input via stdin with the --help flag replaced by interactive mode
	// For now, return a helpful message directing users to use kimi interactively
	log.WithFields(map[string]interface{}{
		"agent_name": k.Name,
	}).Warn("kimi is an interactive-first CLI tool; consider running it directly for best experience")

	// Try to use Kimi with stdin (experimental)
	// Kimi doesn't have a documented non-interactive mode, so this is a best-effort attempt
	cmd := exec.CommandContext(ctx, k.execPath)
	cmd.Stdin = strings.NewReader(prompt)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		// Kimi requires interactive setup and authentication
		outputStr := string(output)
		if strings.Contains(strings.ToLower(outputStr), "not logged in") || strings.Contains(strings.ToLower(outputStr), "authentication") {
			log.WithFields(map[string]interface{}{
				"agent_name": k.Name,
				"duration":   duration.String(),
			}).WithError(err).Error("kimi authentication failed")
			return "", fmt.Errorf("kimi not authenticated - please run 'kimi' and use '.set_api_key' command to authenticate with Moonshot AI")
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.WithFields(map[string]interface{}{
				"agent_name": k.Name,
				"exit_code":  exitErr.ExitCode(),
				"duration":   duration.String(),
			}).WithError(err).Error("kimi execution failed with exit code")
			return "", fmt.Errorf("kimi execution failed (exit code %d): kimi is an interactive tool and may not work in non-interactive mode", exitErr.ExitCode())
		}

		log.WithFields(map[string]interface{}{
			"agent_name": k.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("kimi execution failed")
		return "", fmt.Errorf("kimi execution failed: %w - kimi is designed as an interactive CLI tool", err)
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    k.Name,
		"duration":      duration.String(),
		"response_size": len(output),
	}).Info("kimi message sent successfully (interactive mode)")

	return strings.TrimSpace(string(output)), nil
}

func (k *KimiAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    k.Name,
		"message_count": len(messages),
	}).Debug("starting kimi streaming message")

	// Kimi is interactive-first, streaming support would require special setup
	// For now, use SendMessage and write output to the writer
	response, err := k.SendMessage(ctx, messages)
	if err != nil {
		return err
	}

	if _, err := io.WriteString(writer, response); err != nil {
		log.WithField("agent_name", k.Name).WithError(err).Error("failed to write kimi response to stream")
		return fmt.Errorf("failed to write response to stream: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    k.Name,
		"response_size": len(response),
	}).Info("kimi streaming message completed")

	return nil
}

func (k *KimiAgent) filterRelevantMessages(messages []agent.Message) []agent.Message {
	relevant := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		// Exclude this agent's own messages
		if msg.AgentName == k.Name || msg.AgentID == k.ID {
			continue
		}
		relevant = append(relevant, msg)
	}
	return relevant
}

func (k *KimiAgent) buildPrompt(messages []agent.Message) string {
	var prompt strings.Builder

	// PART 1: IDENTITY AND ROLE
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", k.Name))

	if k.Config.Prompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(k.Config.Prompt)
		prompt.WriteString("\n\n")
	}

	// PART 2: CONVERSATION CONTEXT
	if len(messages) > 0 {
		var initialPrompt string
		var otherMessages []agent.Message

		// Find orchestrator's initial prompt (HOST) vs agent announcements (SYSTEM)
		for _, msg := range messages {
			if msg.Role == "system" && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
				initialPrompt = msg.Content
			} else {
				otherMessages = append(otherMessages, msg)
			}
		}

		// Show initial task prominently
		if initialPrompt != "" {
			prompt.WriteString("YOUR TASK - PLEASE RESPOND TO THIS:\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n")
			prompt.WriteString(initialPrompt)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n\n")
		}

		// Show conversation history
		if len(otherMessages) > 0 {
			prompt.WriteString("CONVERSATION SO FAR:\n")
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n")
			for _, msg := range otherMessages {
				if msg.Role == "system" {
					prompt.WriteString(fmt.Sprintf("SYSTEM: %s\n", msg.Content))
				} else {
					prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.AgentName, msg.Content))
				}
			}
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n\n")
		}

		if initialPrompt != "" {
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.\n", k.Name))
		}
	}

	return prompt.String()
}

func init() {
	agent.RegisterFactory("kimi", NewKimiAgent)
}
