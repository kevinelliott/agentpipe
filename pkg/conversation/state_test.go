package conversation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
)

// TestNewState tests creating a new state
func TestNewState(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Agents = []agent.AgentConfig{
		{ID: "test-1", Type: "claude", Name: "Claude"},
	}

	messages := []agent.Message{
		{AgentID: "test-1", AgentName: "Claude", Content: "Hello", Role: "agent", Timestamp: time.Now().Unix()},
	}

	startedAt := time.Now().Add(-5 * time.Minute)
	state := NewState(messages, cfg, startedAt)

	if state == nil {
		t.Fatal("State should not be nil")
	}

	if state.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", state.Version)
	}

	if len(state.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(state.Messages))
	}

	if state.Config == nil {
		t.Error("Config should not be nil")
	}

	if state.Metadata.TotalMessages != 1 {
		t.Errorf("Expected 1 total message, got %d", state.Metadata.TotalMessages)
	}
}

// TestState_Save tests saving state to file
func TestState_Save(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test-state.json")

	cfg := config.NewDefaultConfig()
	cfg.Agents = []agent.AgentConfig{
		{ID: "test-1", Type: "claude", Name: "Claude"},
	}

	messages := []agent.Message{
		{AgentID: "test-1", AgentName: "Claude", Content: "Test message", Role: "agent", Timestamp: time.Now().Unix()},
	}

	state := NewState(messages, cfg, time.Now())

	// Save state
	if err := state.Save(statePath); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file was not created")
	}

	// Verify file permissions
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Failed to stat state file: %v", err)
	}

	// Check permissions are 0600
	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", mode.Perm())
	}
}

// TestState_Save_CreatesDirectory tests that Save creates parent directory
func TestState_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nested", "dir", "state.json")

	cfg := config.NewDefaultConfig()
	messages := []agent.Message{
		{AgentID: "test-1", AgentName: "Claude", Content: "Test", Role: "agent", Timestamp: time.Now().Unix()},
	}

	state := NewState(messages, cfg, time.Now())

	// Save state (should create nested directories)
	if err := state.Save(statePath); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file was not created")
	}
}

// TestLoadState tests loading state from file
func TestLoadState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test-state.json")

	// Create and save state
	cfg := config.NewDefaultConfig()
	cfg.Agents = []agent.AgentConfig{
		{ID: "test-1", Type: "claude", Name: "Claude"},
		{ID: "test-2", Type: "gemini", Name: "Gemini"},
	}

	messages := []agent.Message{
		{AgentID: "test-1", AgentName: "Claude", Content: "Message 1", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentID: "test-2", AgentName: "Gemini", Content: "Message 2", Role: "agent", Timestamp: time.Now().Unix()},
	}

	startedAt := time.Now().Add(-10 * time.Minute)
	originalState := NewState(messages, cfg, startedAt)
	originalState.Metadata.Description = "Test conversation"

	if err := originalState.Save(statePath); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load state
	loadedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify loaded state matches original
	if loadedState.Version != originalState.Version {
		t.Errorf("Version mismatch: expected %s, got %s", originalState.Version, loadedState.Version)
	}

	if len(loadedState.Messages) != len(originalState.Messages) {
		t.Errorf("Message count mismatch: expected %d, got %d", len(originalState.Messages), len(loadedState.Messages))
	}

	if loadedState.Config == nil {
		t.Error("Loaded config should not be nil")
	}

	if len(loadedState.Config.Agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(loadedState.Config.Agents))
	}

	if loadedState.Metadata.Description != "Test conversation" {
		t.Errorf("Description mismatch: expected 'Test conversation', got '%s'", loadedState.Metadata.Description)
	}

	if loadedState.Metadata.TotalMessages != 2 {
		t.Errorf("Expected 2 total messages, got %d", loadedState.Metadata.TotalMessages)
	}
}

// TestLoadState_NonexistentFile tests error handling for missing file
func TestLoadState_NonexistentFile(t *testing.T) {
	_, err := LoadState("/nonexistent/path/state.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// TestLoadState_InvalidJSON tests error handling for invalid JSON
func TestLoadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(statePath, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	_, err := LoadState(statePath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestGenerateStateFileName tests filename generation
func TestGenerateStateFileName(t *testing.T) {
	filename := GenerateStateFileName()

	if filename == "" {
		t.Error("Filename should not be empty")
	}

	// Check format: conversation-YYYYMMDD-HHMMSS.json
	if filepath.Ext(filename) != ".json" {
		t.Errorf("Expected .json extension, got %s", filepath.Ext(filename))
	}

	if len(filename) != len("conversation-20060102-150405.json") {
		t.Errorf("Unexpected filename length: %s", filename)
	}
}

// TestListStates tests listing saved states
func TestListStates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some state files
	for i := 0; i < 3; i++ {
		statePath := filepath.Join(tmpDir, GenerateStateFileName())
		state := NewState(
			[]agent.Message{{AgentID: "test", AgentName: "Test", Content: "Test", Role: "agent", Timestamp: time.Now().Unix()}},
			config.NewDefaultConfig(),
			time.Now(),
		)
		if err := state.Save(statePath); err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}
		time.Sleep(1 * time.Second) // Ensure unique filenames
	}

	// Create a non-JSON file (should be ignored)
	if err := os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to write non-JSON file: %v", err)
	}

	// List states
	states, err := ListStates(tmpDir)
	if err != nil {
		t.Fatalf("Failed to list states: %v", err)
	}

	if len(states) != 3 {
		t.Errorf("Expected 3 states, got %d", len(states))
	}
}

// TestListStates_EmptyDirectory tests listing from empty directory
func TestListStates_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	states, err := ListStates(tmpDir)
	if err != nil {
		t.Fatalf("Failed to list states: %v", err)
	}

	if len(states) != 0 {
		t.Errorf("Expected 0 states, got %d", len(states))
	}
}

