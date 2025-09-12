package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kevinelliott/agentpipe/internal/version"
)

var (
	checkUpdate bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the current version of agentpipe and check for updates.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVar(&checkUpdate, "check-update", true, "Check for newer versions")
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println(version.GetVersionString())

	if checkUpdate {
		fmt.Println("\n🔍 Checking for updates...")
		hasUpdate, latestVersion, err := version.CheckForUpdate()
		
		if err != nil {
			// Only show error if it's not a silent failure
			if err.Error() != "" {
				fmt.Printf("   ⚠️  Could not check for updates: %v\n", err)
			}
			return
		}

		if hasUpdate {
			fmt.Printf("\n📦 Update available!\n")
			fmt.Printf("   Current version: %s (out of date)\n", version.GetShortVersion())
			fmt.Printf("   Latest version:  %s\n", latestVersion)
			fmt.Printf("\n   Update with: brew upgrade agentpipe\n")
			fmt.Printf("   Or download from: https://github.com/kevinelliott/agentpipe/releases/latest\n")
		} else if latestVersion != "" {
			fmt.Printf("   ✅ You're running the latest version! (%s)\n", latestVersion)
		} else {
			// Couldn't determine the latest version
			fmt.Printf("   ℹ️  Update check unavailable at this time\n")
		}
	}
}