package cmd

import (
	"encoding/json"
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
	listCurrent     bool
	listJSON        bool
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
  agentpipe agents list --outdated   # List outdated agents with version comparison
  agentpipe agents list --current    # Check latest versions for all agents`,
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

// agentsUpgradeCmd upgrades one or more agents
var agentsUpgradeCmd = &cobra.Command{
	Use:   "upgrade [agent...]",
	Short: "Upgrade AI agent CLIs",
	Long: `Upgrade one or more AI agent CLIs to the latest version. Use --all to upgrade all installed agents.

Examples:
  agentpipe agents upgrade claude         # Upgrade Claude CLI
  agentpipe agents upgrade claude ollama  # Upgrade multiple agents
  agentpipe agents upgrade --all          # Upgrade all installed agents`,
	Run: runAgentsUpgrade,
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsInstallCmd)
	agentsCmd.AddCommand(agentsUpgradeCmd)

	agentsListCmd.Flags().BoolVar(&listInstalled, "installed", false, "List only installed agents")
	agentsListCmd.Flags().BoolVar(&listOutdated, "outdated", false, "List outdated agents with version comparison table")
	agentsListCmd.Flags().BoolVar(&listCurrent, "current", false, "Check and display latest versions from the web")
	agentsListCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	agentsInstallCmd.Flags().BoolVar(&installAll, "all", false, "Install all agents")
	agentsUpgradeCmd.Flags().BoolVar(&installAll, "all", false, "Upgrade all agents")
}

// AgentListJSON represents an agent in JSON output
type AgentListJSON struct {
	Name          string `json:"name"`
	Command       string `json:"command"`
	Description   string `json:"description"`
	Docs          string `json:"docs"`
	Installed     bool   `json:"installed"`
	Path          string `json:"path,omitempty"`
	Version       string `json:"version,omitempty"`
	LatestVersion string `json:"latest_version,omitempty"`
	HasUpdate     bool   `json:"has_update,omitempty"`
	InstallCmd    string `json:"install_cmd,omitempty"`
}

