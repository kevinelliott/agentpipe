package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestNewConfigWatcher tests creating a config watcher
func TestNewConfigWatcher(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 5
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.StopWatching()

	config := watcher.GetConfig()
	if config == nil {
		t.Fatal("Config should not be nil")
	}

	if len(config.Agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(config.Agents))
	}

	if config.Orchestrator.Mode != "round-robin" {
		t.Errorf("Expected mode round-robin, got %s", config.Orchestrator.Mode)
	}
}

// TestNewConfigWatcher_InvalidFile tests error handling
func TestNewConfigWatcher_InvalidFile(t *testing.T) {
	_, err := NewConfigWatcher("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// TestConfigWatcher_GetConfig tests thread-safe config retrieval
func TestConfigWatcher_GetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.StopWatching()

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config := watcher.GetConfig()
			if config == nil {
				t.Error("Config should not be nil")
			}
		}()
	}
	wg.Wait()
}

// TestConfigWatcher_OnConfigChange tests callback registration
func TestConfigWatcher_OnConfigChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 5
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.StopWatching()

	callbackCalled := make(chan bool, 1)
	var receivedOld, receivedNew *Config

	watcher.OnConfigChange(func(oldConfig, newConfig *Config) {
		receivedOld = oldConfig
		receivedNew = newConfig
		callbackCalled <- true
	})

	// Start watching in background
	go watcher.StartWatching()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	newConfigContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
  - id: test-2
    type: gemini
    name: TestAgent2
orchestrator:
  mode: reactive
  max_turns: 10
`

	if err := os.WriteFile(configPath, []byte(newConfigContent), 0600); err != nil {
		t.Fatalf("Failed to update test config: %v", err)
	}

	// Wait for callback with timeout
	select {
	case <-callbackCalled:
		// Callback was called
		if receivedOld == nil {
			t.Error("Old config should not be nil")
		}
		if receivedNew == nil {
			t.Error("New config should not be nil")
		}

		if receivedOld.Orchestrator.MaxTurns != 5 {
			t.Errorf("Old config max_turns should be 5, got %d", receivedOld.Orchestrator.MaxTurns)
		}

		if receivedNew.Orchestrator.MaxTurns != 10 {
			t.Errorf("New config max_turns should be 10, got %d", receivedNew.Orchestrator.MaxTurns)
		}

		if len(receivedNew.Agents) != 2 {
			t.Errorf("New config should have 2 agents, got %d", len(receivedNew.Agents))
		}

		if receivedNew.Orchestrator.Mode != "reactive" {
			t.Errorf("New config mode should be reactive, got %s", receivedNew.Orchestrator.Mode)
		}

	case <-time.After(2 * time.Second):
		t.Error("Callback was not called within timeout")
	}
}

// TestConfigWatcher_MultipleCallbacks tests multiple callbacks
func TestConfigWatcher_MultipleCallbacks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 5
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.StopWatching()

	callback1Called := make(chan bool, 1)
	callback2Called := make(chan bool, 1)

	watcher.OnConfigChange(func(oldConfig, newConfig *Config) {
		callback1Called <- true
	})

	watcher.OnConfigChange(func(oldConfig, newConfig *Config) {
		callback2Called <- true
	})

	// Start watching in background
	go watcher.StartWatching()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	newConfigContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 10
`

	if err := os.WriteFile(configPath, []byte(newConfigContent), 0600); err != nil {
		t.Fatalf("Failed to update test config: %v", err)
	}

	// Wait for both callbacks
	timeout := time.After(2 * time.Second)
	callback1Received := false
	callback2Received := false

	for i := 0; i < 2; i++ {
		select {
		case <-callback1Called:
			callback1Received = true
		case <-callback2Called:
			callback2Received = true
		case <-timeout:
			t.Error("Not all callbacks were called within timeout")
			return
		}
	}

	if !callback1Received {
		t.Error("Callback 1 was not called")
	}
	if !callback2Received {
		t.Error("Callback 2 was not called")
	}
}

// TestConfigWatcher_InvalidConfigUpdate tests handling of invalid config on reload
func TestConfigWatcher_InvalidConfigUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 5
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.StopWatching()

	initialConfig := watcher.GetConfig()

	// Start watching in background
	go watcher.StartWatching()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Write invalid config (no agents)
	invalidConfigContent := `version: "1.0"
agents: []
orchestrator:
  mode: round-robin
`

	if err := os.WriteFile(configPath, []byte(invalidConfigContent), 0600); err != nil {
		t.Fatalf("Failed to update test config: %v", err)
	}

	// Give time for reload attempt
	time.Sleep(500 * time.Millisecond)

	// Config should remain unchanged (old valid config)
	currentConfig := watcher.GetConfig()
	if len(currentConfig.Agents) != len(initialConfig.Agents) {
		t.Error("Config should not have changed when reload failed")
	}
}

// TestConfigWatcher_StopWatching tests stopping the watcher
func TestConfigWatcher_StopWatching(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}

	watcherDone := make(chan bool)

	go func() {
		watcher.StartWatching()
		watcherDone <- true
	}()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the watcher
	watcher.StopWatching()

	// Wait for watcher to finish
	select {
	case <-watcherDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Watcher did not stop within timeout")
	}
}

// TestConfigWatcher_ConcurrentReads tests concurrent config reads during reload
func TestConfigWatcher_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 5
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.StopWatching()

	// Start watching in background
	go watcher.StartWatching()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Start concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				config := watcher.GetConfig()
				if config == nil {
					errors <- nil
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Trigger config reload during concurrent reads
	time.Sleep(50 * time.Millisecond)
	newConfigContent := `version: "1.0"
agents:
  - id: test-1
    type: claude
    name: TestAgent
orchestrator:
  mode: round-robin
  max_turns: 10
`

	if err := os.WriteFile(configPath, []byte(newConfigContent), 0600); err != nil {
		t.Fatalf("Failed to update test config: %v", err)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent read error: %v", err)
		}
	}
}
