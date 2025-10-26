package cmd

import (
	"fmt"
	"strings"

	"github.com/kevinelliott/agentpipe/internal/providers"
	"github.com/kevinelliott/agentpipe/pkg/log"
)

// ModelSupport defines whether an agent type supports and/or requires model specification.
type ModelSupport struct {
	Supported bool // Whether the agent CLI supports --model flag
	Required  bool // Whether model specification is required (e.g., OpenRouter)
}

// agentModelSupport defines model support capabilities for each agent type.
var agentModelSupport = map[string]ModelSupport{
	// CLI agents with --model support
	"claude": {
		Supported: true,
		Required:  false,
	},
	"gemini": {
		Supported: true,
		Required:  false,
	},
	"copilot": {
		Supported: true,
		Required:  false,
	},
	"qwen": {
		Supported: true,
		Required:  false,
	},
	"factory": {
		Supported: true,
		Required:  false,
	},
	"qoder": {
		Supported: true,
		Required:  false,
	},
	"codex": {
		Supported: true,
		Required:  false,
	},
	"groq": {
		Supported: true,
		Required:  false,
	},
	"crush": {
		Supported: true,
		Required:  false,
	},

	// API agents that require model
	"openrouter": {
		Supported: true,
		Required:  true,
	},

	// CLI agents without --model support
	"kimi": {
		Supported: false,
		Required:  false,
	},
	"cursor": {
		Supported: false,
		Required:  false,
	},
	"amp": {
		Supported: false,
		Required:  false,
	},
	"opencode": {
		Supported: false,
		Required:  false,
	},

	// Ollama is special - it requires model but uses different mechanism
	"ollama": {
		Supported: true,
		Required:  true,
	},
}

// validateAgentType checks if the agent type is valid and registered.
func validateAgentType(agentType string) error {
	if agentType == "" {
		return fmt.Errorf("agent type cannot be empty")
	}

	// Check if agent type exists in our support map
	if _, exists := agentModelSupport[agentType]; !exists {
		return fmt.Errorf("unknown agent type: %s", agentType)
	}

	return nil
}

// validateModelForAgent checks if model specification is valid for the given agent type.
func validateModelForAgent(agentType, model string) error {
	support, exists := agentModelSupport[agentType]
	if !exists {
		return fmt.Errorf("unknown agent type: %s", agentType)
	}

	// If model is specified but agent doesn't support it
	if model != "" && !support.Supported {
		return fmt.Errorf("agent type '%s' does not support model specification", agentType)
	}

	// If model is required but not provided
	if model == "" && support.Required {
		return fmt.Errorf("agent type '%s' requires model specification (use format: %s:model:name)", agentType, agentType)
	}

	return nil
}

// validateModelInRegistry checks if the model exists in the provider registry.
// This provides helpful warnings for API-based agents but doesn't block CLI agents.
func validateModelInRegistry(agentType, model string) error {
	if model == "" {
		return nil // No model to validate
	}

	registry := providers.GetRegistry()

	// Try to find the model in the registry
	modelInfo, provider, err := registry.GetModel(model)
	if err != nil {
		// For API agents like OpenRouter, this is a warning since the model might exist
		// but not be in our registry yet
		support := agentModelSupport[agentType]
		if support.Required {
			log.WithFields(map[string]interface{}{
				"agent_type": agentType,
				"model":      model,
			}).Warn("model not found in provider registry (cost estimates may be inaccurate)")
		}
		return nil // Don't block - the API/CLI will validate
	}

	log.WithFields(map[string]interface{}{
		"agent_type": agentType,
		"model":      modelInfo.ID,
		"provider":   provider.Name,
	}).Debug("model validated in provider registry")

	return nil
}

// checkModelRequired returns an error if model is required but not provided.
func checkModelRequired(agentType, model string) error {
	support, exists := agentModelSupport[agentType]
	if !exists {
		return nil // Unknown type, will be caught by validateAgentType
	}

	if support.Required && model == "" {
		return fmt.Errorf("agent type '%s' requires model specification", agentType)
	}

	return nil
}

// parseAgentSpecWithModel parses an agent specification in the format:
//   - type:name (existing format)
//   - type:model:name (new format with model)
//
// Returns agentType, model, name, and error.
func parseAgentSpecWithModel(spec string) (agentType, model, name string, err error) {
	if spec == "" {
		return "", "", "", fmt.Errorf("agent specification cannot be empty")
	}

	parts := strings.Split(spec, ":")

	switch len(parts) {
	case 1:
		// Just type, auto-generate name
		agentType = parts[0]
		name = agentType

	case 2:
		// type:name (existing format)
		agentType = parts[0]
		name = parts[1]

	case 3:
		// type:model:name (new format)
		agentType = parts[0]
		model = parts[1]
		name = parts[2]

	default:
		return "", "", "", fmt.Errorf("invalid agent specification format: %s (expected type:name or type:model:name)", spec)
	}

	// Validate agent type
	if err := validateAgentType(agentType); err != nil {
		return "", "", "", err
	}

	// Validate model for this agent type
	if err := validateModelForAgent(agentType, model); err != nil {
		return "", "", "", err
	}

	// Check if model is required
	if err := checkModelRequired(agentType, model); err != nil {
		return "", "", "", err
	}

	// Validate model in registry (warnings only)
	_ = validateModelInRegistry(agentType, model)

	return agentType, model, name, nil
}