func runAgentsList(cmd *cobra.Command, args []string) {
	agents := registry.GetAll()

	// Sort agents by name
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	// If --outdated flag is set, show comparison table
	if listOutdated {
		showOutdatedTable(agents)
		return
	}

	// If --current flag is set along with other modes, show version info
	showVersionInfo := listCurrent

	// Filter agents based on flags
	var filteredAgents []*registry.AgentDefinition
	for _, agent := range agents {
		installed := isAgentInstalled(agent.Command)

		// Apply filters
		if listInstalled && !installed {
			continue
		}

		filteredAgents = append(filteredAgents, agent)
	}

	// Handle JSON output
	if listJSON {
		outputAgentsJSON(filteredAgents, showVersionInfo)
		return
	}

	// Determine title based on flags
	title := "AI Agent CLIs"
	if listInstalled {
		title = "Installed AI Agent CLIs"
	}
	if showVersionInfo {
		title += " - Latest Versions"
	}

	fmt.Printf("\n%s\n", title)
	fmt.Println(strings.Repeat("=", 70))

	if len(filteredAgents) == 0 {
		fmt.Println("\nNo agents found matching the specified criteria.")
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
			version := registry.GetInstalledVersion(agent.Command)
			if version != "" {
				fmt.Printf("   Version: %s\n", version)
			}

			// Check for updates if --current is set
			if showVersionInfo && agent.PackageManager != "" {
				latest, err := agent.GetLatestVersion()
				if err == nil {
					fmt.Printf("   Latest:  %s", latest)
					if version != "" {
						cmp, _ := registry.CompareVersions(version, latest)
						if cmp < 0 {
							fmt.Printf(" ⚠️  (update available)")
						} else if cmp == 0 {
							fmt.Printf(" ✅ (up to date)")
						}
					}
					fmt.Println()
				} else {
					fmt.Printf("   Latest:  (unable to fetch: %v)\n", err)
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

			// Show latest version if --current is set and agent has package manager
			if showVersionInfo && agent.PackageManager != "" {
				latest, err := agent.GetLatestVersion()
				if err == nil {
					fmt.Printf("   Latest:  %s\n", latest)
				}
			}
		}

		fmt.Printf("   Docs: %s\n", agent.Docs)
	}

	fmt.Println()
}

// agentVersionRow represents version information for an agent
type agentVersionRow struct {
	name      string
	installed bool
	current   string
	latest    string
	hasUpdate bool
	canCheck  bool
}

// showOutdatedTable displays a table of agents with version comparison
func showOutdatedTable(agents []*registry.AgentDefinition) {

	// Fetch version info in parallel
	type versionResult struct {
		index   int
		current string
		latest  string
		err     error
	}

	resultChan := make(chan versionResult, len(agents))
	rows := make([]agentVersionRow, len(agents))
	outdatedCount := 0

	// Launch parallel version checks
	for i, agent := range agents {
		go func(index int, ag *registry.AgentDefinition) {
			result := versionResult{index: index}

			// Check if installed and get current version
			installed := isAgentInstalled(ag.Command)
			if installed {
				result.current = registry.GetInstalledVersion(ag.Command)
				if result.current == "" {
					result.current = "unknown"
				}
			} else {
				result.current = "not installed"
			}

			// Fetch latest version if package manager is configured
			if ag.PackageManager != "" {
				latest, err := ag.GetLatestVersion()
				if err == nil {
					result.latest = latest
				} else {
					result.err = err
				}
			} else {
				result.latest = "manual install"
			}

			resultChan <- result
		}(i, agent)
	}

	// Collect results
	for i := 0; i < len(agents); i++ {
		result := <-resultChan
		agent := agents[result.index]
		installed := result.current != "not installed"

		r := agentVersionRow{
			name:      agent.Name,
			installed: installed,
			current:   result.current,
			canCheck:  agent.PackageManager != "",
		}

		if result.err != nil {
			r.latest = fmt.Sprintf("(error: %v)", result.err)
		} else {
			r.latest = result.latest
		}

		// Check for updates
		if installed && r.current != "unknown" && result.latest != "" && result.latest != "manual install" && result.err == nil {
			cmp, err := registry.CompareVersions(r.current, result.latest)
			if err == nil && cmp < 0 {
				r.hasUpdate = true
				outdatedCount++
			}
		}

		rows[result.index] = r
	}
	close(resultChan)

	// Output JSON format if requested
	if listJSON {
		outputOutdatedJSON(rows)
		return
	}

	// Human-readable format
	fmt.Println("\n📊 Agent Version Status")
	fmt.Println(strings.Repeat("=", 85))
	fmt.Println()

	// Print table header
	fmt.Printf("%-12s  %-24s  %-24s  %s\n",
		"Agent", "Installed Version", "Latest Version", "Update")
	fmt.Println(strings.Repeat("-", 85))

	// Print table rows
	for _, r := range rows {
		update := ""
		if r.hasUpdate {
			update = "⚠️  Available"
		} else if r.installed && r.canCheck && r.current != "unknown" {
			update = "✅ Up to date"
		}

		fmt.Printf("%-12s  %-24s  %-24s  %s\n",
			r.name, r.current, r.latest, update)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 85))
	fmt.Printf("\nSummary: %d agent(s) with updates available\n", outdatedCount)
	if outdatedCount > 0 {
		fmt.Println("\nTo upgrade an agent, use: agentpipe agents upgrade <agent>")
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

func runAgentsUpgrade(cmd *cobra.Command, args []string) {
	var agentsToUpgrade []*registry.AgentDefinition

	if installAll {
		// Upgrade all installed agents
		allAgents := registry.GetAll()
		for _, agent := range allAgents {
			if isAgentInstalled(agent.Command) {
				agentsToUpgrade = append(agentsToUpgrade, agent)
			}
		}
		if len(agentsToUpgrade) == 0 {
			fmt.Println("\nNo agents are currently installed.")
			return
		}
		fmt.Printf("\nUpgrading %d installed agent(s)...\n", len(agentsToUpgrade))
	} else if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Please specify at least one agent to upgrade, or use --all\n")
		fmt.Fprintf(os.Stderr, "Usage: agentpipe agents upgrade [agent...]\n")
		fmt.Fprintf(os.Stderr, "       agentpipe agents upgrade --all\n\n")
		fmt.Fprintf(os.Stderr, "Run 'agentpipe agents list --outdated' to see agents with updates available\n")
		os.Exit(1)
		return
	} else {
		// Upgrade specific agents
		for _, name := range args {
			agent, err := registry.GetByName(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Agent '%s' not found in registry\n", name)
				fmt.Fprintf(os.Stderr, "Run 'agentpipe agents list' to see available agents\n")
				os.Exit(1)
				return
			}
			if !isAgentInstalled(agent.Command) {
				fmt.Fprintf(os.Stderr, "Error: Agent '%s' is not currently installed\n", name)
				fmt.Fprintf(os.Stderr, "Use 'agentpipe agents install %s' to install it first\n", name)
				os.Exit(1)
				return
			}
			agentsToUpgrade = append(agentsToUpgrade, agent)
		}
	}

	// Track upgrade results
	successCount := 0
	skipCount := 0
	failCount := 0

	fmt.Println()

	for _, agent := range agentsToUpgrade {
		// Get upgrade command
		upgradeCmd, err := agent.GetUpgradeCommand()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %s: %v\n", agent.Name, err)
			failCount++
			continue
		}

		// Check if upgradeable via command
		if strings.HasPrefix(upgradeCmd, "See ") {
			fmt.Printf("ℹ️  %s: %s\n", agent.Name, upgradeCmd)
			skipCount++
			continue
		}

		// Execute upgrade
		fmt.Printf("⬆️  Upgrading %s...\n", agent.Name)
		fmt.Printf("   Running: %s\n", upgradeCmd)

		if err := executeInstallCommand(upgradeCmd); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to upgrade %s: %v\n", agent.Name, err)
			failCount++
			continue
		}

		fmt.Printf("✅ Successfully upgraded %s\n", agent.Name)
		successCount++
		fmt.Println()
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("\nUpgrade Summary:\n")
	fmt.Printf("  ✅ Upgraded: %d\n", successCount)
	if skipCount > 0 {
		fmt.Printf("  ⏭️  Skipped:  %d\n", skipCount)
	}
	if failCount > 0 {
		fmt.Printf("  ❌ Failed:   %d\n", failCount)
	}
	fmt.Println()

	if failCount > 0 {
		os.Exit(1)
	}
}

