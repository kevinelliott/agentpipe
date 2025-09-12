package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

type AgentCheck struct {
	Name      string
	Command   string
	Available bool
	Path      string
	Error     error
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check if AI agent CLIs are installed and available",
	Long:  `Doctor command checks your system for installed AI agent CLIs and reports their availability.`,
	Run:   runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 AgentPipe Doctor - Checking installed AI agents...")
	fmt.Println("=" + string(make([]byte, 50)))

	agents := []struct {
		name    string
		command string
		notes   string
	}{
		{"Claude", "claude", "Claude Code CLI (https://github.com/anthropics/claude-code)"},
		{"Gemini", "gemini", "Gemini CLI (https://github.com/google/generative-ai-cli)"},
		{"Qwen", "qwen", "Qwen Code CLI (https://github.com/QwenLM/qwen-code)"},
		{"Codex", "codex", "Codex CLI (https://github.com/openai/codex-cli)"},
		{"Ollama", "ollama", "Ollama CLI (https://github.com/ollama/ollama)"},
	}

	availableCount := 0

	for _, agent := range agents {
		check := checkAgent(agent.command)

		statusIcon := "❌"
		statusText := "Not installed"

		if check.Available {
			statusIcon = "✅"
			statusText = fmt.Sprintf("Available at %s", check.Path)
			availableCount++
		} else if check.Error != nil && check.Error != exec.ErrNotFound {
			statusIcon = "⚠️"
			statusText = fmt.Sprintf("Error: %v", check.Error)
		}

		fmt.Printf("\n%s %s:\n", statusIcon, agent.name)
		fmt.Printf("   Command: %s\n", agent.command)
		fmt.Printf("   Status:  %s\n", statusText)
		if agent.notes != "" {
			fmt.Printf("   Notes:   %s\n", agent.notes)
		}
	}

	fmt.Println("\n" + string(make([]byte, 50)) + "=")
	fmt.Printf("\n📊 Summary: %d/%d agents available\n", availableCount, len(agents))

	if availableCount == 0 {
		fmt.Println("\n⚠️  No AI agents found. Please install at least one agent CLI to use AgentPipe.")
		fmt.Println("   Visit the respective documentation pages to install the agents.")
	} else {
		fmt.Printf("\n✨ You can use AgentPipe with the %d available agent(s).\n", availableCount)
		fmt.Println("   Run 'agentpipe run --help' to start a conversation.")
	}
}

func checkAgent(command string) AgentCheck {
	check := AgentCheck{
		Name:    command,
		Command: command,
	}

	path, err := exec.LookPath(command)
	if err != nil {
		check.Error = err
		if err == exec.ErrNotFound {
			check.Available = false
		}
		return check
	}

	check.Available = true
	check.Path = path

	testCmd := exec.Command(command, "--version")
	if err := testCmd.Run(); err != nil {
		testCmd = exec.Command(command, "version")
		if err := testCmd.Run(); err != nil {
			check.Error = fmt.Errorf("installed but may not be properly configured")
		}
	}

	return check
}

