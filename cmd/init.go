package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new AgentPipe configuration",
	Long: `Create a new AgentPipe configuration file interactively.
This command will guide you through setting up agents and orchestration options.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("output", "o", "agentpipe.yaml", "Output configuration file path")
}

func runInit(cmd *cobra.Command, args []string) error {
	outputPath, _ := cmd.Flags().GetString("output")

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║          AgentPipe Configuration Setup           ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Println()

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("⚠️  Configuration file '%s' already exists.\n", outputPath)
		if !promptYesNo(reader, "Overwrite?", false) {
			fmt.Println("❌ Cancelled.")
			return nil
		}
		fmt.Println()
	}

	cfg := &config.Config{
		Version: "1.0",
		Agents:  []agent.AgentConfig{},
	}

	// Configure agents
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Agent Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	availableAgents := []string{"claude", "gemini", "copilot", "cursor", "qwen", "codex"}
	fmt.Println("Available agent types:")
	for i, agentType := range availableAgents {
		fmt.Printf("  %d. %s\n", i+1, agentType)
	}
	fmt.Println()

	for {
		fmt.Printf("Select agents to configure (e.g., 1,2,4 or 'all'): ")
		selection, _ := reader.ReadString('\n')
		selection = strings.TrimSpace(selection)

		if selection == "" {
			fmt.Println("❌ Please select at least one agent.")
			continue
		}

		selectedAgents := []string{}
		if selection == "all" {
			selectedAgents = availableAgents
		} else {
			indices := strings.Split(selection, ",")
			for _, idx := range indices {
				idx = strings.TrimSpace(idx)
				i, err := strconv.Atoi(idx)
				if err != nil || i < 1 || i > len(availableAgents) {
					fmt.Printf("❌ Invalid selection: %s\n", idx)
					continue
				}
				selectedAgents = append(selectedAgents, availableAgents[i-1])
			}
		}

		if len(selectedAgents) == 0 {
			fmt.Println("❌ Please select at least one agent.")
			continue
		}

		// Configure each selected agent
		for _, agentType := range selectedAgents {
			fmt.Println()
			fmt.Printf("Configuring %s agent:\n", agentType)
			fmt.Println("  " + strings.Repeat("─", 40))

			agentCfg := agent.AgentConfig{
				Type: agentType,
			}

			// Agent ID
			agentCfg.ID = promptString(reader, fmt.Sprintf("  Agent ID (default: %s-1)", agentType), fmt.Sprintf("%s-1", agentType))

			// Agent Name
			defaultName := strings.ToUpper(agentType[:1]) + agentType[1:]
			agentCfg.Name = promptString(reader, fmt.Sprintf("  Agent Name (default: %s)", defaultName), defaultName)

			// System Prompt
			defaultPrompt := fmt.Sprintf("You are a helpful AI assistant powered by %s.", defaultName)
			agentCfg.Prompt = promptString(reader, "  System Prompt (default: helpful assistant)", defaultPrompt)

			// Announcement
			agentCfg.Announcement = promptString(reader, "  Announcement (optional)", "")

			// Model (optional)
			agentCfg.Model = promptString(reader, "  Model (optional, e.g., claude-sonnet-4.5)", "")

			cfg.Agents = append(cfg.Agents, agentCfg)
			fmt.Printf("  ✅ Added %s\n", agentCfg.Name)
		}

		break
	}

	// Configure orchestrator
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Orchestrator Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Println("Conversation modes:")
	fmt.Println("  1. round-robin - Agents take turns in order")
	fmt.Println("  2. reactive    - Random agent selection (no repeats)")
	fmt.Println("  3. free-form   - All agents can participate freely")
	fmt.Println()

	modeChoice := promptChoice(reader, "Select mode", []string{"round-robin", "reactive", "free-form"}, 1)
	cfg.Orchestrator.Mode = modeChoice

	cfg.Orchestrator.MaxTurns = promptInt(reader, "Maximum turns (0 for unlimited)", 10)
	cfg.Orchestrator.TurnTimeout = time.Duration(promptInt(reader, "Turn timeout (seconds)", 30)) * time.Second
	cfg.Orchestrator.ResponseDelay = time.Duration(promptInt(reader, "Delay between responses (seconds)", 2)) * time.Second

	if promptYesNo(reader, "Add an initial prompt to start the conversation?", false) {
		cfg.Orchestrator.InitialPrompt = promptString(reader, "Initial prompt", "")
	}

	// Configure logging
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Logging Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	cfg.Logging.Enabled = promptYesNo(reader, "Enable conversation logging?", true)

	if cfg.Logging.Enabled {
		homeDir, _ := os.UserHomeDir()
		defaultLogDir := filepath.Join(homeDir, ".agentpipe", "chats")
		cfg.Logging.ChatLogDir = promptString(reader, fmt.Sprintf("Log directory (default: %s)", defaultLogDir), defaultLogDir)

		formatChoice := promptChoice(reader, "Log format", []string{"text", "json"}, 1)
		cfg.Logging.LogFormat = formatChoice

		cfg.Logging.ShowMetrics = promptYesNo(reader, "Show token/cost metrics in logs?", false)
	}

	// Apply defaults
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save configuration
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Saving Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Configuration saved to: %s\n", outputPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review and edit the configuration if needed")
	fmt.Println("  2. Run: agentpipe doctor")
	fmt.Println("  3. Start conversation: agentpipe run -c " + outputPath)
	fmt.Println()

	return nil
}

func promptString(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s: ", prompt)
	} else {
		fmt.Printf("%s (leave empty to skip): ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func promptInt(reader *bufio.Reader, prompt string, defaultValue int) int {
	for {
		fmt.Printf("%s (default: %d): ", prompt, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultValue
		}

		value, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("  ❌ Invalid number. Please try again.\n")
			continue
		}
		return value
	}
}

func promptYesNo(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	for {
		fmt.Printf("%s [%s]: ", prompt, defaultStr)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			return defaultValue
		}

		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" {
			return false
		}

		fmt.Println("  ❌ Please answer 'y' or 'n'")
	}
}

func promptChoice(reader *bufio.Reader, prompt string, choices []string, defaultIndex int) string {
	for {
		fmt.Printf("%s (1-%d, default: %d): ", prompt, len(choices), defaultIndex)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return choices[defaultIndex-1]
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(choices) {
			fmt.Printf("  ❌ Please select a number between 1 and %d\n", len(choices))
			continue
		}

		return choices[choice-1]
	}
}
