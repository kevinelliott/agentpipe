package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kevinelliott/agentpipe/internal/bridge"
	"github.com/kevinelliott/agentpipe/internal/version"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

var (
	cfgFile           string
	showVersion       bool
	globalJSONEmitter *bridge.StdoutEmitter // Shared across root and run commands for --json mode
)

var rootCmd = &cobra.Command{
	Use:   "agentpipe",
	Short: "Orchestrate conversations between multiple AI agents",
	Long: `AgentPipe is a CLI and TUI application that enables multiple AI agents
to have conversations with each other. It supports various AI CLI tools like
Claude, Gemini, and Qwen, allowing them to communicate in a shared "room".`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Println(version.GetVersionString())

			// Quick update check
			if hasUpdate, latestVersion, err := version.CheckForUpdate(); err == nil && hasUpdate {
				fmt.Printf("\n📦 Update available: %s (current: %s)\n", latestVersion, version.GetShortVersion())
				fmt.Printf("   Run 'agentpipe version' for more details\n")
			}
			os.Exit(0)
		}
		// If no flags, show help
		cmd.Help()
	},
}

func Execute() {
	// Skip logo for --json commands for clean JSON output
	shouldSkipLogo := false
	if len(os.Args) >= 2 {
		// Check if --json flag is present anywhere in args
		for _, arg := range os.Args[1:] {
			if arg == "--json" {
				shouldSkipLogo = true
				break
			}
		}
	}

	if !shouldSkipLogo {
		PrintLogo()
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.agentpipe.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Show version information")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding verbose flag: %v\n", err)
	}
}

func initConfig() {
	// Check if --json flag is present
	isJSONMode := false
	for _, arg := range os.Args[1:] {
		if arg == "--json" {
			isJSONMode = true
			break
		}
	}

	// Initialize logger first
	level := zerolog.InfoLevel
	if viper.GetBool("verbose") {
		level = zerolog.DebugLevel
	}

	if isJSONMode {
		// JSON mode: create emitter and JSON writer for zerolog
		globalJSONEmitter = bridge.NewStdoutEmitter(version.GetShortVersion())
		jsonWriter := bridge.NewZerologJSONWriter(globalJSONEmitter)
		log.InitLogger(jsonWriter, level, false) // false = don't use pretty console output
	} else {
		// Normal mode: use pretty console output
		log.InitLogger(os.Stderr, level, true) // Use pretty console output for CLI
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		log.WithField("config_file", cfgFile).Debug("using specified config file")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.WithError(err).Error("failed to get home directory")
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".agentpipe")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.WithField("config_file", viper.ConfigFileUsed()).Info("loaded configuration file")
		if viper.GetBool("verbose") {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	} else {
		log.WithError(err).Debug("no config file found, using defaults")
	}
}
