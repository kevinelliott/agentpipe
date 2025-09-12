package adapters

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

type QwenAgent struct {
	agent.BaseAgent
	execPath string
}

func NewQwenAgent() agent.Agent {
	return &QwenAgent{}
}

func (q *QwenAgent) Initialize(config agent.AgentConfig) error {
	if err := q.BaseAgent.Initialize(config); err != nil {
		return err
	}

	path, err := exec.LookPath("qwen")
	if err != nil {
		return fmt.Errorf("qwen CLI not found: %w", err)
	}
	q.execPath = path

	return nil
}

func (q *QwenAgent) IsAvailable() bool {
	_, err := exec.LookPath("qwen")
	return err == nil
}

func (q *QwenAgent) HealthCheck(ctx context.Context) error {
	if q.execPath == "" {
		return fmt.Errorf("qwen CLI not initialized")
	}

	// Test with version or help command instead of a prompt
	cmd := exec.CommandContext(ctx, q.execPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try help if version doesn't work
		cmd = exec.CommandContext(ctx, q.execPath, "--help")
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Some CLIs might not support flags, just check if we can execute it
			testCmd := exec.Command(q.execPath)
			if err := testCmd.Start(); err != nil {
				return fmt.Errorf("qwen CLI cannot be executed: %w", err)
			}
			testCmd.Process.Kill()
			// If we can start it, consider it healthy
			return nil
		}
	}

	if len(output) == 0 {
		// Empty output is OK for version/help commands
		return nil
	}

	return nil
}

func (q *QwenAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	conversation := q.formatConversation(messages)
	prompt := q.buildPrompt(conversation)

	// Qwen uses -p/--prompt for non-interactive mode
	args := []string{}
	if q.Config.Model != "" {
		args = append(args, "--model", q.Config.Model)
	}
	// Note: qwen CLI doesn't seem to support temperature/max-tokens flags based on --help output
	args = append(args, "--prompt", prompt)

	cmd := exec.CommandContext(ctx, q.execPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qwen execution failed: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func (q *QwenAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	if len(messages) == 0 {
		return nil
	}

	conversation := q.formatConversation(messages)
	prompt := q.buildPrompt(conversation)

	// Qwen uses -p/--prompt for non-interactive mode
	// Note: Streaming might not be directly supported, fallback to regular execution
	args := []string{}
	if q.Config.Model != "" {
		args = append(args, "--model", q.Config.Model)
	}
	args = append(args, "--prompt", prompt)

	cmd := exec.CommandContext(ctx, q.execPath, args...)

	// For now, just execute and write the output since qwen may not support streaming
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qwen execution failed: %w", err)
	}

	fmt.Fprintln(writer, strings.TrimSpace(string(output)))
	return nil
}

func (q *QwenAgent) formatConversation(messages []agent.Message) string {
	var parts []string

	for _, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", timestamp, msg.AgentName, msg.Content))
	}

	return strings.Join(parts, "\n")
}

func (q *QwenAgent) buildPrompt(conversation string) string {
	var prompt strings.Builder

	if q.Config.Prompt != "" {
		prompt.WriteString(q.Config.Prompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("You are participating in a multi-agent conversation. ")
	prompt.WriteString(fmt.Sprintf("Your name is '%s'. ", q.Name))
	prompt.WriteString("Here is the conversation so far:\n\n")
	prompt.WriteString(conversation)
	prompt.WriteString("\n\nPlease provide your response:")

	return prompt.String()
}

func init() {
	agent.RegisterFactory("qwen", NewQwenAgent)
}

