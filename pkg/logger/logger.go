package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/charmbracelet/lipgloss"
)

type ChatLogger struct {
	logFile     *os.File
	logFormat   string
	console     io.Writer
	agentColors map[string]lipgloss.Style
	colorIndex  int
	termWidth   int
	showMetrics bool
}

var colors = []lipgloss.Color{
	lipgloss.Color("63"),  // Blue
	lipgloss.Color("212"), // Pink
	lipgloss.Color("86"),  // Green
	lipgloss.Color("214"), // Orange
	lipgloss.Color("99"),  // Purple
	lipgloss.Color("51"),  // Cyan
	lipgloss.Color("226"), // Yellow
	lipgloss.Color("201"), // Magenta
}

var (
	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)
	
	systemBadgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("244")).
			Padding(0, 1).
			MarginRight(1)
	
	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
	
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
	
	errorBadgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)
	
	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("236"))
	
	messageBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			PaddingLeft(1).
			PaddingRight(1).
			MarginBottom(1)
)

func NewChatLogger(logDir string, logFormat string, console io.Writer, showMetrics bool) (*ChatLogger, error) {
	if logDir == "" {
		return &ChatLogger{
			console:     console,
			agentColors: make(map[string]lipgloss.Style),
			termWidth:   80,
			showMetrics: showMetrics,
		}, nil
	}
	
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("chat_%s.log", timestamp))
	
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	
	// Get terminal width
	termWidth := 80
	if width, _, err := getTerminalSize(); err == nil && width > 0 {
		termWidth = width
	}
	
	logger := &ChatLogger{
		logFile:     logFile,
		logFormat:   logFormat,
		console:     console,
		agentColors: make(map[string]lipgloss.Style),
		termWidth:   termWidth,
		showMetrics: showMetrics,
	}
	
	// Write header to log file
	logger.writeToFile(fmt.Sprintf("=== AgentPipe Chat Log ===\n"))
	logger.writeToFile(fmt.Sprintf("Started: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	logger.writeToFile(fmt.Sprintf("=====================================\n\n"))
	
	if console != nil {
		fmt.Fprintf(console, "\n📝 Chat logged to: %s\n", logPath)
	}
	
	return logger, nil
}

func (l *ChatLogger) getAgentColor(agentName string) lipgloss.Style {
	if style, exists := l.agentColors[agentName]; exists {
		return style
	}
	
	// Assign a new color
	color := colors[l.colorIndex%len(colors)]
	l.colorIndex++
	
	style := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)
	
	l.agentColors[agentName] = style
	return style
}

func (l *ChatLogger) getAgentBadgeStyle(agentName string) lipgloss.Style {
	if style, exists := l.agentColors[agentName]; exists {
		color := style.GetForeground()
		return lipgloss.NewStyle().
			Background(color).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)
	}
	
	// Default badge
	return lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1).
		MarginRight(1)
}

