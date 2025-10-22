package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kevinelliott/agentpipe/internal/bridge"
	"github.com/kevinelliott/agentpipe/internal/version"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

var (
	bridgeJSON bool
)

// bridgeCmd represents the bridge command
var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Manage streaming bridge configuration",
	Long: `The bridge command manages real-time streaming of conversations to AgentPipe Web.

The streaming bridge is opt-in and allows you to view your conversations
in real-time through the AgentPipe Web interface at https://agentpipe.ai.

Subcommands:
  setup    - Interactive wizard to configure the bridge
  status   - Show current bridge status and configuration
  test     - Test the bridge connection
  disable  - Disable bridge streaming`,
}

// bridgeSetupCmd configures the streaming bridge
var bridgeSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure streaming bridge",
	Long: `Interactive wizard to configure the streaming bridge.

This will guide you through:
1. Enabling/disabling the bridge
2. Setting the AgentPipe Web URL
3. Configuring your API key
4. Setting timeout and retry options

Your API key is stored in your agentpipe configuration file and never logged.`,
	Run: func(cmd *cobra.Command, args []string) {
		runBridgeSetup()
	},
}

// bridgeStatusCmd shows bridge status
var bridgeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show bridge status",
	Long: `Display the current streaming bridge configuration and status.

Shows:
- Whether the bridge is enabled
- Configured URL
- API key status (present/missing, never shows actual key)
- Timeout and retry settings
- Current configuration source`,
	Run: func(cmd *cobra.Command, args []string) {
		runBridgeStatus()
	},
}

// bridgeTestCmd tests the bridge connection
var bridgeTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test bridge connection",
	Long: `Test the streaming bridge connection by sending a test event.

This will:
1. Load your bridge configuration
2. Send a test conversation.started event
3. Report success or failure

This helps verify your API key and network connectivity.`,
	Run: func(cmd *cobra.Command, args []string) {
		runBridgeTest()
	},
}

// bridgeDisableCmd disables the bridge
var bridgeDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable bridge streaming",
	Long: `Disable the streaming bridge.

This sets bridge.enabled to false in your configuration.
Your API key and other settings are preserved.`,
	Run: func(cmd *cobra.Command, args []string) {
		runBridgeDisable()
	},
}

func init() {
	rootCmd.AddCommand(bridgeCmd)
	bridgeCmd.AddCommand(bridgeSetupCmd)
	bridgeCmd.AddCommand(bridgeStatusCmd)
	bridgeCmd.AddCommand(bridgeTestCmd)
	bridgeCmd.AddCommand(bridgeDisableCmd)

	// Add --json flag to status command
	bridgeStatusCmd.Flags().BoolVar(&bridgeJSON, "json", false, "Output status as JSON")
}

