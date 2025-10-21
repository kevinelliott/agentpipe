package tui

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
)

// TestModel_Init tests the initialization of the simple TUI model
func TestModel_Init(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Mode:          "round-robin",
			MaxTurns:      10,
			TurnTimeout:   30 * time.Second,
			ResponseDelay: 1 * time.Second,
		},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		agents:   []agent.Agent{},
		messages: make([]agent.Message, 0),
	}

	cmd := m.Init()
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

// TestModel_Update_KeyMsg tests keyboard input handling
func TestModel_Update_KeyMsg(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	tests := []struct {
		name     string
		keyType  tea.KeyType
		keyStr   string
		running  bool
		wantQuit bool
	}{
		{
			name:     "Ctrl+C quits",
			keyType:  tea.KeyCtrlC,
			wantQuit: true,
		},
		{
			name:     "Escape quits",
			keyType:  tea.KeyEsc,
			wantQuit: true,
		},
		{
			name:     "Ctrl+S starts conversation when stopped",
			keyType:  tea.KeyCtrlS,
			running:  false,
			wantQuit: false,
		},
		{
			name:     "Ctrl+P toggles pause",
			keyType:  tea.KeyCtrlP,
			running:  true,
			wantQuit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				ctx:     context.Background(),
				config:  cfg,
				running: tt.running,
				ready:   true,
			}

			msg := tea.KeyMsg{Type: tt.keyType, Runes: []rune(tt.keyStr)}
			_, cmd := m.Update(msg)

			if tt.wantQuit {
				// Quit command should be returned
				if cmd == nil {
					t.Error("Expected quit command but got nil")
				}
			}
		})
	}
}

// TestModel_Update_WindowSize tests window resize handling
func TestModel_Update_WindowSize(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:    context.Background(),
		config: cfg,
		ready:  false,
	}

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 40,
	}

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(Model)

	if updated.width != 100 {
		t.Errorf("Expected width 100, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("Expected height 40, got %d", updated.height)
	}
	if !updated.ready {
		t.Error("Expected model to be ready after window size")
	}
}

// TestModel_Update_MessageUpdate tests message updates
func TestModel_Update_MessageUpdate(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
		ready:    true,
	}

	// Set up viewport size first
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	// Add a message
	testMsg := agent.Message{
		AgentID:   "test-agent",
		AgentName: "TestAgent",
		Content:   "Test message content",
		Timestamp: time.Now().Unix(),
		Role:      "agent",
	}

	update := messageUpdate{message: testMsg}
	updatedModel, _ = m.Update(update)
	updated := updatedModel.(Model)

	if len(updated.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(updated.messages))
	}
	if updated.messages[0].Content != "Test message content" {
		t.Errorf("Expected message content 'Test message content', got %s", updated.messages[0].Content)
	}
}

// TestModel_Update_ConversationDone tests conversation completion
func TestModel_Update_ConversationDone(t *testing.T) {
	m := Model{
		ctx:     context.Background(),
		config:  &config.Config{},
		running: true,
	}

	updatedModel, _ := m.Update(conversationDone{})
	updated := updatedModel.(Model)

	if updated.running {
		t.Error("Expected running to be false after conversationDone")
	}
}

// TestModel_Update_ErrMsg tests error message handling
func TestModel_Update_ErrMsg(t *testing.T) {
	m := Model{
		ctx:     context.Background(),
		config:  &config.Config{},
		running: true,
	}

	testErr := errMsg{err: context.DeadlineExceeded}
	updatedModel, _ := m.Update(testErr)
	updated := updatedModel.(Model)

	if updated.err == nil {
		t.Error("Expected error to be set")
	}
	if updated.running {
		t.Error("Expected running to be false after error")
	}
}

