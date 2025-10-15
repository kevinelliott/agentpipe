package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

func TestNewChatLoggerWithoutLogDir(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewChatLogger("", "text", &buf, false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be created")
	}
	if logger.logFile != nil {
		t.Error("expected no log file when logDir is empty")
	}
	if logger.console == nil {
		t.Error("expected console writer to be set")
	}
}

func TestNewChatLoggerWithLogDir(t *testing.T) {
	tempDir := t.TempDir()
	var buf bytes.Buffer

	logger, err := NewChatLogger(tempDir, "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be created")
	}
	defer logger.Close()

	if logger.logFile == nil {
		t.Error("expected log file to be created")
	}

	// Check that log file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 log file, got %d", len(files))
	}

	// Check console output contains log path
	output := buf.String()
	if !strings.Contains(output, "Chat logged to") {
		t.Error("expected console output to contain log file path")
	}
}

func TestNewChatLoggerJSONFormat(t *testing.T) {
	tempDir := t.TempDir()
	var buf bytes.Buffer

	logger, err := NewChatLogger(tempDir, "json", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	if logger.logFormat != "json" {
		t.Errorf("expected json format, got %s", logger.logFormat)
	}
}

func TestLogMessageToFile(t *testing.T) {
	tempDir := t.TempDir()
	var buf bytes.Buffer

	logger, err := NewChatLogger(tempDir, "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	msg := agent.Message{
		AgentID:   "test-agent",
		AgentName: "TestAgent",
		Content:   "Hello, world!",
		Timestamp: time.Now().Unix(),
		Role:      "agent",
	}

	logger.LogMessage(msg)

	// Read log file
	files, _ := os.ReadDir(tempDir)
	logPath := filepath.Join(tempDir, files[0].Name())
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "TestAgent") {
		t.Error("expected log to contain agent name")
	}
	if !strings.Contains(logContent, "Hello, world!") {
		t.Error("expected log to contain message content")
	}
}

func TestLogMessageToFileJSON(t *testing.T) {
	tempDir := t.TempDir()
	var buf bytes.Buffer

	logger, err := NewChatLogger(tempDir, "json", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	msg := agent.Message{
		AgentID:   "test-agent",
		AgentName: "TestAgent",
		Content:   "Test message",
		Timestamp: time.Now().Unix(),
		Role:      "agent",
	}

	logger.LogMessage(msg)

	// Read log file
	files, _ := os.ReadDir(tempDir)
	logPath := filepath.Join(tempDir, files[0].Name())
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Find the JSON line (skip header lines)
	lines := strings.Split(string(content), "\n")
	var jsonLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "{") {
			jsonLine = line
			break
		}
	}

	if jsonLine == "" {
		t.Fatal("expected JSON message in log file")
	}

	// Parse JSON
	var parsed agent.Message
	if err := json.Unmarshal([]byte(jsonLine), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.AgentName != "TestAgent" {
		t.Errorf("expected agent name 'TestAgent', got '%s'", parsed.AgentName)
	}
	if parsed.Content != "Test message" {
		t.Errorf("expected content 'Test message', got '%s'", parsed.Content)
	}
}

func TestLogMessageToConsole(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewChatLogger("", "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := agent.Message{
		AgentID:   "test-agent",
		AgentName: "TestAgent",
		Content:   "Console message",
		Timestamp: time.Now().Unix(),
		Role:      "agent",
	}

	logger.LogMessage(msg)

	output := buf.String()
	if !strings.Contains(output, "TestAgent") {
		t.Error("expected console output to contain agent name")
	}
	if !strings.Contains(output, "Console message") {
		t.Error("expected console output to contain message content")
	}
}

func TestLogMessageWithMetrics(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewChatLogger("", "text", &buf, true) // showMetrics = true
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := agent.Message{
		AgentID:   "test-agent",
		AgentName: "TestAgent",
		Content:   "Message with metrics",
		Timestamp: time.Now().Unix(),
		Role:      "agent",
		Metrics: &agent.ResponseMetrics{
			Duration:     500 * time.Millisecond,
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
			Cost:         0.001234,
		},
	}

	logger.LogMessage(msg)

	output := buf.String()
	if !strings.Contains(output, "30 tokens") {
		t.Error("expected metrics to show token count")
	}
	if !strings.Contains(output, "0.5") || !strings.Contains(output, "s") {
		t.Error("expected metrics to show duration")
	}
	if !strings.Contains(output, "0.001234") {
		t.Error("expected metrics to show cost")
	}
}

func TestLogMessageSystemRole(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewChatLogger("", "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := agent.Message{
		AgentID:   "system",
		AgentName: "System",
		Content:   "System message",
		Timestamp: time.Now().Unix(),
		Role:      "system",
	}

	logger.LogMessage(msg)

	output := buf.String()
	if !strings.Contains(output, "SYSTEM") {
		t.Error("expected system badge in output")
	}
	if !strings.Contains(output, "System message") {
		t.Error("expected system message in output")
	}
}