func runBridgeSetup() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║        AgentPipe Streaming Bridge Setup Wizard          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Load current config
	currentConfig := bridge.LoadConfig()

	fmt.Println("The streaming bridge sends real-time conversation updates to")
	fmt.Println("AgentPipe Web, allowing you to view conversations in a browser.")
	fmt.Println()
	fmt.Printf("Current status: %s\n", enabledStatus(currentConfig.Enabled))
	fmt.Println()

	// Ask if they want to enable
	fmt.Print("Enable streaming bridge? [y/N]: ")
	enableInput, _ := reader.ReadString('\n')
	enableInput = strings.TrimSpace(strings.ToLower(enableInput))

	if enableInput != "y" && enableInput != "yes" {
		// Disable and exit
		viper.Set("bridge.enabled", false)

		// Try to write config, create if doesn't exist
		if err := viper.WriteConfig(); err != nil {
			// If config doesn't exist, create it
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Get home directory and create config file
				home, homeErr := os.UserHomeDir()
				if homeErr != nil {
					fmt.Printf("Error getting home directory: %v\n", homeErr)
					os.Exit(1)
				}
				configPath := fmt.Sprintf("%s/.agentpipe.yaml", home)
				if writeErr := viper.WriteConfigAs(configPath); writeErr != nil {
					fmt.Printf("Error creating configuration file: %v\n", writeErr)
					os.Exit(1)
				}
			} else {
				fmt.Printf("Error saving config: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println("✓ Bridge disabled")
		return
	}

	// Get URL (with default)
	defaultURL := currentConfig.URL
	if defaultURL == "" {
		defaultURL = bridge.DefaultURL
	}
	fmt.Printf("\nAgentPipe Web URL [%s]: ", defaultURL)
	urlInput, _ := reader.ReadString('\n')
	urlInput = strings.TrimSpace(urlInput)
	if urlInput == "" {
		urlInput = defaultURL
	}

	// Get API key
	fmt.Print("\nAPI Key: ")
	apiKeyInput, _ := reader.ReadString('\n')
	apiKeyInput = strings.TrimSpace(apiKeyInput)

	if apiKeyInput == "" {
		fmt.Println("✗ API key is required")
		os.Exit(1)
	}

	// Optional: timeout
	fmt.Printf("\nRequest timeout in milliseconds [%d]: ", currentConfig.TimeoutMs)
	timeoutInput, _ := reader.ReadString('\n')
	timeoutInput = strings.TrimSpace(timeoutInput)
	timeoutMs := currentConfig.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = 10000
	}
	if timeoutInput != "" {
		_, _ = fmt.Sscanf(timeoutInput, "%d", &timeoutMs)
	}

	// Optional: retries
	fmt.Printf("Retry attempts [%d]: ", currentConfig.RetryAttempts)
	retryInput, _ := reader.ReadString('\n')
	retryInput = strings.TrimSpace(retryInput)
	retryAttempts := currentConfig.RetryAttempts
	if retryAttempts == 0 {
		retryAttempts = 3
	}
	if retryInput != "" {
		_, _ = fmt.Sscanf(retryInput, "%d", &retryAttempts)
	}

	// Save configuration
	viper.Set("bridge.enabled", true)
	viper.Set("bridge.url", urlInput)
	viper.Set("bridge.api_key", apiKeyInput)
	viper.Set("bridge.timeout_ms", timeoutMs)
	viper.Set("bridge.retry_attempts", retryAttempts)
	viper.Set("bridge.log_level", "info")

	// Try to write config, create if doesn't exist
	if err := viper.WriteConfig(); err != nil {
		// If config doesn't exist, create it
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Get home directory and create config file
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				fmt.Printf("\n✗ Error getting home directory: %v\n", homeErr)
				os.Exit(1)
			}
			configPath := fmt.Sprintf("%s/.agentpipe.yaml", home)
			if writeErr := viper.WriteConfigAs(configPath); writeErr != nil {
				fmt.Printf("\n✗ Error creating configuration file: %v\n", writeErr)
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n✗ Error saving configuration: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\n✓ Bridge configuration saved successfully!")
	fmt.Println("\nRun 'agentpipe bridge test' to verify your connection.")
}

func runBridgeStatus() {
	config := bridge.LoadConfig()

	if bridgeJSON {
		outputStatusJSON(config)
		return
	}

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║          AgentPipe Streaming Bridge Status              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Enabled:        %s\n", enabledStatus(config.Enabled))
	fmt.Printf("URL:            %s\n", config.URL)
	fmt.Printf("API Key:        %s\n", apiKeyStatus(config.APIKey))
	fmt.Printf("Timeout:        %dms\n", config.TimeoutMs)
	fmt.Printf("Retry Attempts: %d\n", config.RetryAttempts)
	fmt.Printf("Log Level:      %s\n", config.LogLevel)
	fmt.Println()

	// Show configuration source
	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		fmt.Printf("Configuration: %s\n", configFile)
	}

	// Show environment overrides
	if os.Getenv("AGENTPIPE_STREAM_ENABLED") != "" {
		fmt.Println("\n⚠ AGENTPIPE_STREAM_ENABLED environment variable is set")
	}
	if os.Getenv("AGENTPIPE_STREAM_URL") != "" {
		fmt.Println("⚠ AGENTPIPE_STREAM_URL environment variable is set")
	}
	if os.Getenv("AGENTPIPE_STREAM_API_KEY") != "" {
		fmt.Println("⚠ AGENTPIPE_STREAM_API_KEY environment variable is set")
	}
}

