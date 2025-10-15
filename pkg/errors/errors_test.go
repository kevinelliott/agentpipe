package errors

import (
	"errors"
	"testing"
)

func TestAgentError(t *testing.T) {
	baseErr := errors.New("connection refused")
	err := NewAgentError("claude", "send_message", baseErr)

	if err.AgentName != "claude" {
		t.Errorf("expected AgentName 'claude', got '%s'", err.AgentName)
	}
	if err.Operation != "send_message" {
		t.Errorf("expected Operation 'send_message', got '%s'", err.Operation)
	}

	expected := "agent claude failed during send_message: connection refused"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}

	if !errors.Is(err, baseErr) {
		t.Error("expected Unwrap to return base error")
	}
}

func TestConfigError(t *testing.T) {
	t.Run("with field", func(t *testing.T) {
		err := NewConfigError("max_turns", 0, "must be positive")

		expected := "config error in field 'max_turns': must be positive"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := NewConfigError("", nil, "invalid configuration")

		expected := "config error: invalid configuration"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("with cause", func(t *testing.T) {
		baseErr := errors.New("file not found")
		err := NewConfigErrorWithCause("config_file", "/path/to/config", "cannot read file", baseErr)

		if !errors.Is(err, baseErr) {
			t.Error("expected Unwrap to return base error")
		}
	})
}

func TestInitializationError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		baseErr := errors.New("CLI not found")
		err := NewInitializationError("claude-agent", "executable not in PATH", baseErr)

		expected := "initialization failed for claude-agent: executable not in PATH: CLI not found"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}

		if !errors.Is(err, baseErr) {
			t.Error("expected Unwrap to return base error")
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := NewInitializationError("gemini-agent", "not configured", nil)

		expected := "initialization failed for gemini-agent: not configured"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestCommunicationError(t *testing.T) {
	t.Run("with agent name", func(t *testing.T) {
		baseErr := errors.New("context deadline exceeded")
		err := NewCommunicationError("copilot", "timeout", "request took too long", baseErr)

		expected := "communication error with agent copilot (timeout): request took too long"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}

		if !errors.Is(err, baseErr) {
			t.Error("expected Unwrap to return base error")
		}
	})

	t.Run("without agent name", func(t *testing.T) {
		err := NewCommunicationError("", "network", "connection lost", nil)

		expected := "communication error (network): connection lost"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		err := NewValidationError("temperature", 2.5, "must be between 0 and 1")

		expected := "validation error for field 'temperature' (value: 2.5): must be between 0 and 1"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("without value", func(t *testing.T) {
		err := NewValidationError("api_key", nil, "is required")

		expected := "validation error for field 'api_key': is required"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestOrchestratorError(t *testing.T) {
	t.Run("with turn number", func(t *testing.T) {
		baseErr := errors.New("no agents available")
		err := NewOrchestratorError("round-robin", 5, "all agents failed", baseErr)

		expected := "orchestrator error in round-robin mode (turn 5): all agents failed"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}

		if !errors.Is(err, baseErr) {
			t.Error("expected Unwrap to return base error")
		}
	})

	t.Run("without turn number", func(t *testing.T) {
		err := NewOrchestratorError("reactive", 0, "configuration invalid", nil)

		expected := "orchestrator error in reactive mode: configuration invalid"
		if err.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}