func TestLogError(t *testing.T) {
	tempDir := t.TempDir()
	var buf bytes.Buffer

	logger, err := NewChatLogger(tempDir, "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	testErr := errors.New("test error")
	logger.LogError("TestAgent", testErr)

	// Check console output
	consoleOutput := buf.String()
	if !strings.Contains(consoleOutput, "ERROR") {
		t.Error("expected ERROR in console output")
	}
	if !strings.Contains(consoleOutput, "TestAgent") {
		t.Error("expected agent name in console output")
	}
	if !strings.Contains(consoleOutput, "test error") {
		t.Error("expected error message in console output")
	}

	// Check file output
	files, _ := os.ReadDir(tempDir)
	logPath := filepath.Join(tempDir, files[0].Name())
	content, _ := os.ReadFile(logPath)
	fileOutput := string(content)

	if !strings.Contains(fileOutput, "ERROR") {
		t.Error("expected ERROR in file output")
	}
	if !strings.Contains(fileOutput, "TestAgent") {
		t.Error("expected agent name in file output")
	}
}

func TestLogSystem(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewChatLogger("", "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logger.LogSystem("Test system message")

	output := buf.String()
	if !strings.Contains(output, "SYSTEM") {
		t.Error("expected SYSTEM badge")
	}
	if !strings.Contains(output, "Test system message") {
		t.Error("expected system message content")
	}
}

func TestGetAgentColor(t *testing.T) {
	logger := &ChatLogger{
		agentColors: make(map[string]lipgloss.Style),
		colorIndex:  0,
	}

	// First call should assign a color
	style1 := logger.getAgentColor("Agent1")
	if style1.GetForeground() == lipgloss.Color("") {
		t.Error("expected color to be assigned")
	}

	// Second call for same agent should return cached color
	style2 := logger.getAgentColor("Agent1")
	if style1.GetForeground() != style2.GetForeground() {
		t.Error("expected same color for same agent")
	}

	// Different agent should get different color
	style3 := logger.getAgentColor("Agent2")
	if style1.GetForeground() == style3.GetForeground() {
		t.Error("expected different colors for different agents")
	}

	// Check that color index increased
	if logger.colorIndex != 2 {
		t.Errorf("expected colorIndex to be 2, got %d", logger.colorIndex)
	}
}

func TestGetAgentBadgeStyle(t *testing.T) {
	logger := &ChatLogger{
		agentColors: make(map[string]lipgloss.Style),
		colorIndex:  0,
	}

	// Assign a color first
	logger.getAgentColor("Agent1")

	// Get badge style
	badgeStyle := logger.getAgentBadgeStyle("Agent1")
	bg := badgeStyle.GetBackground()

	if bg == lipgloss.Color("") {
		t.Error("expected background color to be set")
	}

	// Test default badge for unknown agent
	defaultBadge := logger.getAgentBadgeStyle("UnknownAgent")
	if defaultBadge.GetBackground() == lipgloss.Color("") {
		t.Error("expected default background color")
	}
}

func TestWrapText(t *testing.T) {
	logger := &ChatLogger{
		termWidth: 40,
	}

	tests := []struct {
		name     string
		input    string
		indent   int
		wantWrap bool
	}{
		{
			name:     "short text",
			input:    "Hello",
			indent:   2,
			wantWrap: false,
		},
		{
			name:     "long text",
			input:    "This is a very long line that should be wrapped to fit within the terminal width",
			indent:   2,
			wantWrap: true,
		},
		{
			name:     "multiline text",
			input:    "Line 1\nLine 2\nLine 3",
			indent:   2,
			wantWrap: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.wrapText(tt.input, tt.indent)

			// Check indentation
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				if len(line) > 0 && !strings.HasPrefix(line, "  ") {
					t.Errorf("expected line to be indented: %q", line)
				}
			}

			// Check wrapping occurred if expected
			if tt.wantWrap {
				if len(lines) <= strings.Count(tt.input, "\n") {
					t.Error("expected text to be wrapped into more lines")
				}
			}
		})
	}
}

func TestWrapTextZeroWidth(t *testing.T) {
	logger := &ChatLogger{
		termWidth: 0, // Zero width should return original text
	}

	input := "This is a test"
	result := logger.wrapText(input, 2)

	if result != input {
		t.Errorf("expected original text when termWidth is 0, got: %q", result)
	}
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	var buf bytes.Buffer

	logger, err := NewChatLogger(tempDir, "text", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logger.Close()

	// Check that file was closed properly by reading it
	files, _ := os.ReadDir(tempDir)
	logPath := filepath.Join(tempDir, files[0].Name())
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file after close: %v", err)
	}

	fileContent := string(content)
	if !strings.Contains(fileContent, "Chat Ended") {
		t.Error("expected end marker in log file")
	}
}

func TestColorCycling(t *testing.T) {
	logger := &ChatLogger{
		agentColors: make(map[string]lipgloss.Style),
		colorIndex:  0,
	}

	// Create more agents than available colors
	numAgents := len(colors) + 3
	colorsSeen := make(map[string]int)

	for i := 0; i < numAgents; i++ {
		agentName := "Agent" + string(rune('A'+i))
		style := logger.getAgentColor(agentName)
		color := style.GetForeground()
		// Convert TerminalColor to string using fmt
		colorStr := fmt.Sprint(color)
		colorsSeen[colorStr]++
	}

	// Should cycle through colors
	if logger.colorIndex != numAgents {
		t.Errorf("expected colorIndex %d, got %d", numAgents, logger.colorIndex)
	}

	// Some colors should be reused
	reusedColors := 0
	for _, count := range colorsSeen {
		if count > 1 {
			reusedColors++
		}
	}

	if reusedColors == 0 {
		t.Error("expected some colors to be reused when cycling")
	}
}

func TestLoggerWithNilConsole(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewChatLogger(tempDir, "text", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Should not panic with nil console
	msg := agent.Message{
		AgentID:   "test",
		AgentName: "Test",
		Content:   "Test message",
		Timestamp: time.Now().Unix(),
		Role:      "agent",
	}

	logger.LogMessage(msg)
	logger.LogError("Test", errors.New("test error"))
	logger.LogSystem("System message")

	// If we get here without panicking, test passes
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a    int
		b    int
		want int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{0, 0, 0},
		{-5, 5, -5},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
