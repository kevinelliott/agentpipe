package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/kevinelliott/agentpipe/internal/providers"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

var (
	providersJSONOutput bool
	providersVerbose    bool
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage AI provider configurations and pricing",
	Long: `Manage AI provider configurations and pricing data.

Provider pricing data is sourced from Catwalk's provider configs and can be
updated to get the latest pricing information.`,
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available providers and models",
	Long: `List all available AI providers and their models with pricing information.

By default, displays a human-readable table. Use --json for JSON output.`,
	Run: runProvidersList,
}

var providersShowCmd = &cobra.Command{
	Use:   "show <provider-id>",
	Short: "Show detailed information for a specific provider",
	Long: `Show detailed information for a specific provider, including all available
models and their pricing.

Example:
  agentpipe providers show anthropic
  agentpipe providers show openai --json`,
	Args: cobra.ExactArgs(1),
	Run:  runProvidersShow,
}

var providersUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update provider pricing data from Catwalk",
	Long: `Update provider pricing data by fetching the latest configurations from
Catwalk's GitHub repository.

This will download all provider configs and save them to:
  ~/.agentpipe/providers.json

The updated pricing will be used instead of the embedded defaults.`,
	Run: runProvidersUpdate,
}

func init() {
	rootCmd.AddCommand(providersCmd)
	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(providersShowCmd)
	providersCmd.AddCommand(providersUpdateCmd)

	providersListCmd.Flags().BoolVar(&providersJSONOutput, "json", false, "Output in JSON format")
	providersListCmd.Flags().BoolVarP(&providersVerbose, "verbose", "v", false, "Show detailed model information")

	providersShowCmd.Flags().BoolVar(&providersJSONOutput, "json", false, "Output in JSON format")
}

func runProvidersList(cmd *cobra.Command, args []string) {
	registry := providers.GetRegistry()
	allProviders := registry.ListProviders()

	if len(allProviders) == 0 {
		fmt.Println("No providers found")
		return
	}

	if providersJSONOutput {
		data, err := json.MarshalIndent(allProviders, "", "  ")
		if err != nil {
			log.WithError(err).Error("failed to marshal providers to JSON")
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Human-readable table output
	config := registry.GetConfig()
	fmt.Printf("Provider Pricing Data (v%s)\n", config.Version)
	fmt.Printf("Updated: %s\n", config.UpdatedAt)
	fmt.Printf("Source: %s\n\n", config.Source)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tID\tMODELS\tDEFAULT LARGE\tDEFAULT SMALL")
	fmt.Fprintln(w, "--------\t--\t------\t-------------\t-------------")

	for _, p := range allProviders {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			p.Name,
			p.ID,
			len(p.Models),
			truncate(p.DefaultLargeModelID, 30),
			truncate(p.DefaultSmallModelID, 30),
		)
	}
	w.Flush()

	if providersVerbose {
		fmt.Println("\nModels by Provider:")
		for _, p := range allProviders {
			fmt.Printf("\n%s (%s):\n", p.Name, p.ID)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  MODEL ID\tNAME\tINPUT $/1M\tOUTPUT $/1M\tCONTEXT")
			for _, m := range p.Models {
				fmt.Fprintf(w, "  %s\t%s\t$%.2f\t$%.2f\t%d\n",
					truncate(m.ID, 40),
					truncate(m.Name, 30),
					m.CostPer1MIn,
					m.CostPer1MOut,
					m.ContextWindow,
				)
			}
			w.Flush()
		}
	} else {
		fmt.Println("\nUse --verbose to show detailed model information")
	}
}

func runProvidersShow(cmd *cobra.Command, args []string) {
	providerID := args[0]
	registry := providers.GetRegistry()

	provider, err := registry.GetProvider(providerID)
	if err != nil {
		log.WithError(err).Errorf("provider not found: %s", providerID)
		os.Exit(1)
	}

	if providersJSONOutput {
		data, err := json.MarshalIndent(provider, "", "  ")
		if err != nil {
			log.WithError(err).Error("failed to marshal provider to JSON")
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Human-readable output
	fmt.Printf("Provider: %s (%s)\n", provider.Name, provider.ID)
	fmt.Printf("Type: %s\n", provider.Type)
	if provider.APIEndpoint != "" {
		fmt.Printf("API Endpoint: %s\n", provider.APIEndpoint)
	}
	if provider.APIKey != "" {
		fmt.Printf("API Key: %s\n", provider.APIKey)
	}
	if provider.DefaultLargeModelID != "" {
		fmt.Printf("Default Large Model: %s\n", provider.DefaultLargeModelID)
	}
	if provider.DefaultSmallModelID != "" {
		fmt.Printf("Default Small Model: %s\n", provider.DefaultSmallModelID)
	}

	fmt.Printf("\nModels (%d):\n", len(provider.Models))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MODEL ID\tNAME\tINPUT $/1M\tOUTPUT $/1M\tCONTEXT\tREASON\tATTACH")

	for _, m := range provider.Models {
		reason := "No"
		if m.CanReason {
			reason = "Yes"
		}
		attach := "No"
		if m.SupportsAttachments {
			attach = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t$%.2f\t$%.2f\t%d\t%s\t%s\n",
			m.ID,
			truncate(m.Name, 30),
			m.CostPer1MIn,
			m.CostPer1MOut,
			m.ContextWindow,
			reason,
			attach,
		)
	}
	w.Flush()
}

func runProvidersUpdate(cmd *cobra.Command, args []string) {
	fmt.Println("Fetching latest provider configs from Catwalk...")

	config, err := providers.FetchProvidersFromCatwalk()
	if err != nil {
		log.WithError(err).Error("failed to fetch provider configs")
		os.Exit(1)
	}

	fmt.Printf("Successfully fetched %d providers\n", len(config.Providers))

	// Save to ~/.agentpipe/providers.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.WithError(err).Error("failed to get home directory")
		os.Exit(1)
	}

	agentpipeDir := filepath.Join(homeDir, ".agentpipe")
	if mkdirErr := os.MkdirAll(agentpipeDir, 0755); mkdirErr != nil {
		log.WithError(mkdirErr).Error("failed to create .agentpipe directory")
		os.Exit(1)
	}

	outputPath := filepath.Join(agentpipeDir, "providers.json")
	data, marshalErr := json.MarshalIndent(config, "", "  ")
	if marshalErr != nil {
		log.WithError(marshalErr).Error("failed to marshal config to JSON")
		os.Exit(1)
	}

	if writeErr := os.WriteFile(outputPath, data, 0600); writeErr != nil {
		log.WithError(writeErr).Error("failed to write providers.json")
		os.Exit(1)
	}

	fmt.Printf("Saved provider config to: %s\n", outputPath)
	fmt.Printf("Updated at: %s\n", config.UpdatedAt)
	fmt.Printf("Source: %s\n", config.Source)
	fmt.Println("\nProvider pricing data has been updated successfully!")

	// Reload the registry to use the new config
	registry := providers.GetRegistry()
	if err := registry.Reload(); err != nil {
		log.WithError(err).Warn("failed to reload provider registry")
	} else {
		fmt.Println("Provider registry reloaded with new data")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Helper function to get provider summary