// TestModel_View tests the view rendering
func TestModel_View(t *testing.T) {
	tests := []struct {
		name     string
		ready    bool
		running  bool
		agentCnt int
		want     string
	}{
		{
			name:  "Not ready shows initialization",
			ready: false,
			want:  "Initializing...",
		},
		{
			name:     "Ready shows UI",
			ready:    true,
			running:  true,
			agentCnt: 2,
			want:     "AgentPipe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
			}

			agents := make([]agent.Agent, tt.agentCnt)

			m := Model{
				ctx:     context.Background(),
				config:  cfg,
				agents:  agents,
				ready:   tt.ready,
				running: tt.running,
			}

			// Initialize viewport if ready
			if tt.ready {
				msg := tea.WindowSizeMsg{Width: 100, Height: 40}
				updatedModel, _ := m.Update(msg)
				m = updatedModel.(Model)
			}

			view := m.View()
			if !strings.Contains(view, tt.want) {
				t.Errorf("Expected view to contain %q, got %q", tt.want, view)
			}
		})
	}
}

// TestModel_RenderMessages tests message rendering
func TestModel_RenderMessages(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name     string
		messages []agent.Message
		want     []string
	}{
		{
			name:     "Empty messages",
			messages: []agent.Message{},
			want:     []string{},
		},
		{
			name: "System message",
			messages: []agent.Message{
				{
					AgentID:   "system",
					AgentName: "System",
					Content:   "System message",
					Timestamp: now,
					Role:      "system",
				},
			},
			want: []string{"System", "System message"},
		},
		{
			name: "Agent message",
			messages: []agent.Message{
				{
					AgentID:   "agent-1",
					AgentName: "TestAgent",
					Content:   "Agent response",
					Timestamp: now,
					Role:      "agent",
				},
			},
			want: []string{"TestAgent", "Agent response"},
		},
		{
			name: "Multiple messages",
			messages: []agent.Message{
				{
					AgentID:   "system",
					AgentName: "System",
					Content:   "First message",
					Timestamp: now,
					Role:      "system",
				},
				{
					AgentID:   "agent-1",
					AgentName: "Agent1",
					Content:   "Second message",
					Timestamp: now,
					Role:      "agent",
				},
			},
			want: []string{"System", "First message", "Agent1", "Second message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				messages: tt.messages,
			}

			rendered := m.renderMessages()

			for _, expected := range tt.want {
				if !strings.Contains(rendered, expected) {
					t.Errorf("Expected rendered messages to contain %q, got %q", expected, rendered)
				}
			}
		})
	}
}

// TestTuiWriter tests the tuiWriter implementation
func TestTuiWriter(t *testing.T) {
	w := &tuiWriter{
		messageChan: make(chan agent.Message, 10),
	}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "Write empty",
			input: "",
			want:  0,
		},
		{
			name:  "Write text",
			input: "Hello, World!",
			want:  13,
		},
		{
			name:  "Write with newline",
			input: "Line 1\nLine 2\n",
			want:  14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := w.Write([]byte(tt.input))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if n != tt.want {
				t.Errorf("Expected to write %d bytes, wrote %d", tt.want, n)
			}
		})
	}
}

// TestModel_StartConversation tests conversation startup
func TestModel_StartConversation(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Mode:          "round-robin",
			MaxTurns:      5,
			TurnTimeout:   10 * time.Second,
			ResponseDelay: 1 * time.Second,
			InitialPrompt: "Test prompt",
		},
	}

	m := Model{
		ctx:    context.Background(),
		config: cfg,
		agents: []agent.Agent{},
	}

	cmd := m.startConversation()
	if cmd == nil {
		t.Error("Expected startConversation to return a command")
	}

	// Execute the command and check result
	msg := cmd()
	if msg == nil {
		t.Error("Expected command to return a message")
	}
}

