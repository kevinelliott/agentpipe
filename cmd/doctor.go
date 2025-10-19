package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

type AgentCheck struct {
	Name          string `json:"name"`
	Command       string `json:"command"`
	Available     bool   `json:"available"`
	Path          string `json:"path,omitempty"`
	Version       string `json:"version,omitempty"`
	Error         error  `json:"-"`
	ErrorMessage  string `json:"error,omitempty"`
	InstallCmd    string `json:"install_cmd,omitempty"`
	UpgradeCmd    string `json:"upgrade_cmd,omitempty"`
	Docs          string `json:"docs,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

type SystemCheck struct {
	Name    string `json:"name"`
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Icon    string `json:"icon,omitempty"`
}

type DoctorOutput struct {
	SystemEnvironment []SystemCheck `json:"system_environment"`
	SupportedAgents   []AgentCheck  `json:"supported_agents"`
	AvailableAgents   []AgentCheck  `json:"available_agents"`
	Configuration     []SystemCheck `json:"configuration"`
	Summary           DoctorSummary `json:"summary"`
}

type DoctorSummary struct {
	TotalAgents    int      `json:"total_agents"`
	AvailableCount int      `json:"available_count"`
	MissingAgents  []string `json:"missing_agents,omitempty"`
	Ready          bool     `json:"ready"`
}

var (
	doctorJSON bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check if AI agent CLIs are installed and available",
	Long:  `Doctor command checks your system for installed AI agent CLIs, versions, and configuration.`,
	Run:   runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results in JSON format")
}

func runDoctor(cmd *cobra.Command, args []string) {
	// Define all supported agents
	agents := []struct {
		name       string
		command    string
		installCmd string
		upgradeCmd string
		docs       string
	}{
		{"Amp", "amp", "See https://ampcode.com/install", "See https://ampcode.com/install for upgrade instructions", "https://ampcode.com"},
		{"Claude", "claude", "See https://docs.claude.com/en/docs/claude-code/installation", "See https://docs.claude.com/en/docs/claude-code/installation for upgrade instructions", "https://github.com/anthropics/claude-code"},
		{"Codex", "codex", "npm install -g @openai/codex-cli", "npm update -g @openai/codex-cli", "https://github.com/openai/codex-cli"},
		{"Copilot", "copilot", "npm install -g @github/copilot", "npm update -g @github/copilot", "https://github.com/github/copilot-cli"},
		{"Cursor", "cursor-agent", "curl https://cursor.com/install -fsS | bash", "curl https://cursor.com/install -fsS | bash", "https://cursor.com/cli"},
		{"Factory", "droid", "curl -fsSL https://app.factory.ai/cli | sh", "See https://docs.factory.ai/cli for upgrade instructions", "https://docs.factory.ai/cli"},
		{"Gemini", "gemini", "npm install -g @google/generative-ai-cli", "npm update -g @google/generative-ai-cli", "https://github.com/google/generative-ai-cli"},
		{"Qoder", "qodercli", "See https://qoder.com/cli", "See https://qoder.com/cli for upgrade instructions", "https://qoder.com/cli"},
		{"Qwen", "qwen", "See https://github.com/QwenLM/qwen-code", "See https://github.com/QwenLM/qwen-code for upgrade instructions", "https://github.com/QwenLM/qwen-code"},
		{"Ollama", "ollama", "See https://ollama.com/download", "See https://ollama.com/download for upgrade instructions", "https://ollama.com"},
	}

	// Perform system checks
	systemChecks := performSystemChecks()

	// Check all agents
	supportedAgents := make([]AgentCheck, 0, len(agents))
	availableAgents := make([]AgentCheck, 0, len(agents))
	unavailableAgents := make([]string, 0, len(agents))

	for _, agent := range agents {
		check := checkAgent(agent.command, agent.installCmd)
		check.Name = agent.name
		check.UpgradeCmd = agent.upgradeCmd
		check.Docs = agent.docs

		if check.Error != nil {
			check.ErrorMessage = check.Error.Error()
		}

		supportedAgents = append(supportedAgents, check)

		if check.Available {
			availableAgents = append(availableAgents, check)
		} else {
			unavailableAgents = append(unavailableAgents, agent.name)
		}
	}

	// Configuration checks
	configChecks := performConfigChecks()

	// Build summary
	summary := DoctorSummary{
		TotalAgents:    len(agents),
		AvailableCount: len(availableAgents),
		MissingAgents:  unavailableAgents,
		Ready:          len(availableAgents) > 0,
	}

	// Build complete output
	output := DoctorOutput{
		SystemEnvironment: systemChecks,
		SupportedAgents:   supportedAgents,
		AvailableAgents:   availableAgents,
		Configuration:     configChecks,
		Summary:           summary,
	}

	// Output in requested format
	if doctorJSON {
		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	} else {
		printHumanReadableOutput(output, agents)
	}
}

func printHumanReadableOutput(output DoctorOutput, agents []struct {
	name       string
	command    string
	installCmd string
	upgradeCmd string
	docs       string
}) {
	fmt.Println("\n🔍 AgentPipe Doctor - System Health Check")
	fmt.Println(strings.Repeat("=", 61))

	// System environment checks
	fmt.Println("\n📋 SYSTEM ENVIRONMENT")
	fmt.Println(strings.Repeat("-", 61))
	for _, check := range output.SystemEnvironment {
		fmt.Printf("  %s %s: %s\n", check.Icon, check.Name, check.Message)
	}
	fmt.Println()

	// Agent checks
	fmt.Println("\n🤖 AI AGENT CLIS")
	fmt.Println(strings.Repeat("-", 61))

	for i, check := range output.SupportedAgents {
		statusIcon := "❌"
		if check.Available {
			statusIcon = "✅"
		}

		// Add spacing between agents (but not before the first one)
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("\n  %s %s\n", statusIcon, check.Name)
		fmt.Printf("     Command:  %s\n", check.Command)

		if check.Available {
			fmt.Printf("     Path:     %s\n", check.Path)
			if check.Version != "" {
				fmt.Printf("     Version:  %s\n", check.Version)
			}
			if check.UpgradeCmd != "" {
				fmt.Printf("     Upgrade:  %s\n", check.UpgradeCmd)
			}
			// Check authentication where applicable
			if check.Authenticated {
				fmt.Printf("     Auth:     ✅ Authenticated\n")
			} else if check.Name == "Claude" || check.Name == "Cursor" || check.Name == "Qoder" || check.Name == "Factory" {
				fmt.Printf("     Auth:     ⚠️  Not authenticated (run '%s' and authenticate)\n", check.Command)
			}
		} else {
			fmt.Printf("     Status:   Not installed\n")
			if check.InstallCmd != "" {
				fmt.Printf("     Install:  %s\n", check.InstallCmd)
			}
		}
		fmt.Printf("     Docs:     %s\n", check.Docs)
	}
	fmt.Println()

	// Configuration checks
	fmt.Println("\n⚙️  CONFIGURATION")
	fmt.Println(strings.Repeat("-", 61))
	for _, check := range output.Configuration {
		fmt.Printf("  %s %s: %s\n", check.Icon, check.Name, check.Message)
	}
	fmt.Println()

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 61))
	fmt.Printf("\n📊 SUMMARY\n")
	fmt.Printf("   Available Agents: %d/%d\n", output.Summary.AvailableCount, output.Summary.TotalAgents)

	if len(output.Summary.MissingAgents) > 0 {
		fmt.Printf("   Missing Agents:   %s\n", strings.Join(output.Summary.MissingAgents, ", "))
	}

	if output.Summary.AvailableCount == 0 {
		fmt.Println()
		fmt.Println("⚠️  No AI agents found. Please install at least one agent CLI to use AgentPipe.")
		fmt.Println("   Visit the respective documentation pages above for installation instructions.")
	} else {
		fmt.Println()
		fmt.Printf("✨ AgentPipe is ready! You can use %d agent(s).\n", output.Summary.AvailableCount)
		fmt.Println("   Run 'agentpipe run --help' to start a conversation.")
	}

	fmt.Println()
}

func performSystemChecks() []SystemCheck {
	checks := []SystemCheck{}

	// Go version check
	goVersion := runtime.Version()
	checks = append(checks, SystemCheck{
		Name:    "Go Runtime",
		Status:  true,
		Message: fmt.Sprintf("%s (%s/%s)", goVersion, runtime.GOOS, runtime.GOARCH),
		Icon:    "✅",
	})

	// Check PATH
	pathEnv := os.Getenv("PATH")
	pathCount := len(strings.Split(pathEnv, string(os.PathListSeparator)))
	checks = append(checks, SystemCheck{
		Name:    "PATH",
		Status:  pathCount > 0,
		Message: fmt.Sprintf("%d directories in PATH", pathCount),
		Icon:    "✅",
	})

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		checks = append(checks, SystemCheck{
			Name:    "Home Directory",
			Status:  true,
			Message: homeDir,
			Icon:    "✅",
		})
	}

	// Check agentpipe directories
	agentpipeDir := filepath.Join(homeDir, ".agentpipe")
	chatsDir := filepath.Join(agentpipeDir, "chats")
	statesDir := filepath.Join(agentpipeDir, "states")

	if _, err := os.Stat(chatsDir); err == nil {
		checks = append(checks, SystemCheck{
			Name:    "Chat Logs Directory",
			Status:  true,
			Message: chatsDir,
			Icon:    "✅",
		})
	} else {
		checks = append(checks, SystemCheck{
			Name:    "Chat Logs Directory",
			Status:  false,
			Message: "Will be created on first use",
			Icon:    "ℹ️",
		})
	}

	if _, err := os.Stat(statesDir); err == nil {
		checks = append(checks, SystemCheck{
			Name:    "States Directory",
			Status:  true,
			Message: statesDir,
			Icon:    "✅",
		})
	}

	return checks
}

func performConfigChecks() []SystemCheck {
	checks := []SystemCheck{}

	homeDir, _ := os.UserHomeDir()

	// Check for example configs
	exampleConfigPaths := []string{
		"examples/simple-conversation.yaml",
		"examples/brainstorm.yaml",
	}

	foundExamples := 0
	for _, path := range exampleConfigPaths {
		if _, err := os.Stat(path); err == nil {
			foundExamples++
		}
	}

	if foundExamples > 0 {
		checks = append(checks, SystemCheck{
			Name:    "Example Configs",
			Status:  true,
			Message: fmt.Sprintf("%d example configurations found", foundExamples),
			Icon:    "✅",
		})
	} else {
		checks = append(checks, SystemCheck{
			Name:    "Example Configs",
			Status:  false,
			Message: "No example configs found (expected in ./examples/)",
			Icon:    "ℹ️",
		})
	}

	// Check for user config
	configPath := filepath.Join(homeDir, ".agentpipe", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		checks = append(checks, SystemCheck{
			Name:    "User Config",
			Status:  true,
			Message: configPath,
			Icon:    "✅",
		})
	} else {
		checks = append(checks, SystemCheck{
			Name:    "User Config",
			Status:  false,
			Message: "No user config (use 'agentpipe init' to create one)",
			Icon:    "ℹ️",
		})
	}

	return checks
}

func checkAgent(command string, installCmd string) AgentCheck {
	check := AgentCheck{
		Name:       command,
		Command:    command,
		InstallCmd: installCmd,
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

	// Try to get version
	versionCmd := exec.Command(command, "--version")
	if output, err := versionCmd.CombinedOutput(); err == nil {
		version := strings.TrimSpace(string(output))
		// Clean up version output (take first line if multi-line)
		if lines := strings.Split(version, "\n"); len(lines) > 0 {
			check.Version = strings.TrimSpace(lines[0])
			// Limit version string length
			if len(check.Version) > 60 {
				check.Version = check.Version[:60] + "..."
			}
		}
	} else {
		// Try alternative version commands
		versionCmd = exec.Command(command, "version")
		if output, err := versionCmd.CombinedOutput(); err == nil {
			version := strings.TrimSpace(string(output))
			if lines := strings.Split(version, "\n"); len(lines) > 0 {
				check.Version = strings.TrimSpace(lines[0])
				if len(check.Version) > 60 {
					check.Version = check.Version[:60] + "..."
				}
			}
		}
	}

	// Check authentication status for specific agents
	check.Authenticated = checkAuthentication(command)

	return check
}

func checkAuthentication(command string) bool {
	switch command {
	case "claude":
		// Try a simple command that requires auth
		cmd := exec.Command(command, "--help")
		return cmd.Run() == nil
	case "cursor-agent":
		// Check status command
		cmd := exec.Command(command, "status")
		output, _ := cmd.CombinedOutput()
		return !strings.Contains(strings.ToLower(string(output)), "not logged in")
	case "qodercli":
		// Qoder might need specific auth check
		cmd := exec.Command(command, "--help")
		return cmd.Run() == nil
	case "droid":
		// Factory CLI requires authentication
		cmd := exec.Command(command, "--help")
		return cmd.Run() == nil
	default:
		// Default: assume authenticated if command exists
		return true
	}
}