func runBridgeTest() {
	fmt.Println("Testing streaming bridge connection...")
	fmt.Println()

	config := bridge.LoadConfig()

	if !config.Enabled {
		fmt.Println("✗ Bridge is not enabled")
		fmt.Println("  Run 'agentpipe bridge setup' to configure")
		os.Exit(1)
	}

	if config.APIKey == "" {
		fmt.Println("✗ No API key configured")
		fmt.Println("  Run 'agentpipe bridge setup' to configure")
		os.Exit(1)
	}

	fmt.Printf("URL:     %s\n", config.URL)
	fmt.Printf("Timeout: %dms\n", config.TimeoutMs)
	fmt.Println()

	// Send test event
	fmt.Println("Sending test event...")
	client := bridge.NewClient(config)

	event := &bridge.Event{
		Type:      bridge.EventBridgeTest,
		Timestamp: bridge.UTCTime{Time: time.Now()},
		Data: bridge.BridgeTestData{
			Message:    "Bridge connection test",
			SystemInfo: bridge.CollectSystemInfo(version.GetShortVersion()),
		},
	}

	err := client.SendEvent(event)
	if err != nil {
		fmt.Printf("✗ Connection failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Connection successful!")
}

func runBridgeDisable() {
	viper.Set("bridge.enabled", false)

	// Try to write config, create if doesn't exist
	if err := viper.WriteConfig(); err != nil {
		// If config doesn't exist, create it
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Get home directory and create config file
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				fmt.Printf("Error getting home directory: %v\n", homeErr)
				os.Exit(1)
			}
			configPath := fmt.Sprintf("%s/.agentpipe.yaml", home)
			if writeErr := viper.WriteConfigAs(configPath); writeErr != nil {
				fmt.Printf("Error creating configuration file: %v\n", writeErr)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error saving configuration: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("✓ Bridge disabled")
	fmt.Println("  Your API key and other settings have been preserved")
	fmt.Println("  Run 'agentpipe bridge setup' to re-enable")
}

// Helper functions

func enabledStatus(enabled bool) string {
	if enabled {
		return "✓ Enabled"
	}
	return "✗ Disabled"
}

func apiKeyStatus(apiKey string) string {
	if apiKey == "" {
		return "✗ Not configured"
	}
	// Show first 4 chars and last 4 chars for verification
	if len(apiKey) <= 8 {
		return "✓ Configured (***)"
	}
	return fmt.Sprintf("✓ Configured (%s...%s)", apiKey[:4], apiKey[len(apiKey)-4:])
}

type BridgeStatusJSON struct {
	Enabled       bool   `json:"enabled"`
	URL           string `json:"url"`
	HasAPIKey     bool   `json:"has_api_key"`
	TimeoutMs     int    `json:"timeout_ms"`
	RetryAttempts int    `json:"retry_attempts"`
	LogLevel      string `json:"log_level"`
	ConfigFile    string `json:"config_file,omitempty"`
}

func outputStatusJSON(config *bridge.Config) {
	status := BridgeStatusJSON{
		Enabled:       config.Enabled,
		URL:           config.URL,
		HasAPIKey:     config.APIKey != "",
		TimeoutMs:     config.TimeoutMs,
		RetryAttempts: config.RetryAttempts,
		LogLevel:      config.LogLevel,
		ConfigFile:    viper.ConfigFileUsed(),
	}

	output, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		log.WithError(err).Error("failed to marshal bridge status to JSON")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}