// TestModel_SearchMode tests entering and exiting search mode
func TestModel_SearchMode(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
		ready:    true,
	}

	// Initialize with window size to set up search input
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	// Add some test messages
	m.messages = []agent.Message{
		{AgentName: "Agent1", Content: "Hello world", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent2", Content: "Testing search", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent3", Content: "Another message", Role: "agent", Timestamp: time.Now().Unix()},
	}

	// Test entering search mode with Ctrl+F
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlF}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	if !m.searchMode {
		t.Error("Expected search mode to be enabled after Ctrl+F")
	}

	// Test exiting search mode with Esc
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.searchMode {
		t.Error("Expected search mode to be disabled after Esc")
	}
	if m.searchInput.Value() != "" {
		t.Error("Expected search input to be cleared after Esc")
	}
}

func TestModel_PerformSearch(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:        context.Background(),
		config:     cfg,
		messages:   make([]agent.Message, 0),
		ready:      true,
		searchMode: true,
	}

	// Initialize with window size to set up search input
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)
	m.searchMode = true // Re-enable search mode after window size update

	// Add test messages
	m.messages = []agent.Message{
		{AgentName: "Agent1", Content: "Hello world", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent2", Content: "Testing search", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent3", Content: "Another message", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent1", Content: "Hello again", Role: "agent", Timestamp: time.Now().Unix()},
	}

	tests := []struct {
		name          string
		searchTerm    string
		expectedCount int
		expectedFirst int
	}{
		{
			name:          "Search for 'hello'",
			searchTerm:    "hello",
			expectedCount: 2,
			expectedFirst: 0,
		},
		{
			name:          "Search for 'search'",
			searchTerm:    "search",
			expectedCount: 1,
			expectedFirst: 1,
		},
		{
			name:          "Search for 'Agent1'",
			searchTerm:    "Agent1",
			expectedCount: 2,
			expectedFirst: 0,
		},
		{
			name:          "Search for non-existent term",
			searchTerm:    "xyz123",
			expectedCount: 0,
			expectedFirst: -1,
		},
		{
			name:          "Empty search",
			searchTerm:    "",
			expectedCount: 0,
			expectedFirst: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.searchInput.SetValue(tt.searchTerm)
			m.performSearch()

			if len(m.searchResults) != tt.expectedCount {
				t.Errorf("Expected %d search results, got %d", tt.expectedCount, len(m.searchResults))
			}

			if tt.expectedCount > 0 {
				if m.currentSearchIndex != 0 {
					t.Errorf("Expected currentSearchIndex to be 0, got %d", m.currentSearchIndex)
				}
				if m.searchResults[0] != tt.expectedFirst {
					t.Errorf("Expected first result at index %d, got %d", tt.expectedFirst, m.searchResults[0])
				}
			} else {
				if m.currentSearchIndex != -1 {
					t.Errorf("Expected currentSearchIndex to be -1 when no results, got %d", m.currentSearchIndex)
				}
			}
		})
	}
}

func TestModel_SearchNavigation(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:        context.Background(),
		config:     cfg,
		messages:   make([]agent.Message, 0),
		ready:      true,
		searchMode: true,
	}

	// Initialize with window size to set up search input
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)
	m.searchMode = true // Re-enable search mode after window size update

	// Add test messages with multiple matches
	m.messages = []agent.Message{
		{AgentName: "Agent1", Content: "test message 1", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent2", Content: "test message 2", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentName: "Agent3", Content: "test message 3", Role: "agent", Timestamp: time.Now().Unix()},
	}

	// Perform search
	m.searchInput.SetValue("test")
	m.performSearch()

	if len(m.searchResults) != 3 {
		t.Fatalf("Expected 3 search results, got %d", len(m.searchResults))
	}
	if m.currentSearchIndex != 0 {
		t.Fatalf("Expected initial index 0, got %d", m.currentSearchIndex)
	}

	// Test next navigation (n key)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	if m.currentSearchIndex != 1 {
		t.Errorf("Expected index 1 after 'n', got %d", m.currentSearchIndex)
	}

	// Test next again
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.currentSearchIndex != 2 {
		t.Errorf("Expected index 2 after second 'n', got %d", m.currentSearchIndex)
	}

	// Test wrapping (should go back to 0)
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.currentSearchIndex != 0 {
		t.Errorf("Expected index 0 after wrap, got %d", m.currentSearchIndex)
	}

	// Test previous navigation (N key)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.currentSearchIndex != 2 {
		t.Errorf("Expected index 2 after 'N' from 0 (reverse wrap), got %d", m.currentSearchIndex)
	}
}

