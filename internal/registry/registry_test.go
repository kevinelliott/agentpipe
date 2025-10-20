package registry

import (
	"runtime"
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	registry, err := LoadRegistry()
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	if registry == nil {
		t.Fatal("Registry is nil")
	}

	if len(registry.agents) == 0 {
		t.Fatal("Registry has no agents")
	}
}

func TestGetAll(t *testing.T) {
	agents := GetAll()

	if len(agents) == 0 {
		t.Fatal("GetAll returned no agents")
	}

	// Verify we have the expected agents
	expectedCount := 11 // Amp, Claude, Codex, Copilot, Cursor, Factory, Gemini, OpenCode, Qoder, Qwen, Ollama
	if len(agents) != expectedCount {
		t.Errorf("Expected %d agents, got %d", expectedCount, len(agents))
	}
}

func TestGetByName(t *testing.T) {
	tests := []struct {
		name      string
		wantError bool
	}{
		{"Claude", false},
		{"claude", false}, // Case insensitive
		{"CLAUDE", false}, // Case insensitive
		{"Ollama", false},
		{"NonExistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := GetByName(tt.name)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for agent '%s', got nil", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for agent '%s': %v", tt.name, err)
				}
				if agent == nil {
					t.Errorf("Expected agent for '%s', got nil", tt.name)
				}
			}
		})
	}
}

func TestGetByCommand(t *testing.T) {
	tests := []struct {
		command   string
		wantError bool
		wantName  string
	}{
		{"claude", false, "Claude"},
		{"ollama", false, "Ollama"},
		{"gemini", false, "Gemini"},
		{"nonexistent", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			agent, err := GetByCommand(tt.command)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for command '%s', got nil", tt.command)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for command '%s': %v", tt.command, err)
				}
				if agent == nil {
					t.Errorf("Expected agent for command '%s', got nil", tt.command)
				}
				if agent.Name != tt.wantName {
					t.Errorf("Expected agent name '%s', got '%s'", tt.wantName, agent.Name)
				}
			}
		})
	}
}

func TestGetInstallCommand(t *testing.T) {
	agent, err := GetByName("Claude")
	if err != nil {
		t.Fatalf("Failed to get Claude agent: %v", err)
	}

	cmd, err := agent.GetInstallCommand()
	if err != nil {
		t.Errorf("GetInstallCommand failed: %v", err)
	}

	if cmd == "" {
		t.Error("Install command is empty")
	}

	// Verify it returns the correct command for current OS
	expectedCmd := agent.Install[runtime.GOOS]
	if cmd != expectedCmd {
		t.Errorf("Expected install command '%s', got '%s'", expectedCmd, cmd)
	}
}

func TestGetUpgradeCommand(t *testing.T) {
	agent, err := GetByName("Claude")
	if err != nil {
		t.Fatalf("Failed to get Claude agent: %v", err)
	}

	cmd, err := agent.GetUpgradeCommand()
	if err != nil {
		t.Errorf("GetUpgradeCommand failed: %v", err)
	}

	if cmd == "" {
		t.Error("Upgrade command is empty")
	}
}

func TestIsInstallable(t *testing.T) {
	tests := []struct {
		name        string
		wantInstall bool
	}{
		{"Claude", true}, // npm install
		{"Ollama", true}, // brew install (darwin) or curl script
		{"Amp", true},    // npm install
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := GetByName(tt.name)
			if err != nil {
				t.Fatalf("Failed to get agent '%s': %v", tt.name, err)
			}

			isInstallable := agent.IsInstallable()
			if isInstallable != tt.wantInstall {
				t.Errorf("Expected IsInstallable() = %v for %s, got %v", tt.wantInstall, tt.name, isInstallable)
			}
		})
	}
}

func TestAgentMetadata(t *testing.T) {
	agent, err := GetByName("Claude")
	if err != nil {
		t.Fatalf("Failed to get Claude agent: %v", err)
	}

	if agent.Name != "Claude" {
		t.Errorf("Expected name 'Claude', got '%s'", agent.Name)
	}

	if agent.Command != "claude" {
		t.Errorf("Expected command 'claude', got '%s'", agent.Command)
	}

	if agent.Description == "" {
		t.Error("Description is empty")
	}

	if agent.Docs == "" {
		t.Error("Docs URL is empty")
	}

	if !agent.RequiresAuth {
		t.Error("Claude should require authentication")
	}
}

func TestOllamaDoesNotRequireAuth(t *testing.T) {
	agent, err := GetByName("Ollama")
	if err != nil {
		t.Fatalf("Failed to get Ollama agent: %v", err)
	}

	if agent.RequiresAuth {
		t.Error("Ollama should not require authentication")
	}
}
