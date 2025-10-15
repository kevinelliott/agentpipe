package tui

import (
	"context"
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

// TestModel_MultiplePanelUpdates tests sequential updates
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