func TestModel_CommandMode(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
		ready:    true,
	}

	// Initialize with window size
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	// Test entering command mode with /
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	if !m.commandMode {
		t.Error("Expected command mode to be enabled after /")
	}

	// Test exiting command mode with Esc
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.commandMode {
		t.Error("Expected command mode to be disabled after Esc")
	}
	if m.commandInput.Value() != "" {
		t.Error("Expected command input to be cleared after Esc")
	}
}

func TestModel_ExecuteFilterCommand(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	// Create mock agents
	mockAgent1 := &mockAgent{id: "agent-1", name: "Agent1"}
	mockAgent2 := &mockAgent{id: "agent-2", name: "Agent2"}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		agents:   []agent.Agent{mockAgent1, mockAgent2},
		messages: make([]agent.Message, 0),
		ready:    true,
	}

	// Initialize with window size
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	// Set command to filter by Agent1
	m.commandInput.SetValue("filter Agent1")
	m.executeCommand()

	if m.filterAgent != "Agent1" {
		t.Errorf("Expected filterAgent to be 'Agent1', got '%s'", m.filterAgent)
	}
	if !strings.Contains(m.statusMessage, "Agent1") {
		t.Errorf("Expected status message to contain 'Agent1', got '%s'", m.statusMessage)
	}
}

func TestModel_ExecuteClearCommand(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:         context.Background(),
		config:      cfg,
		messages:    make([]agent.Message, 0),
		ready:       true,
		filterAgent: "Agent1",
	}

	// Initialize with window size
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	// Execute clear command
	m.commandInput.SetValue("clear")
	m.executeCommand()

	if m.filterAgent != "" {
		t.Errorf("Expected filterAgent to be empty, got '%s'", m.filterAgent)
	}
	if !strings.Contains(m.statusMessage, "cleared") {
		t.Errorf("Expected status message to contain 'cleared', got '%s'", m.statusMessage)
	}
}

func TestModel_FilterMessages(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:    context.Background(),
		config: cfg,
		messages: []agent.Message{
			{AgentName: "Agent1", Content: "Message from Agent1", Role: "agent", Timestamp: time.Now().Unix()},
			{AgentName: "Agent2", Content: "Message from Agent2", Role: "agent", Timestamp: time.Now().Unix()},
			{AgentName: "Agent1", Content: "Another from Agent1", Role: "agent", Timestamp: time.Now().Unix()},
			{AgentName: "System", Content: "System message", Role: "system", Timestamp: time.Now().Unix()},
		},
		ready:       true,
		filterAgent: "",
	}

	// Test without filter - should show all messages
	rendered := m.renderMessages()
	if !strings.Contains(rendered, "Agent1") {
		t.Error("Expected rendered messages to contain Agent1")
	}
	if !strings.Contains(rendered, "Agent2") {
		t.Error("Expected rendered messages to contain Agent2")
	}
	if !strings.Contains(rendered, "System") {
		t.Error("Expected rendered messages to contain System")
	}

	// Test with filter - should show only Agent1 and System
	m.filterAgent = "Agent1"
	rendered = m.renderMessages()

	if !strings.Contains(rendered, "Message from Agent1") {
		t.Error("Expected rendered messages to contain Agent1's messages")
	}
	if strings.Contains(rendered, "Message from Agent2") {
		t.Error("Expected rendered messages NOT to contain Agent2's messages")
	}
	if !strings.Contains(rendered, "System message") {
		t.Error("Expected rendered messages to contain System messages even with filter")
	}
}

