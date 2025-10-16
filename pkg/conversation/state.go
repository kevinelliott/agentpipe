// Package conversation provides conversation state management for AgentPipe.
// It enables saving and resuming conversations across sessions.
package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

// State represents a saved conversation state.
// It contains all information needed to resume a conversation.
type State struct {
	// Version is the state file format version
	Version string `json:"version"`

	// SavedAt is when the state was saved
	SavedAt time.Time `json:"saved_at"`

	// Messages is the conversation history
	Messages []agent.Message `json:"messages"`

	// Config is the configuration used for this conversation
	Config *config.Config `json:"config"`

	// Metadata contains additional information about the conversation
	Metadata StateMetadata `json:"metadata"`
}

// StateMetadata contains metadata about a saved conversation state.
type StateMetadata struct {
	// TotalTurns is the number of conversation turns completed
	TotalTurns int `json:"total_turns"`

	// TotalMessages is the total number of messages
	TotalMessages int `json:"total_messages"`

	// TotalDuration is the total conversation duration in milliseconds
	TotalDuration int64 `json:"total_duration_ms"`

	// StartedAt is when the conversation was started
	StartedAt time.Time `json:"started_at"`

	// Description is an optional description of the conversation
	Description string `json:"description,omitempty"`
}

// NewState creates a new conversation state.
func NewState(messages []agent.Message, cfg *config.Config, startedAt time.Time) *State {
	return &State{
		Version:  "1.0",
		SavedAt:  time.Now(),
		Messages: messages,
		Config:   cfg,
		Metadata: StateMetadata{
			TotalTurns:    len(messages),
			TotalMessages: len(messages),
			StartedAt:     startedAt,
			TotalDuration: int64(time.Since(startedAt).Milliseconds()),
		},
	}
}

// Save writes the conversation state to a file.
// The file is created with 0600 permissions (read/write for owner only).
func (s *State) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.WithError(err).WithField("directory", dir).Error("failed to create state directory")
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.WithError(err).Error("failed to marshal conversation state")
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.WithError(err).WithField("path", path).Error("failed to write state file")
		return fmt.Errorf("failed to write state file: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"path":          path,
		"messages":      len(s.Messages),
		"total_turns":   s.Metadata.TotalTurns,
		"file_size":     len(data),
	}).Info("conversation state saved")

	return nil
}

// LoadState loads a conversation state from a file.
func LoadState(path string) (*State, error) {
	log.WithField("path", path).Debug("loading conversation state")

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		log.WithError(err).WithField("path", path).Error("failed to read state file")
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		log.WithError(err).WithField("path", path).Error("failed to parse state file")
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"path":         path,
		"version":      state.Version,
		"messages":     len(state.Messages),
		"saved_at":     state.SavedAt,
		"started_at":   state.Metadata.StartedAt,
		"total_turns":  state.Metadata.TotalTurns,
	}).Info("conversation state loaded")

	return &state, nil
}

// GetDefaultStateDir returns the default directory for saving conversation states.
// This is ~/.agentpipe/states by default.
func GetDefaultStateDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".agentpipe", "states"), nil
}

// GenerateStateFileName generates a filename for a conversation state.
// Format: conversation-YYYYMMDD-HHMMSS.json
func GenerateStateFileName() string {
	return fmt.Sprintf("conversation-%s.json", time.Now().Format("20060102-150405"))
}

// ListStates lists all saved conversation states in a directory.
func ListStates(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	states := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			states = append(states, filepath.Join(dir, entry.Name()))
		}
	}

	return states, nil
}

// StateInfo contains summary information about a saved state.
type StateInfo struct {
	Path          string
	SavedAt       time.Time
	StartedAt     time.Time
	Messages      int
	Turns         int
	Description   string
	AgentCount    int
	Mode          string
}

// GetStateInfo reads summary information from a state file without loading full state.
func GetStateInfo(path string) (*StateInfo, error) {
	state, err := LoadState(path)
	if err != nil {
		return nil, err
	}

	agentCount := 0
	if state.Config != nil {
		agentCount = len(state.Config.Agents)
	}

	mode := ""
	if state.Config != nil {
		mode = state.Config.Orchestrator.Mode
	}

	return &StateInfo{
		Path:        path,
		SavedAt:     state.SavedAt,
		StartedAt:   state.Metadata.StartedAt,
		Messages:    len(state.Messages),
		Turns:       state.Metadata.TotalTurns,
		Description: state.Metadata.Description,
		AgentCount:  agentCount,
		Mode:        mode,
	}, nil
}
