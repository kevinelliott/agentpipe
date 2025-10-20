package registry

import (
	"embed"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

//go:embed agents.json
var agentsFS embed.FS

// AgentDefinition represents an AI agent CLI with installation metadata
type AgentDefinition struct {
	Name           string            `json:"name"`
	Command        string            `json:"command"`
	Description    string            `json:"description"`
	Docs           string            `json:"docs"`
	PackageManager string            `json:"package_manager,omitempty"` // npm, homebrew, or empty for manual install
	PackageName    string            `json:"package_name,omitempty"`    // Package name for the package manager
	Install        map[string]string `json:"install"`
	Uninstall      map[string]string `json:"uninstall"`
	Upgrade        map[string]string `json:"upgrade"`
	RequiresAuth   bool              `json:"requires_auth"`
}

// AgentRegistry holds all agent definitions
type AgentRegistry struct {
	agents map[string]*AgentDefinition
}

type agentsFile struct {
	Agents []AgentDefinition `json:"agents"`
}

var defaultRegistry *AgentRegistry

// init loads the agent registry on package initialization
func init() {
	var err error
	defaultRegistry, err = LoadRegistry()
	if err != nil {
		// This should never happen in production since agents.json is embedded
		panic(fmt.Sprintf("Failed to load agent registry: %v", err))
	}
}

// LoadRegistry loads agent definitions from the embedded JSON file
func LoadRegistry() (*AgentRegistry, error) {
	data, err := agentsFS.ReadFile("agents.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read agents.json: %w", err)
	}

	var af agentsFile
	if err := json.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("failed to parse agents.json: %w", err)
	}

	registry := &AgentRegistry{
		agents: make(map[string]*AgentDefinition),
	}

	for i := range af.Agents {
		agent := &af.Agents[i]
		// Index by lowercase name for case-insensitive lookup
		registry.agents[strings.ToLower(agent.Name)] = agent
	}

	return registry, nil
}

// GetAll returns all agent definitions
func (r *AgentRegistry) GetAll() []*AgentDefinition {
	agents := make([]*AgentDefinition, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// GetByName returns an agent definition by name (case-insensitive)
func (r *AgentRegistry) GetByName(name string) (*AgentDefinition, error) {
	agent, ok := r.agents[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found in registry", name)
	}
	return agent, nil
}

// GetByCommand returns an agent definition by command name
func (r *AgentRegistry) GetByCommand(command string) (*AgentDefinition, error) {
	for _, agent := range r.agents {
		if agent.Command == command {
			return agent, nil
		}
	}
	return nil, fmt.Errorf("agent with command '%s' not found in registry", command)
}

// GetInstallCommand returns the install command for the current OS
func (a *AgentDefinition) GetInstallCommand() (string, error) {
	os := runtime.GOOS
	cmd, ok := a.Install[os]
	if !ok {
		return "", fmt.Errorf("no install command available for OS: %s", os)
	}
	return cmd, nil
}

// GetUninstallCommand returns the uninstall command for the current OS
func (a *AgentDefinition) GetUninstallCommand() (string, error) {
	os := runtime.GOOS
	cmd, ok := a.Uninstall[os]
	if !ok {
		return "", fmt.Errorf("no uninstall command available for OS: %s", os)
	}
	return cmd, nil
}

// GetUpgradeCommand returns the upgrade command for the current OS
func (a *AgentDefinition) GetUpgradeCommand() (string, error) {
	os := runtime.GOOS
	cmd, ok := a.Upgrade[os]
	if !ok {
		return "", fmt.Errorf("no upgrade command available for OS: %s", os)
	}
	return cmd, nil
}

// IsInstallable returns true if the agent can be installed via a command (not just instructions)
func (a *AgentDefinition) IsInstallable() bool {
	cmd, err := a.GetInstallCommand()
	if err != nil {
		return false
	}
	// Check if it's an actual command (not just "See https://...")
	return !strings.HasPrefix(cmd, "See ")
}

// Default returns the default global registry instance
func Default() *AgentRegistry {
	return defaultRegistry
}

// GetAll returns all agent definitions from the default registry
func GetAll() []*AgentDefinition {
	return defaultRegistry.GetAll()
}

// GetByName returns an agent definition by name from the default registry
func GetByName(name string) (*AgentDefinition, error) {
	return defaultRegistry.GetByName(name)
}

// GetByCommand returns an agent definition by command from the default registry
func GetByCommand(command string) (*AgentDefinition, error) {
	return defaultRegistry.GetByCommand(command)
}