func TestModel_UnknownCommand(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
		ready:    true,
	}

	// Initialize with window size
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	// Execute unknown command
	m.commandInput.SetValue("unknown arg1 arg2")
	m.executeCommand()

	if !strings.Contains(m.statusMessage, "Unknown command") {
		t.Errorf("Expected status message to contain 'Unknown command', got '%s'", m.statusMessage)
	}
}

// mockAgent is a simple mock implementation for testing
type mockAgent struct {
	id   string
	name string
}

func (m *mockAgent) GetID() string                             { return m.id }
func (m *mockAgent) GetName() string                           { return m.name }
func (m *mockAgent) GetType() string                           { return "mock" }
func (m *mockAgent) GetModel() string                          { return "mock-model" }
func (m *mockAgent) GetRateLimit() float64                     { return 0 }
func (m *mockAgent) GetRateLimitBurst() int                    { return 0 }
func (m *mockAgent) GetCLIVersion() string                     { return "1.0.0" }
func (m *mockAgent) Initialize(config agent.AgentConfig) error { return nil }
func (m *mockAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	return "mock response", nil
}
func (m *mockAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	return nil
}
func (m *mockAgent) Announce() string                      { return m.name + " joined" }
func (m *mockAgent) IsAvailable() bool                     { return true }
func (m *mockAgent) HealthCheck(ctx context.Context) error { return nil }

func TestModel_HelpModal(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
		ready:    true,
	}

	// Initialize with window size
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	// Test showing help modal with ?
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	if !m.showHelp {
		t.Error("Expected showHelp to be true after ?")
	}

	// Test that view shows help content
	view := m.View()
	if !strings.Contains(view, "Keyboard Shortcuts Help") {
		t.Error("Expected view to contain help modal title")
	}
	if !strings.Contains(view, "General Controls") {
		t.Error("Expected view to contain General Controls section")
	}
	if !strings.Contains(view, "Ctrl+F") {
		t.Error("Expected view to contain Ctrl+F keybinding")
	}

	// Test closing help modal with Esc
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.showHelp {
		t.Error("Expected showHelp to be false after Esc")
	}

	// Test toggling help modal again with ?
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if !m.showHelp {
		t.Error("Expected showHelp to be true after second ?")
	}

	// Test closing help modal with ? (toggle)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.showHelp {
		t.Error("Expected showHelp to be false after toggling with ?")
	}
}

func TestModel_HelpModalContent(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
		ready:    true,
		showHelp: true,
	}

	helpContent := m.renderHelp()

	// Verify all sections are present
	expectedSections := []string{
		"Keyboard Shortcuts Help",
		"General Controls",
		"Conversation Controls",
		"Search",
		"Commands (Slash Commands)",
	}

	for _, section := range expectedSections {
		if !strings.Contains(helpContent, section) {
			t.Errorf("Expected help content to contain '%s'", section)
		}
	}

	// Verify key keybindings are documented
	expectedKeys := []string{
		"Ctrl+C",
		"Ctrl+S",
		"Ctrl+P",
		"Ctrl+F",
		"/",
		"?",
		"filter <agent>",
		"clear",
	}

	for _, key := range expectedKeys {
		if !strings.Contains(helpContent, key) {
			t.Errorf("Expected help content to document '%s' key", key)
		}
	}
}

func TestModel_MultiplePanelUpdates(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{Mode: "round-robin"},
	}

	m := Model{
		ctx:      context.Background(),
		config:   cfg,
		messages: make([]agent.Message, 0),
	}

	// Simulate window resize
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	m = updatedModel.(Model)

	if !m.ready {
		t.Error("Expected model to be ready after resize")
	}

	// Add multiple messages
	for i := 0; i < 5; i++ {
		msg := messageUpdate{
			message: agent.Message{
				AgentID:   "agent-1",
				AgentName: "TestAgent",
				Content:   "Test message",
				Timestamp: time.Now().Unix(),
				Role:      "agent",
			},
		}
		updatedModel, _ = m.Update(msg)
		m = updatedModel.(Model)
	}

	if len(m.messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(m.messages))
	}
}
