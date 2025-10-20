package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kevinelliott/agentpipe/internal/registry"
)

var (
	installAll      bool
	listInstalled   bool
	listOutdated    bool
)

// agentsCmd represents the agents command
var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage AI agent CLIs",
	Long: `Manage AI agent CLIs including listing, installing, and getting information about supported agents.

Examples:
  agentpipe agents list              # List all supported agents
  agentpipe agents install claude    # Install Claude CLI
  agentpipe agents install --all     # Install all agents`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// agentsListCmd lists all supported agents
var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all supported AI agent CLIs",
	Long: `List all supported AI agent CLIs with their installation status, description, and documentation links.

Examples:
  agentpipe agents list              # List all agents
  agentpipe agents list --installed  # List only installed agents
  agentpipe agents list --outdated   # List only outdated agents`,
	Run: runAgentsList,
}

// agentsInstallCmd installs one or more agents
var agentsInstallCmd = &cobra.Command{
	Use:   "install [agent...]",
	Short: "Install AI agent CLIs",
	Long: `Install one or more AI agent CLIs. Use --all to install all supported agents.

Examples:
  agentpipe agents install claude         # Install Claude CLI
  agentpipe agents install claude ollama  # Install multiple agents
  agentpipe agents install --all          # Install all agents`,
	Run: runAgentsInstall,
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsInstallCmd)

	agentsListCmd.Flags().BoolVar(&listInstalled, "installed", false, "List only installed agents")
	agentsListCmd.Flags().BoolVar(&listOutdated, "outdated", false, "List only outdated agents")
	agentsInstallCmd.Flags().BoolVar(&installAll, "all", false, "Install all agents")
}

func runAgentsList(cmd *cobra.Command, args []string) {
	agents := registry.GetAll()

	// Filter agents based on flags
	var filteredAgents []*registry.AgentDefinition
	for _, agent := range agents {
		installed := isAgentInstalled(agent.Command)

		// Apply filters
		if listInstalled && !installed {
			continue
		}

		if listOutdated {
			if !installed {
				continue
			}
			// Check if agent has updates available
			hasUpdate, _, _ := checkForAgentUpdate(agent)
			if !hasUpdate {
				continue
			}
		}

		filteredAgents = append(filteredAgents, agent)
	}

	// Sort agents by name
	sort.Slice(filteredAgents, func(i, j int) bool {
		return filteredAgents[i].Name < filteredAgents[j].Name
	})

	// Determine title based on flags
	title := "AI Agent CLIs"
	if listInstalled {
		title = "Installed AI Agent CLIs"
	} else if listOutdated {
		title = "Outdated AI Agent CLIs"
	}

	fmt.Printf("\n%s\n", title)
	fmt.Println(strings.Repeat("=", 70))

	if len(filteredAgents) == 0 {
		fmt.Println("\nNo agents found matching the specified criteria.")
		if listOutdated {
			fmt.Println("All installed agents are up to date!")
		}
		fmt.Println()
		return
	}

	for i, agent := range filteredAgents {
		// Add spacing between agents
		if i > 0 {
			fmt.Println()
		}

		// Check if agent is installed
		installed := isAgentInstalled(agent.Command)
		statusIcon := "✅"
		if !installed {
			statusIcon = "❌"
		}

		fmt.Printf("\n%s %s (%s)\n", statusIcon, agent.Name, agent.Command)
		fmt.Printf("   %s\n", agent.Description)

		if installed {
			// Show path if installed
			if path, err := exec.LookPath(agent.Command); err == nil {
				fmt.Printf("   Installed: %s\n", path)
			}

			// Show current version if available
			version := getAgentVersion(agent.Command)
			if version != "" {
				fmt.Printf("   Version: %s\n", version)
			}

			// Check for updates
			if hasUpdate, latestVersion, _ := checkForAgentUpdate(agent); hasUpdate {
				fmt.Printf("   ⚠️  Update available: %s\n", latestVersion)
				if upgradeCmd, err := agent.GetUpgradeCommand(); err == nil {
					fmt.Printf("   Upgrade: %s\n", upgradeCmd)
				}
			}
		} else {
			// Show install command or instructions
			installCmd, err := agent.GetInstallCommand()
			if err == nil {
				if agent.IsInstallable() {
					fmt.Printf("   Install: agentpipe agents install %s\n", strings.ToLower(agent.Name))
				} else {
					fmt.Printf("   Install: %s\n", installCmd)
				}
			}
		}

		fmt.Printf("   Docs: %s\n", agent.Docs)
	}

	fmt.Println()
}

