package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kevinelliott/agentpipe/pkg/conversation"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <state-file>",
	Short: "Resume a saved conversation",
	Long: `Resume a conversation from a previously saved state file.

The state file contains the conversation history, configuration, and metadata.
You can resume the conversation and continue where you left off.

Example:
  agentpipe resume ~/.agentpipe/states/conversation-20231215-143022.json
  agentpipe resume --list  # List all saved states`,
	Args: cobra.MaximumNArgs(1),
	Run:  runResume,
}

var (
	listStates           bool
	continueConversation bool
)

func init() {
	rootCmd.AddCommand(resumeCmd)

	resumeCmd.Flags().BoolVar(&listStates, "list", false, "List all saved conversation states")
	resumeCmd.Flags().BoolVar(&continueConversation, "continue", false, "Continue the conversation (default: just load and display)")
}

func runResume(cmd *cobra.Command, args []string) {
	if listStates {
		listSavedStates()
		return
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: State file path required")
		fmt.Fprintln(os.Stderr, "Use 'agentpipe resume --list' to see available states")
		os.Exit(1)
	}

	statePath := args[0]

	log.WithField("state_path", statePath).Info("resuming conversation from state file")

	// Load state
	state, err := conversation.LoadState(statePath)
	if err != nil {
		log.WithError(err).WithField("state_path", statePath).Error("failed to load conversation state")
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Display state information
	fmt.Println("📂 Loaded conversation state")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Saved at:        %s\n", state.SavedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Started at:      %s\n", state.Metadata.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total messages:  %d\n", len(state.Messages))
	fmt.Printf("Total turns:     %d\n", state.Metadata.TotalTurns)

	if state.Config != nil {
		fmt.Printf("Mode:            %s\n", state.Config.Orchestrator.Mode)
		fmt.Printf("Max turns:       %d\n", state.Config.Orchestrator.MaxTurns)
		fmt.Printf("Agents:          %d\n", len(state.Config.Agents))

		if len(state.Config.Agents) > 0 {
			fmt.Println("\nAgents:")
			for _, agent := range state.Config.Agents {
				fmt.Printf("  - %s (%s)\n", agent.Name, agent.Type)
			}
		}
	}

	if state.Metadata.Description != "" {
		fmt.Printf("\nDescription:     %s\n", state.Metadata.Description)
	}

	fmt.Println(strings.Repeat("=", 60))

	// Display recent messages
	if len(state.Messages) > 0 {
		fmt.Println("\n💬 Recent messages:")
		fmt.Println(strings.Repeat("-", 60))

		start := 0
		if len(state.Messages) > 5 {
			start = len(state.Messages) - 5
			fmt.Printf("(Showing last 5 of %d messages)\n\n", len(state.Messages))
		}

		for i := start; i < len(state.Messages); i++ {
			msg := state.Messages[i]
			fmt.Printf("[%s] %s\n", msg.AgentName, msg.Content)
		}
		fmt.Println(strings.Repeat("-", 60))
	}

	if continueConversation {
		fmt.Println("\n🚀 Continuing conversation...")

		// TODO: Implement conversation continuation
		// This would require:
		// 1. Reinitializing agents from config
		// 2. Loading message history into orchestrator
		// 3. Resuming the conversation loop

		fmt.Println("Note: Conversation continuation is not yet implemented.")
		fmt.Println("For now, you can:")
		fmt.Println("  1. View the saved state with: agentpipe resume <state-file>")
		fmt.Println("  2. Export to different formats with: agentpipe export <state-file>")
		fmt.Println("  3. Start a new conversation with the same config")
	}
}

func listSavedStates() {
	stateDir, err := conversation.GetDefaultStateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state directory: %v\n", err)
		os.Exit(1)
	}

	log.WithField("state_dir", stateDir).Debug("listing saved states")

	states, err := conversation.ListStates(stateDir)
	if err != nil {
		log.WithError(err).WithField("state_dir", stateDir).Error("failed to list states")
		fmt.Fprintf(os.Stderr, "Error listing states: %v\n", err)
		os.Exit(1)
	}

	if len(states) == 0 {
		fmt.Println("No saved conversation states found.")
		fmt.Printf("States are saved to: %s\n", stateDir)
		fmt.Println("\nTo save a conversation state, use:")
		fmt.Println("  agentpipe run -c config.yaml --save-state")
		return
	}

	fmt.Printf("📚 Saved conversation states (%d found):\n", len(states))
	fmt.Println(strings.Repeat("=", 60))

	for i, statePath := range states {
		info, err := conversation.GetStateInfo(statePath)
		if err != nil {
			log.WithError(err).WithField("state_path", statePath).Warn("failed to read state info")
			fmt.Printf("%d. %s (error reading info)\n", i+1, statePath)
			continue
		}

		fmt.Printf("\n%d. %s\n", i+1, statePath)
		fmt.Printf("   Saved:    %s\n", info.SavedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Messages: %d\n", info.Messages)
		fmt.Printf("   Agents:   %d\n", info.AgentCount)
		fmt.Printf("   Mode:     %s\n", info.Mode)

		if info.Description != "" {
			fmt.Printf("   Desc:     %s\n", info.Description)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nTo resume a conversation:")
	fmt.Println("  agentpipe resume <state-file>")
}
