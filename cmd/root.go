package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kevinelliott/agentpipe/internal/version"
)

var (
	cfgFile     string
	showVersion bool
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
		_ = cmd.Help()
	},
}

func Execute() {
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
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
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
		if viper.GetBool("verbose") {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}