func runAgentsInstall(cmd *cobra.Command, args []string) {
	var agentsToInstall []*registry.AgentDefinition

	if installAll {
		// Install all agents
		agentsToInstall = registry.GetAll()
		fmt.Println("\nInstalling all agents...")
	} else if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Please specify at least one agent to install, or use --all\n")
		fmt.Fprintf(os.Stderr, "Usage: agentpipe agents install [agent...]\n")
		fmt.Fprintf(os.Stderr, "       agentpipe agents install --all\n\n")
		fmt.Fprintf(os.Stderr, "Run 'agentpipe agents list' to see available agents\n")
		os.Exit(1)
		return
	} else {
		// Install specific agents
		for _, name := range args {
			agent, err := registry.GetByName(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Agent '%s' not found in registry\n", name)
				fmt.Fprintf(os.Stderr, "Run 'agentpipe agents list' to see available agents\n")
				os.Exit(1)
				return
			}
			agentsToInstall = append(agentsToInstall, agent)
		}
	}

	// Track installation results
	successCount := 0
	skipCount := 0
	failCount := 0

	fmt.Println()

	for _, agent := range agentsToInstall {
		// Check if already installed
		if isAgentInstalled(agent.Command) {
			fmt.Printf("⏭️  %s is already installed (skipping)\n", agent.Name)
			skipCount++
			continue
		}

		// Get install command
		installCmd, err := agent.GetInstallCommand()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %s: %v\n", agent.Name, err)
			failCount++
			continue
		}

		// Check if installable via command
		if !agent.IsInstallable() {
			fmt.Printf("ℹ️  %s: %s\n", agent.Name, installCmd)
			skipCount++
			continue
		}

		// Execute installation
		fmt.Printf("📦 Installing %s...\n", agent.Name)
		fmt.Printf("   Running: %s\n", installCmd)

		if err := executeInstallCommand(installCmd); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to install %s: %v\n", agent.Name, err)
			failCount++
			continue
		}

		// Verify installation
		if isAgentInstalled(agent.Command) {
			fmt.Printf("✅ Successfully installed %s\n", agent.Name)
			fmt.Printf("   Run '%s --help' to get started\n", agent.Command)
			successCount++
		} else {
			fmt.Fprintf(os.Stderr, "⚠️  %s installation completed but command not found in PATH\n", agent.Name)
			fmt.Fprintf(os.Stderr, "   You may need to restart your shell or add the installation directory to PATH\n")
			failCount++
		}

		fmt.Println()
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("\nInstallation Summary:\n")
	fmt.Printf("  ✅ Installed: %d\n", successCount)
	if skipCount > 0 {
		fmt.Printf("  ⏭️  Skipped:   %d\n", skipCount)
	}
	if failCount > 0 {
		fmt.Printf("  ❌ Failed:    %d\n", failCount)
	}
	fmt.Println()

	if failCount > 0 {
		os.Exit(1)
	}
}

// getAgentVersion gets the version of an installed agent
func getAgentVersion(command string) string {
	// Try --version first
	cmd := exec.Command(command, "--version")
	if output, err := cmd.CombinedOutput(); err == nil {
		version := strings.TrimSpace(string(output))
		// Take first line if multiline
		if lines := strings.Split(version, "\n"); len(lines) > 0 {
			version = strings.TrimSpace(lines[0])
		}
		// Limit length
		if len(version) > 60 {
			version = version[:60] + "..."
		}
		return version
	}

	// Try version subcommand
	cmd = exec.Command(command, "version")
	if output, err := cmd.CombinedOutput(); err == nil {
		version := strings.TrimSpace(string(output))
		if lines := strings.Split(version, "\n"); len(lines) > 0 {
			version = strings.TrimSpace(lines[0])
		}
		if len(version) > 60 {
			version = version[:60] + "..."
		}
		return version
	}

	return ""
}

// checkForAgentUpdate checks if an agent has an update available
// Returns: hasUpdate, latestVersion, error
func checkForAgentUpdate(agent *registry.AgentDefinition) (bool, string, error) {
	// For now, we'll return false as implementing proper version checking
	// requires npm/homebrew/package manager integration which is complex.
	// This is a placeholder that can be enhanced later.

	// Future enhancement: Check npm registry, homebrew, etc. for latest versions
	// and compare with installed version using semantic versioning.

	return false, "", nil
}

// isAgentInstalled checks if an agent CLI is available in PATH
func isAgentInstalled(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// executeInstallCommand executes an installation command
func executeInstallCommand(installCmd string) error {
	// Parse the command - handle both simple commands and piped commands
	var cmd *exec.Cmd

	// Check if it's a curl piped command (common pattern)
	if strings.Contains(installCmd, "|") {
		// Execute via shell to handle pipes
		cmd = exec.Command("sh", "-c", installCmd)
	} else {
		// Parse as space-separated arguments
		parts := strings.Fields(installCmd)
		if len(parts) == 0 {
			return fmt.Errorf("empty install command")
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	// Set up output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Run the command
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