// outputAgentsJSON outputs agent list in JSON format
func outputAgentsJSON(agents []*registry.AgentDefinition, showVersionInfo bool) {
	var jsonAgents []AgentListJSON

	for _, agent := range agents {
		installed := isAgentInstalled(agent.Command)

		agentJSON := AgentListJSON{
			Name:        agent.Name,
			Command:     agent.Command,
			Description: agent.Description,
			Docs:        agent.Docs,
			Installed:   installed,
		}

		if installed {
			if path, err := exec.LookPath(agent.Command); err == nil {
				agentJSON.Path = path
			}

			version := registry.GetInstalledVersion(agent.Command)
			if version != "" {
				agentJSON.Version = version
			}

			// Check for updates if showVersionInfo is true
			if showVersionInfo && agent.PackageManager != "" {
				latest, err := agent.GetLatestVersion()
				if err == nil {
					agentJSON.LatestVersion = latest
					if version != "" {
						cmp, _ := registry.CompareVersions(version, latest)
						agentJSON.HasUpdate = cmp < 0
					}
				}
			}
		} else {
			// Get install command for non-installed agents
			installCmd, err := agent.GetInstallCommand()
			if err == nil {
				if agent.IsInstallable() {
					agentJSON.InstallCmd = fmt.Sprintf("agentpipe agents install %s", strings.ToLower(agent.Name))
				} else {
					agentJSON.InstallCmd = installCmd
				}
			}

			// Show latest version if showVersionInfo is set
			if showVersionInfo && agent.PackageManager != "" {
				latest, err := agent.GetLatestVersion()
				if err == nil {
					agentJSON.LatestVersion = latest
				}
			}
		}

		jsonAgents = append(jsonAgents, agentJSON)
	}

	output, err := json.MarshalIndent(jsonAgents, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

// outputOutdatedJSON outputs agent version information in JSON format
func outputOutdatedJSON(rows []agentVersionRow) {
	type OutdatedAgentJSON struct {
		Name          string `json:"name"`
		Installed     bool   `json:"installed"`
		CurrentVer    string `json:"current_version"`
		LatestVer     string `json:"latest_version"`
		HasUpdate     bool   `json:"has_update"`
		CanCheckVer   bool   `json:"can_check_version"`
	}

	var jsonAgents []OutdatedAgentJSON
	for _, r := range rows {
		jsonAgents = append(jsonAgents, OutdatedAgentJSON{
			Name:        r.name,
			Installed:   r.installed,
			CurrentVer:  r.current,
			LatestVer:   r.latest,
			HasUpdate:   r.hasUpdate,
			CanCheckVer: r.canCheck,
		})
	}

	output, err := json.MarshalIndent(jsonAgents, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
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
