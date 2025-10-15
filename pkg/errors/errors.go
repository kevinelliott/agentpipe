package errors

import (
	"fmt"
)

// AgentError represents an error that occurred during agent operations
type AgentError struct {
	AgentName string
	Operation string
	Err       error
}

func (e *AgentError) Error() string {
	return fmt.Sprintf("agent %s failed during %s: %v", e.AgentName, e.Operation, e.Err)
}

func (e *AgentError) Unwrap() error {
	return e.Err
}

// NewAgentError creates a new AgentError
func NewAgentError(agentName, operation string, err error) *AgentError {
	return &AgentError{
		AgentName: agentName,
		Operation: operation,
		Err:       err,
	}
}

// ConfigError represents a configuration-related error
type ConfigError struct {
	Field   string
	Value   interface{}
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError
func NewConfigError(field string, value interface{}, message string) *ConfigError {
	return &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewConfigErrorWithCause creates a new ConfigError with an underlying cause
func NewConfigErrorWithCause(field string, value interface{}, message string, err error) *ConfigError {
	return &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
		Err:     err,
	}
}

// InitializationError represents an error during agent or system initialization
type InitializationError struct {
	Component string
	Reason    string
	Err       error
}

func (e *InitializationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("initialization failed for %s: %s: %v", e.Component, e.Reason, e.Err)
	}
	return fmt.Sprintf("initialization failed for %s: %s", e.Component, e.Reason)
}

func (e *InitializationError) Unwrap() error {
	return e.Err
}

// NewInitializationError creates a new InitializationError
func NewInitializationError(component, reason string, err error) *InitializationError {
	return &InitializationError{
		Component: component,
		Reason:    reason,
		Err:       err,
	}
}

// CommunicationError represents an error during agent communication
type CommunicationError struct {
	AgentName string
	Type      string // "timeout", "network", "protocol", etc.
	Message   string
	Err       error
}

func (e *CommunicationError) Error() string {
	if e.AgentName != "" {
		return fmt.Sprintf("communication error with agent %s (%s): %s", e.AgentName, e.Type, e.Message)
	}
	return fmt.Sprintf("communication error (%s): %s", e.Type, e.Message)
}

func (e *CommunicationError) Unwrap() error {
	return e.Err
}

// NewCommunicationError creates a new CommunicationError
func NewCommunicationError(agentName, errorType, message string, err error) *CommunicationError {
	return &CommunicationError{
		AgentName: agentName,
		Type:      errorType,
		Message:   message,
		Err:       err,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation error for field '%s' (value: %v): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// OrchestratorError represents an error in the orchestrator
type OrchestratorError struct {
	Mode    string
	Turn    int
	Message string
	Err     error
}

func (e *OrchestratorError) Error() string {
	if e.Turn > 0 {
		return fmt.Sprintf("orchestrator error in %s mode (turn %d): %s", e.Mode, e.Turn, e.Message)
	}
	return fmt.Sprintf("orchestrator error in %s mode: %s", e.Mode, e.Message)
}

func (e *OrchestratorError) Unwrap() error {
	return e.Err
}

// NewOrchestratorError creates a new OrchestratorError
func NewOrchestratorError(mode string, turn int, message string, err error) *OrchestratorError {
	return &OrchestratorError{
		Mode:    mode,
		Turn:    turn,
		Message: message,
		Err:     err,
	}
}
