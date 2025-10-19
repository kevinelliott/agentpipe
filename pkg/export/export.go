// Package export provides functionality to export conversations to different formats.
// Supported formats include JSON, Markdown, and HTML.
package export

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// Format represents the export format type.
type Format string

const (
	// FormatJSON exports conversation as JSON
	FormatJSON Format = "json"
	// FormatMarkdown exports conversation as Markdown
	FormatMarkdown Format = "markdown"
	// FormatHTML exports conversation as HTML
	FormatHTML Format = "html"
)

// ExportOptions contains options for exporting conversations.
type ExportOptions struct {
	// Format specifies the export format (json, markdown, html)
	Format Format
	// IncludeMetrics includes token counts and costs in export
	IncludeMetrics bool
	// IncludeTimestamps includes message timestamps in export
	IncludeTimestamps bool
	// Title is an optional title for the exported conversation
	Title string
}

// Exporter handles conversation exports to different formats.
type Exporter struct {
	options ExportOptions
}

// NewExporter creates a new Exporter with the given options.
func NewExporter(options ExportOptions) *Exporter {
	return &Exporter{
		options: options,
	}
}

// Export writes the conversation messages to the writer in the configured format.
func (e *Exporter) Export(messages []agent.Message, writer io.Writer) error {
	switch e.options.Format {
	case FormatJSON:
		return e.exportJSON(messages, writer)
	case FormatMarkdown:
		return e.exportMarkdown(messages, writer)
	case FormatHTML:
		return e.exportHTML(messages, writer)
	default:
		return fmt.Errorf("unsupported export format: %s", e.options.Format)
	}
}

