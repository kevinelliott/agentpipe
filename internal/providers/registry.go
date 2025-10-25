package providers

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kevinelliott/agentpipe/pkg/log"
)

var (
	//go:embed providers.json
	embeddedProvidersJSON []byte

	// globalRegistry is the singleton registry instance
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
)

// Registry manages provider configurations and pricing lookups.
type Registry struct {
	config *ProviderConfig
	mu     sync.RWMutex
}

// GetRegistry returns the global provider registry singleton.
// It loads the embedded providers.json by default, but will override
// with ~/.agentpipe/providers.json if it exists.
func GetRegistry() *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &Registry{}
		if err := globalRegistry.Load(); err != nil {
			log.WithError(err).Error("failed to load provider registry, using embedded defaults")
		}
	})
	return globalRegistry
}

// Load loads the provider configuration from the hybrid source:
// 1. Start with embedded providers.json
// 2. Override with ~/.agentpipe/providers.json if it exists
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load embedded config first
	var config ProviderConfig
	if err := json.Unmarshal(embeddedProvidersJSON, &config); err != nil {
		return fmt.Errorf("failed to parse embedded providers.json: %w", err)
	}
	r.config = &config

	log.WithFields(map[string]interface{}{
		"providers": len(config.Providers),
		"version":   config.Version,
		"source":    "embedded",
	}).Debug("loaded embedded provider config")

	// Try to load override from ~/.agentpipe/providers.json
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return nil // No override available, use embedded
	}

	overridePath := filepath.Join(homeDir, ".agentpipe", "providers.json")
	if _, statErr := os.Stat(overridePath); os.IsNotExist(statErr) {
		return nil // No override file, use embedded
	}

	// Load override config
	data, readErr := os.ReadFile(overridePath)
	if readErr != nil {
		log.WithError(readErr).Warn("failed to read provider override config, using embedded")
		return nil
	}

	var overrideConfig ProviderConfig
	if unmarshalErr := json.Unmarshal(data, &overrideConfig); unmarshalErr != nil {
		log.WithError(unmarshalErr).Warn("failed to parse provider override config, using embedded")
		return nil
	}

	r.config = &overrideConfig

	log.WithFields(map[string]interface{}{
		"providers": len(overrideConfig.Providers),
		"version":   overrideConfig.Version,
		"updated":   overrideConfig.UpdatedAt,
		"source":    "override",
		"path":      overridePath,
	}).Info("loaded provider override config")

	return nil
}

// Reload forces a reload of the provider configuration.
func (r *Registry) Reload() error {
	return r.Load()
}

// GetProvider returns a provider by ID.
func (r *Registry) GetProvider(id string) (*Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config == nil {
		return nil, fmt.Errorf("provider registry not initialized")
	}

	id = strings.ToLower(id)
	for i := range r.config.Providers {
		if strings.ToLower(r.config.Providers[i].ID) == id {
			return &r.config.Providers[i], nil
		}
	}

	return nil, fmt.Errorf("provider not found: %s", id)
}

// GetModel performs smart model lookup:
// 1. Try exact match on model ID
// 2. Try prefix match (e.g., "claude-sonnet-4" matches "claude-sonnet-4-5")
// 3. Try fuzzy match (contains)
// Returns the model and the provider it belongs to.
func (r *Registry) GetModel(modelID string) (*Model, *Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config == nil {
		return nil, nil, fmt.Errorf("provider registry not initialized")
	}

	modelID = strings.ToLower(modelID)

	// Pass 1: Exact match
	for i := range r.config.Providers {
		provider := &r.config.Providers[i]
		for j := range provider.Models {
			if strings.ToLower(provider.Models[j].ID) == modelID {
				log.WithFields(map[string]interface{}{
					"model_id": modelID,
					"match":    "exact",
					"provider": provider.Name,
					"model":    provider.Models[j].Name,
				}).Debug("found model via exact match")
				return &provider.Models[j], provider, nil
			}
		}
	}

	// Pass 2: Prefix match (model starts with the search term)
	for i := range r.config.Providers {
		provider := &r.config.Providers[i]
		for j := range provider.Models {
			modelIDLower := strings.ToLower(provider.Models[j].ID)
			if strings.HasPrefix(modelIDLower, modelID) {
				log.WithFields(map[string]interface{}{
					"model_id":  modelID,
					"match":     "prefix",
					"provider":  provider.Name,
					"model":     provider.Models[j].Name,
					"actual_id": provider.Models[j].ID,
				}).Info("found model via prefix match")
				return &provider.Models[j], provider, nil
			}
		}
	}

	// Pass 3: Fuzzy match (model ID contains the search term)
	for i := range r.config.Providers {
		provider := &r.config.Providers[i]
		for j := range provider.Models {
			modelIDLower := strings.ToLower(provider.Models[j].ID)
			if strings.Contains(modelIDLower, modelID) {
				log.WithFields(map[string]interface{}{
					"model_id":  modelID,
					"match":     "fuzzy",
					"provider":  provider.Name,
					"model":     provider.Models[j].Name,
					"actual_id": provider.Models[j].ID,
				}).Warn("found model via fuzzy match - this may not be accurate")
				return &provider.Models[j], provider, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("model not found: %s", modelID)
}

// ListProviders returns all available providers.
func (r *Registry) ListProviders() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config == nil {
		return nil
	}

	// Return a copy to prevent external modification
	providers := make([]Provider, len(r.config.Providers))
	copy(providers, r.config.Providers)
	return providers
}

// GetConfig returns the current provider configuration.
func (r *Registry) GetConfig() *ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.config
}