func (l *ChatLogger) LogMessage(msg agent.Message) {
	timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")
	
	// Write to file
	if l.logFile != nil {
		if l.logFormat == "json" {
			data, _ := json.Marshal(msg)
			l.writeToFile(string(data) + "\n")
		} else {
			l.writeToFile(fmt.Sprintf("[%s] %s (%s): %s\n\n", 
				timestamp, msg.AgentName, msg.Role, msg.Content))
		}
	}
	
	// Write to console with colors
	if l.console != nil {
		var output strings.Builder
		
		// Add a subtle separator line
		output.WriteString(separatorStyle.Render(strings.Repeat("─", min(l.termWidth, 80))))
		output.WriteString("\n")
		
		// Format timestamp with icon
		output.WriteString(timestampStyle.Render("🕐 " + timestamp + " "))
		
		// Format agent name with badge
		if msg.Role == "system" {
			output.WriteString(systemBadgeStyle.Render(" SYSTEM "))
			output.WriteString(systemStyle.Render(msg.Content))
		} else {
			// Use colored badge for agents
			badgeStyle := l.getAgentBadgeStyle(msg.AgentName)
			output.WriteString(badgeStyle.Render(" " + msg.AgentName + " "))
			
			// Add metrics if enabled and available
			if l.showMetrics && msg.Metrics != nil {
				metricsStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("240")).
					Italic(true)
				
				metricsStr := fmt.Sprintf("(%.2fs, %d tokens, $%.6f)",
					msg.Metrics.Duration.Seconds(),
					msg.Metrics.TotalTokens,
					msg.Metrics.Cost)
				
				output.WriteString(" ")
				output.WriteString(metricsStyle.Render(metricsStr))
			}
			
			output.WriteString("\n\n")
			
			// Format and wrap message content with nice indentation
			wrappedContent := l.wrapText(msg.Content, 2)
			agentStyle := l.getAgentColor(msg.AgentName)
			
			// Apply color to each line
			lines := strings.Split(wrappedContent, "\n")
			for _, line := range lines {
				output.WriteString(agentStyle.Render(line))
				output.WriteString("\n")
			}
		}
		
		output.WriteString("\n")
		
		fmt.Fprint(l.console, output.String())
	}
}

func (l *ChatLogger) LogError(agentName string, err error) {
	timestamp := time.Now().Format("15:04:05")
	
	// Write to file
	if l.logFile != nil {
		l.writeToFile(fmt.Sprintf("[%s] ERROR - %s: %v\n", timestamp, agentName, err))
	}
	
	// Write to console
	if l.console != nil {
		output := fmt.Sprintf("%s %s %s: %v\n",
			timestampStyle.Render(fmt.Sprintf("[%s]", timestamp)),
			errorStyle.Render("ERROR"),
			agentName,
			err)
		fmt.Fprint(l.console, output)
	}
}

func (l *ChatLogger) LogSystem(message string) {
	msg := agent.Message{
		AgentID:   "system",
		AgentName: "System",
		Content:   message,
		Timestamp: time.Now().Unix(),
		Role:      "system",
	}
	l.LogMessage(msg)
}

func (l *ChatLogger) wrapText(text string, indent int) string {
	if l.termWidth <= 0 {
		return text
	}
	
	maxWidth := l.termWidth - indent - 2 // Leave some margin
	if maxWidth <= 20 {
		maxWidth = 20 // Minimum width
	}
	
	lines := strings.Split(text, "\n")
	var wrapped []string
	indentStr := strings.Repeat(" ", indent)
	
	for _, line := range lines {
		if len(line) <= maxWidth {
			wrapped = append(wrapped, indentStr+line)
			continue
		}
		
		// Wrap long lines at word boundaries
		words := strings.Fields(line)
		current := indentStr
		
		for _, word := range words {
			if len(current)+len(word)+1 > l.termWidth {
				if len(current) > indent {
					wrapped = append(wrapped, current)
					current = indentStr + word
				} else {
					// Word is too long, break it
					wrapped = append(wrapped, indentStr+word[:maxWidth])
					current = indentStr + word[maxWidth:]
				}
			} else {
				if len(current) > indent {
					current += " "
				}
				current += word
			}
		}
		
		if len(current) > indent {
			wrapped = append(wrapped, current)
		}
	}
	
	return strings.Join(wrapped, "\n")
}

func (l *ChatLogger) writeToFile(content string) {
	if l.logFile != nil {
		l.logFile.WriteString(content)
		l.logFile.Sync()
	}
}

func (l *ChatLogger) Close() {
	if l.logFile != nil {
		l.writeToFile(fmt.Sprintf("\n=== Chat Ended ===\n"))
		l.writeToFile(fmt.Sprintf("Ended: %s\n", time.Now().Format("2006-01-02 15:04:05")))
		l.logFile.Close()
	}
}

// Helper function to get terminal size
func getTerminalSize() (int, int, error) {
	// This is a simplified version - in production you'd use golang.org/x/term
	// For now, return default values
	return 80, 24, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}