// exportJSON exports messages as JSON.
func (e *Exporter) exportJSON(messages []agent.Message, writer io.Writer) error {
	output := struct {
		Title      string          `json:"title,omitempty"`
		ExportedAt string          `json:"exported_at"`
		Messages   []agent.Message `json:"messages"`
		Summary    *ExportSummary  `json:"summary,omitempty"`
	}{
		Title:      e.options.Title,
		ExportedAt: time.Now().Format(time.RFC3339),
		Messages:   messages,
	}

	if e.options.IncludeMetrics {
		output.Summary = calculateSummary(messages)
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// exportMarkdown exports messages as Markdown.
func (e *Exporter) exportMarkdown(messages []agent.Message, writer io.Writer) error {
	var sb strings.Builder

	// Title
	if e.options.Title != "" {
		sb.WriteString("# ")
		sb.WriteString(e.options.Title)
		sb.WriteString("\n\n")
	}

	// Export metadata
	sb.WriteString("*Exported: ")
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("*\n\n")

	// Summary
	if e.options.IncludeMetrics {
		summary := calculateSummary(messages)
		sb.WriteString("## Summary\n\n")
		sb.WriteString(fmt.Sprintf("- **Messages**: %d\n", summary.TotalMessages))
		sb.WriteString(fmt.Sprintf("- **Agents**: %d\n", summary.UniqueAgents))
		sb.WriteString(fmt.Sprintf("- **Total Tokens**: %d\n", summary.TotalTokens))
		sb.WriteString(fmt.Sprintf("- **Total Cost**: $%.4f\n", summary.TotalCost))
		sb.WriteString("\n---\n\n")
	}

	// Messages
	sb.WriteString("## Conversation\n\n")

	for _, msg := range messages {
		// Agent/System badge
		if msg.Role == "system" {
			sb.WriteString("### [SYSTEM]")
		} else {
			sb.WriteString("### ")
			sb.WriteString(msg.AgentName)
		}

		// Timestamp
		if e.options.IncludeTimestamps {
			sb.WriteString(" - ")
			sb.WriteString(time.Unix(msg.Timestamp, 0).Format("15:04:05"))
		}

		sb.WriteString("\n\n")

		// Content
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")

		// Metrics
		if e.options.IncludeMetrics && msg.Metrics != nil {
			sb.WriteString("*")
			sb.WriteString(fmt.Sprintf("Duration: %v | ", msg.Metrics.Duration))
			sb.WriteString(fmt.Sprintf("Tokens: %d | ", msg.Metrics.TotalTokens))
			sb.WriteString(fmt.Sprintf("Cost: $%.4f", msg.Metrics.Cost))
			sb.WriteString("*\n\n")
		}

		sb.WriteString("---\n\n")
	}

	_, err := writer.Write([]byte(sb.String()))
	return err
}

// exportHTML exports messages as HTML.
func (e *Exporter) exportHTML(messages []agent.Message, writer io.Writer) error {
	var sb strings.Builder

	// HTML header
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"en\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")

	title := e.options.Title
	if title == "" {
		title = "AgentPipe Conversation"
	}
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))

	// CSS
	sb.WriteString("  <style>\n")
	sb.WriteString(getCSS())
	sb.WriteString("  </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")

	// Header
	sb.WriteString("  <div class=\"container\">\n")
	sb.WriteString("    <header>\n")
	sb.WriteString(fmt.Sprintf("      <h1>%s</h1>\n", html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf("      <p class=\"export-date\">Exported: %s</p>\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("    </header>\n\n")

	// Summary
	if e.options.IncludeMetrics {
		summary := calculateSummary(messages)
		sb.WriteString("    <div class=\"summary\">\n")
		sb.WriteString("      <h2>Summary</h2>\n")
		sb.WriteString("      <div class=\"summary-stats\">\n")
		sb.WriteString(fmt.Sprintf("        <div class=\"stat\"><strong>Messages:</strong> %d</div>\n", summary.TotalMessages))
		sb.WriteString(fmt.Sprintf("        <div class=\"stat\"><strong>Agents:</strong> %d</div>\n", summary.UniqueAgents))
		sb.WriteString(fmt.Sprintf("        <div class=\"stat\"><strong>Total Tokens:</strong> %d</div>\n", summary.TotalTokens))
		sb.WriteString(fmt.Sprintf("        <div class=\"stat\"><strong>Total Cost:</strong> $%.4f</div>\n", summary.TotalCost))
		sb.WriteString("      </div>\n")
		sb.WriteString("    </div>\n\n")
	}

	// Messages
	sb.WriteString("    <div class=\"conversation\">\n")
	sb.WriteString("      <h2>Conversation</h2>\n")

	for _, msg := range messages {
		roleClass := "message-agent"
		if msg.Role == "system" {
			roleClass = "message-system"
		}

		sb.WriteString(fmt.Sprintf("      <div class=\"message %s\">\n", roleClass))

		// Header
		sb.WriteString("        <div class=\"message-header\">\n")
		if msg.Role == "system" {
			sb.WriteString("          <span class=\"agent-name system\">SYSTEM</span>\n")
		} else {
			sb.WriteString(fmt.Sprintf("          <span class=\"agent-name\">%s</span>\n", html.EscapeString(msg.AgentName)))
		}

		if e.options.IncludeTimestamps {
			timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
			sb.WriteString(fmt.Sprintf("          <span class=\"timestamp\">%s</span>\n", timestamp))
		}
		sb.WriteString("        </div>\n")

		// Content
		sb.WriteString("        <div class=\"message-content\">\n")
		// Convert newlines to <br> tags
		content := html.EscapeString(msg.Content)
		content = strings.ReplaceAll(content, "\n", "<br>")
		sb.WriteString("          ")
		sb.WriteString(content)
		sb.WriteString("\n")
		sb.WriteString("        </div>\n")

		// Metrics
		if e.options.IncludeMetrics && msg.Metrics != nil {
			sb.WriteString("        <div class=\"message-metrics\">\n")
			sb.WriteString(fmt.Sprintf("          Duration: %v | ", msg.Metrics.Duration))
			sb.WriteString(fmt.Sprintf("Tokens: %d | ", msg.Metrics.TotalTokens))
			sb.WriteString(fmt.Sprintf("Cost: $%.4f\n", msg.Metrics.Cost))
			sb.WriteString("        </div>\n")
		}

		sb.WriteString("      </div>\n\n")
	}

	sb.WriteString("    </div>\n")
	sb.WriteString("  </div>\n")
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")

	_, err := writer.Write([]byte(sb.String()))
	return err
}

// ExportSummary contains summary statistics for an exported conversation.
type ExportSummary struct {
	TotalMessages int     `json:"total_messages"`
	UniqueAgents  int     `json:"unique_agents"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}

// calculateSummary computes summary statistics from messages.
func calculateSummary(messages []agent.Message) *ExportSummary {
	summary := &ExportSummary{}
	agents := make(map[string]bool)

	for _, msg := range messages {
		summary.TotalMessages++
		agents[msg.AgentID] = true

		if msg.Metrics != nil {
			summary.TotalTokens += msg.Metrics.TotalTokens
			summary.TotalCost += msg.Metrics.Cost
		}
	}

	summary.UniqueAgents = len(agents)
	return summary
}

// getCSS returns the CSS styles for HTML export.
func getCSS() string {
	return `    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      line-height: 1.6;
      color: #333;
      max-width: 100%;
      margin: 0;
      padding: 0;
      background-color: #f5f5f5;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
      padding: 20px;
      background-color: white;
      box-shadow: 0 0 10px rgba(0,0,0,0.1);
    }
    header {
      border-bottom: 2px solid #e0e0e0;
      padding-bottom: 20px;
      margin-bottom: 30px;
    }
    h1 {
      margin: 0;
      color: #2c3e50;
    }
    h2 {
      color: #34495e;
      border-bottom: 1px solid #e0e0e0;
      padding-bottom: 10px;
    }
    .export-date {
      color: #7f8c8d;
      font-style: italic;
      margin: 10px 0 0 0;
    }
    .summary {
      background-color: #ecf0f1;
      padding: 20px;
      border-radius: 8px;
      margin-bottom: 30px;
    }
    .summary-stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 15px;
      margin-top: 15px;
    }
    .stat {
      background-color: white;
      padding: 10px;
      border-radius: 4px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    }
    .conversation {
      margin-top: 30px;
    }
    .message {
      margin-bottom: 25px;
      padding: 15px;
      border-radius: 8px;
      background-color: #fff;
      border-left: 4px solid #3498db;
      box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    }
    .message-system {
      border-left-color: #95a5a6;
      background-color: #fafafa;
    }
    .message-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 10px;
      padding-bottom: 8px;
      border-bottom: 1px solid #e0e0e0;
    }
    .agent-name {
      font-weight: bold;
      color: #2980b9;
      font-size: 1.1em;
    }
    .agent-name.system {
      color: #7f8c8d;
    }
    .timestamp {
      color: #95a5a6;
      font-size: 0.9em;
    }
    .message-content {
      margin: 10px 0;
      line-height: 1.8;
    }
    .message-metrics {
      margin-top: 10px;
      padding-top: 10px;
      border-top: 1px solid #e0e0e0;
      font-size: 0.85em;
      color: #7f8c8d;
      font-style: italic;
    }
    @media print {
      .container {
        box-shadow: none;
      }
      .message {
        break-inside: avoid;
      }
    }`
}
