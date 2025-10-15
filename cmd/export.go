package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/export"
)

var exportCmd = &cobra.Command{
	Use:   "export [log-file]",
	Short: "Export a conversation to different formats",
	Long: `Export a conversation log file to JSON, Markdown, or HTML format.

The export command reads a conversation log file and converts it to the specified
format with optional metrics and timestamps.

Examples:
  # Export to JSON
  agentpipe export ~/.agentpipe/chats/conversation_20231015.txt --format json

  # Export to Markdown with metrics
  agentpipe export chat.txt --format markdown --metrics

  # Export to HTML with custom title
  agentpipe export chat.txt --format html --title "Team Brainstorm"

  # Export latest conversation
  agentpipe export --latest --format markdown
`,
	RunE: runExport,
}

var (
	exportFormat     string
	exportOutput     string
	exportMetrics    bool
	exportTimestamps bool
	exportTitle      string
	exportLatest     bool
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "markdown", "Export format (json, markdown, html)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
	exportCmd.Flags().BoolVar(&exportMetrics, "metrics", true, "Include metrics (tokens, cost)")
	exportCmd.Flags().BoolVar(&exportTimestamps, "timestamps", true, "Include timestamps")
	exportCmd.Flags().StringVar(&exportTitle, "title", "", "Conversation title")
	exportCmd.Flags().BoolVar(&exportLatest, "latest", false, "Export the latest conversation")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Determine input file
	var inputFile string
	if exportLatest {
		// Find latest conversation in default log directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		logDir := filepath.Join(homeDir, ".agentpipe", "chats")
		latest, err := findLatestLog(logDir)
		if err != nil {
			return fmt.Errorf("failed to find latest log: %w", err)
		}
		inputFile = latest
		fmt.Fprintf(os.Stderr, "Exporting latest conversation: %s\n", filepath.Base(inputFile))
	} else {
		if len(args) == 0 {
			return fmt.Errorf("log file path required (or use --latest flag)")
		}
		inputFile = args[0]
	}

	// Read messages from log file
	messages, err := readLogFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages found in log file")
	}

	// Determine export format
	format := export.Format(strings.ToLower(exportFormat))
	switch format {
	case export.FormatJSON, export.FormatMarkdown, export.FormatHTML:
		// Valid format
	default:
		return fmt.Errorf("invalid format: %s (use json, markdown, or html)", exportFormat)
	}

	// Set default title if not provided
	title := exportTitle
	if title == "" {
		title = fmt.Sprintf("Conversation - %s", filepath.Base(inputFile))
	}

	// Create exporter
	exporter := export.NewExporter(export.ExportOptions{
		Format:            format,
		IncludeMetrics:    exportMetrics,
		IncludeTimestamps: exportTimestamps,
		Title:             title,
	})

	// Determine output writer
	var writer *os.File
	if exportOutput == "" {
		writer = os.Stdout
	} else {
		f, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	}

	// Export
	if err := exporter.Export(messages, writer); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Print success message to stderr (so it doesn't mix with output)
	if exportOutput != "" {
		fmt.Fprintf(os.Stderr, "✅ Exported %d messages to %s\n", len(messages), exportOutput)
	}

	return nil
}

// readLogFile reads and parses a conversation log file.
// This is a simplified implementation - in production, you'd want more robust parsing.
func readLogFile(path string) ([]agent.Message, error) {
	// For now, return an error indicating this needs the logger package support
	// In a real implementation, this would parse the log file format
	return nil, fmt.Errorf("log file parsing not yet implemented - this feature requires log file format support")
}

// findLatestLog finds the most recent log file in the given directory.
func findLatestLog(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var latestFile string
	var latestTime int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Look for text or JSON log files
		name := entry.Name()
		if !strings.HasSuffix(name, ".txt") && !strings.HasSuffix(name, ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Unix() > latestTime {
			latestTime = info.ModTime().Unix()
			latestFile = filepath.Join(dir, name)
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("no log files found in %s", dir)
	}

	return latestFile, nil
}
