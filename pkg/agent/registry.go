package agent

import (
	"fmt"
	"sync"
)

type Factory func() Agent

type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
	agents    map[string]Agent
}

var defaultRegistry = &Registry{
	factories: make(map[string]Factory),
	agents:    make(map[string]Agent),
}

func RegisterFactory(agentType string, factory Factory) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.factories[agentType] = factory
}

func CreateAgent(config AgentConfig) (Agent, error) {
	defaultRegistry.mu.RLock()
	factory, ok := defaultRegistry.factories[config.Type]
	defaultRegistry.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown agent type: %s", config.Type)
	}

	agent := factory()
	if err := agent.Initialize(config); err != nil {
		return nil, fmt.Errorf("failed to initialize agent: %w", err)
	}

	defaultRegistry.mu.Lock()
	defaultRegistry.agents[config.ID] = agent
	defaultRegistry.mu.Unlock()

	return agent, nil
}

func GetAgent(id string) (Agent, bool) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	agent, ok := defaultRegistry.agents[id]
	return agent, ok
}

func ListAgents() []Agent {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()

	agents := make([]Agent, 0, len(defaultRegistry.agents))
	for _, agent := range defaultRegistry.agents {
		agents = append(agents, agent)
	}
	return agents
}

func ClearAgents() {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.agents = make(map[string]Agent)
}