// TestListStates_NonexistentDirectory tests listing from nonexistent directory
func TestListStates_NonexistentDirectory(t *testing.T) {
	states, err := ListStates("/nonexistent/directory")
	if err != nil {
		t.Fatalf("Should not error on nonexistent directory: %v", err)
	}

	if len(states) != 0 {
		t.Errorf("Expected 0 states, got %d", len(states))
	}
}

// TestGetStateInfo tests getting state info
func TestGetStateInfo(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test-state.json")

	cfg := config.NewDefaultConfig()
	cfg.Orchestrator.Mode = "round-robin"
	cfg.Agents = []agent.AgentConfig{
		{ID: "test-1", Type: "claude", Name: "Claude"},
		{ID: "test-2", Type: "gemini", Name: "Gemini"},
	}

	messages := []agent.Message{
		{AgentID: "test-1", AgentName: "Claude", Content: "Msg 1", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentID: "test-2", AgentName: "Gemini", Content: "Msg 2", Role: "agent", Timestamp: time.Now().Unix()},
		{AgentID: "test-1", AgentName: "Claude", Content: "Msg 3", Role: "agent", Timestamp: time.Now().Unix()},
	}

	startedAt := time.Now().Add(-15 * time.Minute)
	state := NewState(messages, cfg, startedAt)
	state.Metadata.Description = "Test info"

	if err := state.Save(statePath); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Get state info
	info, err := GetStateInfo(statePath)
	if err != nil {
		t.Fatalf("Failed to get state info: %v", err)
	}

	if info.Path != statePath {
		t.Errorf("Path mismatch: expected %s, got %s", statePath, info.Path)
	}

	if info.Messages != 3 {
		t.Errorf("Expected 3 messages, got %d", info.Messages)
	}

	if info.AgentCount != 2 {
		t.Errorf("Expected 2 agents, got %d", info.AgentCount)
	}

	if info.Mode != "round-robin" {
		t.Errorf("Expected mode round-robin, got %s", info.Mode)
	}

	if info.Description != "Test info" {
		t.Errorf("Description mismatch: expected 'Test info', got '%s'", info.Description)
	}
}

// TestGetDefaultStateDir tests default state directory
func TestGetDefaultStateDir(t *testing.T) {
	dir, err := GetDefaultStateDir()
	if err != nil {
		t.Fatalf("Failed to get default state dir: %v", err)
	}

	if dir == "" {
		t.Error("Default state dir should not be empty")
	}

	// Should end with .agentpipe/states
	if !filepath.IsAbs(dir) {
		t.Error("Default state dir should be absolute path")
	}

	if filepath.Base(dir) != "states" {
		t.Errorf("Expected dir to end with 'states', got %s", filepath.Base(dir))
	}
}

// TestState_RoundTrip tests saving and loading preserves data
func TestState_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "roundtrip.json")

	// Create complex state
	cfg := config.NewDefaultConfig()
	cfg.Orchestrator.Mode = "reactive"
	cfg.Orchestrator.MaxTurns = 50
	cfg.Agents = []agent.AgentConfig{
		{ID: "a1", Type: "claude", Name: "Claude", Prompt: "You are Claude"},
		{ID: "a2", Type: "gemini", Name: "Gemini", Prompt: "You are Gemini"},
		{ID: "a3", Type: "qwen", Name: "Qwen", Prompt: "You are Qwen"},
	}

	messages := make([]agent.Message, 0)
	for i := 0; i < 10; i++ {
		messages = append(messages, agent.Message{
			AgentID:   "a1",
			AgentName: "Claude",
			Content:   "Message from Claude",
			Role:      "agent",
			Timestamp: time.Now().Unix(),
		})
		messages = append(messages, agent.Message{
			AgentID:   "a2",
			AgentName: "Gemini",
			Content:   "Message from Gemini",
			Role:      "agent",
			Timestamp: time.Now().Unix(),
		})
	}

	startedAt := time.Now().Add(-30 * time.Minute)
	originalState := NewState(messages, cfg, startedAt)
	originalState.Metadata.Description = "Complex conversation"

	// Save
	if err := originalState.Save(statePath); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load
	loadedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify
	if len(loadedState.Messages) != len(originalState.Messages) {
		t.Errorf("Message count mismatch: %d vs %d", len(loadedState.Messages), len(originalState.Messages))
	}

	if len(loadedState.Config.Agents) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(loadedState.Config.Agents))
	}

	if loadedState.Config.Orchestrator.Mode != "reactive" {
		t.Errorf("Mode mismatch: expected reactive, got %s", loadedState.Config.Orchestrator.Mode)
	}

	if loadedState.Config.Orchestrator.MaxTurns != 50 {
		t.Errorf("MaxTurns mismatch: expected 50, got %d", loadedState.Config.Orchestrator.MaxTurns)
	}
}
