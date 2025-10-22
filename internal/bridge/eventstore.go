package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// EventStore handles local storage of events for later upload
type EventStore struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
	events   []*Event
}

// NewEventStore creates a new event store that saves events to a JSON file
func NewEventStore(conversationID string, logDir string) (*EventStore, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create event log file path
	filename := fmt.Sprintf("events_%s.jsonl", conversationID)
	filePath := filepath.Join(logDir, filename)

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open event log file: %w", err)
	}

	return &EventStore{
		filePath: filePath,
		file:     file,
		events:   make([]*Event, 0),
	}, nil
}

// SaveEvent saves an event to the local store (JSON Lines format)
func (s *EventStore) SaveEvent(event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Append to in-memory list
	s.events = append(s.events, event)

	// Write to file in JSON Lines format (one JSON object per line)
	encoder := json.NewEncoder(s.file)
	if err := encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to write event to file: %w", err)
	}

	return nil
}

// GetEvents returns all events stored in memory
func (s *EventStore) GetEvents() []*Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return a copy to prevent external modification
	eventsCopy := make([]*Event, len(s.events))
	copy(eventsCopy, s.events)
	return eventsCopy
}

// GetFilePath returns the path to the event log file
func (s *EventStore) GetFilePath() string {
	return s.filePath
}

// Close closes the event log file
func (s *EventStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// LoadEventsFromFile reads events from a JSON Lines file
func LoadEventsFromFile(filePath string) ([]*Event, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var events []*Event
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}
