package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
)

func TestExportJSON(t *testing.T) {
	messages := createTestMessages()

	exporter := NewExporter(ExportOptions{
		Format:         FormatJSON,
		IncludeMetrics: true,
		Title:          "Test Conversation",
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify structure
	if result["title"] != "Test Conversation" {
		t.Errorf("Expected title 'Test Conversation', got %v", result["title"])
	}

	messagesArray, ok := result["messages"].([]interface{})
	if !ok {
		t.Fatal("messages field is not an array")
	}

	if len(messagesArray) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messagesArray))
	}

	// Verify summary
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("summary field is missing or invalid")
	}

	if summary["total_messages"] != float64(3) {
		t.Errorf("Expected 3 total messages in summary, got %v", summary["total_messages"])
	}
}

func TestExportMarkdown(t *testing.T) {
	messages := createTestMessages()

	exporter := NewExporter(ExportOptions{
		Format:            FormatMarkdown,
		IncludeMetrics:    true,
		IncludeTimestamps: true,
		Title:             "Test Conversation",
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Verify title
	if !strings.Contains(output, "# Test Conversation") {
		t.Error("Expected markdown to contain title")
	}

	// Verify summary section
	if !strings.Contains(output, "## Summary") {
		t.Error("Expected markdown to contain summary section")
	}

	if !strings.Contains(output, "**Messages**: 3") {
		t.Error("Expected summary to show 3 messages")
	}

	// Verify conversation section
	if !strings.Contains(output, "## Conversation") {
		t.Error("Expected markdown to contain conversation section")
	}

	// Verify agent names
	if !strings.Contains(output, "### Agent1") {
		t.Error("Expected markdown to contain Agent1")
	}

	if !strings.Contains(output, "### [SYSTEM]") {
		t.Error("Expected markdown to contain system messages")
	}

	// Verify metrics
	if !strings.Contains(output, "Duration:") {
		t.Error("Expected markdown to contain duration metrics")
	}

	if !strings.Contains(output, "Tokens:") {
		t.Error("Expected markdown to contain token metrics")
	}
}

func TestExportHTML(t *testing.T) {
	messages := createTestMessages()

	exporter := NewExporter(ExportOptions{
		Format:            FormatHTML,
		IncludeMetrics:    true,
		IncludeTimestamps: true,
		Title:             "Test Conversation",
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Verify HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Expected valid HTML document")
	}

	if !strings.Contains(output, "<html lang=\"en\">") {
		t.Error("Expected HTML tag")
	}

	if !strings.Contains(output, "<title>Test Conversation</title>") {
		t.Error("Expected HTML title")
	}

	// Verify CSS
	if !strings.Contains(output, "<style>") {
		t.Error("Expected CSS styles")
	}

	// Verify content
	if !strings.Contains(output, "<h1>Test Conversation</h1>") {
		t.Error("Expected h1 title")
	}

	// Verify summary
	if !strings.Contains(output, "<div class=\"summary\">") {
		t.Error("Expected summary div")
	}

	// Verify messages
	if !strings.Contains(output, "<div class=\"message") {
		t.Error("Expected message divs")
	}

	// Verify HTML escaping
	if !strings.Contains(output, "Test message from Agent1") {
		t.Error("Expected escaped content")
	}
}

func TestExportWithoutMetrics(t *testing.T) {
	messages := createTestMessages()

	exporter := NewExporter(ExportOptions{
		Format:         FormatJSON,
		IncludeMetrics: false,
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify no summary when metrics disabled
	if _, ok := result["summary"]; ok {
		t.Error("Expected no summary when IncludeMetrics is false")
	}
}

func TestExportWithoutTimestamps(t *testing.T) {
	messages := createTestMessages()

	exporter := NewExporter(ExportOptions{
		Format:            FormatMarkdown,
		IncludeTimestamps: false,
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Timestamps should not be in format "15:04:05"
	// This is a simple check - in real output there might be dates
	lines := strings.Split(output, "\n")
	hasTimestampFormat := false
	for _, line := range lines {
		if strings.Contains(line, "###") && strings.Contains(line, " - ") {
			// This pattern would appear with timestamps: "### Agent1 - 12:34:56"
			hasTimestampFormat = true
			break
		}
	}

	if hasTimestampFormat {
		t.Error("Expected no timestamps in output when IncludeTimestamps is false")
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	messages := createTestMessages()

	exporter := NewExporter(ExportOptions{
		Format: "invalid",
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)

	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported export format") {
		t.Errorf("Expected 'unsupported export format' error, got: %v", err)
	}
}

func TestCalculateSummary(t *testing.T) {
	messages := createTestMessages()
	summary := calculateSummary(messages)

	if summary.TotalMessages != 3 {
		t.Errorf("Expected 3 total messages, got %d", summary.TotalMessages)
	}

	// 3 unique agents: system, agent-1, agent-2
	if summary.UniqueAgents != 3 {
		t.Errorf("Expected 3 unique agents, got %d", summary.UniqueAgents)
	}

	if summary.TotalTokens != 300 {
		t.Errorf("Expected 300 total tokens, got %d", summary.TotalTokens)
	}

	expectedCost := 0.0030
	if summary.TotalCost != expectedCost {
		t.Errorf("Expected cost %.4f, got %.4f", expectedCost, summary.TotalCost)
	}
}

func TestExportEmptyMessages(t *testing.T) {
	messages := []agent.Message{}

	exporter := NewExporter(ExportOptions{
		Format: FormatJSON,
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	messagesArray, ok := result["messages"].([]interface{})
	if !ok {
		t.Fatal("messages field is not an array")
	}

	if len(messagesArray) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messagesArray))
	}
}

func TestHTMLSpecialCharacters(t *testing.T) {
	messages := []agent.Message{
		{
			AgentID:   "agent-1",
			AgentName: "Agent<script>alert('xss')</script>",
			Content:   "Message with <html> & special chars",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
		},
	}

	exporter := NewExporter(ExportOptions{
		Format: FormatHTML,
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Verify HTML escaping
	if strings.Contains(output, "<script>") {
		t.Error("HTML not properly escaped - XSS vulnerability")
	}

	if !strings.Contains(output, "&lt;html&gt;") {
		t.Error("Expected HTML entities for <html>")
	}

	if !strings.Contains(output, "&amp;") {
		t.Error("Expected HTML entity for &")
	}
}

func TestMarkdownMultipleAgents(t *testing.T) {
	messages := []agent.Message{
		{
			AgentID:   "agent-1",
			AgentName: "Alice",
			Content:   "Hello from Alice",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
		},
		{
			AgentID:   "agent-2",
			AgentName: "Bob",
			Content:   "Hello from Bob",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
		},
		{
			AgentID:   "agent-3",
			AgentName: "Charlie",
			Content:   "Hello from Charlie",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
		},
	}

	exporter := NewExporter(ExportOptions{
		Format:         FormatMarkdown,
		IncludeMetrics: true,
	})

	var buf bytes.Buffer
	err := exporter.Export(messages, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Verify all agents present
	if !strings.Contains(output, "### Alice") {
		t.Error("Expected Alice in output")
	}
	if !strings.Contains(output, "### Bob") {
		t.Error("Expected Bob in output")
	}
	if !strings.Contains(output, "### Charlie") {
		t.Error("Expected Charlie in output")
	}

	// Verify unique agents count
	if !strings.Contains(output, "**Agents**: 3") {
		t.Error("Expected 3 unique agents in summary")
	}
}

// Helper function to create test messages
func createTestMessages() []agent.Message {
	return []agent.Message{
		{
			AgentID:   "system",
			AgentName: "System",
			Content:   "Conversation started",
			Timestamp: time.Now().Unix(),
			Role:      "system",
		},
		{
			AgentID:   "agent-1",
			AgentName: "Agent1",
			Content:   "Test message from Agent1",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
			Metrics: &agent.ResponseMetrics{
				Duration:     100 * time.Millisecond,
				InputTokens:  50,
				OutputTokens: 50,
				TotalTokens:  100,
				Model:        "test-model",
				Cost:         0.0010,
			},
		},
		{
			AgentID:   "agent-2",
			AgentName: "Agent2",
			Content:   "Test message from Agent2",
			Timestamp: time.Now().Unix(),
			Role:      "agent",
			Metrics: &agent.ResponseMetrics{
				Duration:     150 * time.Millisecond,
				InputTokens:  100,
				OutputTokens: 100,
				TotalTokens:  200,
				Model:        "test-model",
				Cost:         0.0020,
			},
		},
	}
}
