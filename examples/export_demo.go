// Package main demonstrates the export functionality
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/export"
)

func main() {
	// Create sample conversation
	messages := []agent.Message{
		{
			AgentID:   "system",
			AgentName: "System",
			Content:   "Welcome to the brainstorming session!",
			Timestamp: time.Now().Unix(),
			Role:      "system",
		},
		{
			AgentID:   "claude",
			AgentName: "Claude",
			Content:   "I think we should focus on user experience improvements.",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
			Metrics: &agent.ResponseMetrics{
				Duration:     250 * time.Millisecond,
				InputTokens:  50,
				OutputTokens: 75,
				TotalTokens:  125,
				Model:        "claude-sonnet-4.5",
				Cost:         0.0012,
			},
		},
		{
			AgentID:   "gemini",
			AgentName: "Gemini",
			Content:   "Great idea! We could also improve performance.",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
			Metrics: &agent.ResponseMetrics{
				Duration:     180 * time.Millisecond,
				InputTokens:  125,
				OutputTokens: 60,
				TotalTokens:  185,
				Model:        "gemini-pro",
				Cost:         0.0009,
			},
		},
	}

	// Export to Markdown
	fmt.Println("=== Markdown Export ===")
	mdExporter := export.NewExporter(export.ExportOptions{
		Format:            export.FormatMarkdown,
		IncludeMetrics:    true,
		IncludeTimestamps: true,
		Title:             "Product Brainstorm Session",
	})
	if err := mdExporter.Export(messages, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Markdown export failed: %v\n", err)
	}

	fmt.Println("\n\n=== JSON Export ===")
	// Export to JSON
	jsonExporter := export.NewExporter(export.ExportOptions{
		Format:         export.FormatJSON,
		IncludeMetrics: true,
		Title:          "Product Brainstorm Session",
	})
	if err := jsonExporter.Export(messages, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "JSON export failed: %v\n", err)
	}

	fmt.Println("\n\n=== HTML Export (snippet) ===")
	// Export to HTML
	htmlExporter := export.NewExporter(export.ExportOptions{
		Format:            export.FormatHTML,
		IncludeMetrics:    true,
		IncludeTimestamps: true,
		Title:             "Product Brainstorm Session",
	})
	if err := htmlExporter.Export(messages, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "HTML export failed: %v\n", err)
	}
}
