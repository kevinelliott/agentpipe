package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetRegistry(t *testing.T) {
	registry := GetRegistry()
	if registry == nil {
		t.Fatal("GetRegistry returned nil")
	}

	if registry.config == nil {
		t.Fatal("Registry config is nil")
	}

	if len(registry.config.Providers) == 0 {
		t.Fatal("No providers loaded in registry")
	}

	t.Logf("Loaded %d providers", len(registry.config.Providers))
}

func TestGetProvider(t *testing.T) {
	registry := GetRegistry()

	tests := []struct {
		id          string
		shouldExist bool
	}{
		{"anthropic", true},
		{"openai", true},
		{"gemini", true},
		{"deepseek", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			provider, err := registry.GetProvider(tt.id)
			if tt.shouldExist {
				if err != nil {
					t.Fatalf("Expected provider %s to exist, got error: %v", tt.id, err)
				}
				if provider == nil {
					t.Fatalf("Expected provider %s to exist, got nil", tt.id)
				}
				if provider.ID != tt.id {
					t.Errorf("Expected provider ID %s, got %s", tt.id, provider.ID)
				}
			} else {
				if err == nil {
					t.Fatalf("Expected provider %s to not exist, but it does", tt.id)
				}
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	registry := GetRegistry()

	tests := []struct {
		modelID     string
		shouldExist bool
		matchType   string // "exact", "prefix", "fuzzy", or ""
	}{
		// Exact matches
		{"claude-sonnet-4-5-20250929", true, "exact"},
		{"gpt-5", true, "exact"},
		{"gemini-2.5-pro", true, "exact"},

		// Prefix matches
		{"claude-sonnet-4", true, "prefix"},
		{"gpt", true, "prefix"},

		// Should not exist
		{"totally-fake-model", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			model, provider, err := registry.GetModel(tt.modelID)
			if tt.shouldExist {
				if err != nil {
					t.Fatalf("Expected model %s to exist, got error: %v", tt.modelID, err)
				}
				if model == nil {
					t.Fatalf("Expected model %s to exist, got nil", tt.modelID)
				}
				if provider == nil {
					t.Fatalf("Expected provider for model %s to exist, got nil", tt.modelID)
				}
				t.Logf("Found model %s (%s) from provider %s via %s match",
					model.ID, model.Name, provider.Name, tt.matchType)
			} else {
				if err == nil {
					t.Fatalf("Expected model %s to not exist, but it does", tt.modelID)
				}
			}
		})
	}
}

func TestListProviders(t *testing.T) {
	registry := GetRegistry()
	providers := registry.ListProviders()

	if len(providers) == 0 {
		t.Fatal("ListProviders returned empty list")
	}

	// Verify we got the expected number of providers (16 from Catwalk)
	expectedCount := 16
	if len(providers) != expectedCount {
		t.Logf("Warning: Expected %d providers, got %d", expectedCount, len(providers))
	}

	// Verify each provider has at least one model
	for _, p := range providers {
		if len(p.Models) == 0 {
			t.Errorf("Provider %s has no models", p.Name)
		}
	}
}

func TestRegistryReload(t *testing.T) {
	registry := GetRegistry()

	// Create a test override config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	agentpipeDir := filepath.Join(homeDir, ".agentpipe")
	if mkdirErr := os.MkdirAll(agentpipeDir, 0755); mkdirErr != nil {
		t.Skip("Cannot create .agentpipe directory")
	}

	overridePath := filepath.Join(agentpipeDir, "providers.json")

	// Save the original file if it exists
	originalData, originalExists := []byte{}, false
	if fileData, readErr := os.ReadFile(overridePath); readErr == nil {
		originalData = fileData
		originalExists = true
	}

	// Cleanup function to restore original state
	defer func() {
		if originalExists {
			os.WriteFile(overridePath, originalData, 0644)
		} else {
			os.Remove(overridePath)
		}
	}()

	// Create a minimal test config
	testConfig := &ProviderConfig{
		Version:   "test",
		UpdatedAt: "2025-01-01T00:00:00Z",
		Source:    "test",
		Providers: []Provider{
			{
				ID:   "test-provider",
				Name: "Test Provider",
				Models: []Model{
					{
						ID:           "test-model",
						Name:         "Test Model",
						CostPer1MIn:  1.0,
						CostPer1MOut: 2.0,
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if writeErr := os.WriteFile(overridePath, data, 0600); writeErr != nil {
		t.Fatalf("Failed to write test config: %v", writeErr)
	}

	// Reload the registry
	if reloadErr := registry.Reload(); reloadErr != nil {
		t.Fatalf("Failed to reload registry: %v", reloadErr)
	}

	// Verify the test provider is loaded
	provider, getErr := registry.GetProvider("test-provider")
	if getErr != nil {
		t.Fatalf("Expected test-provider to exist after reload, got error: %v", getErr)
	}
	if provider.Name != "Test Provider" {
		t.Errorf("Expected provider name 'Test Provider', got %s", provider.Name)
	}
}

func TestModelPricing(t *testing.T) {
	// Force reload to ensure we have the embedded config, not test override
	registry := GetRegistry()
	if err := registry.Load(); err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Test Anthropic Claude pricing
	model, _, err := registry.GetModel("claude-sonnet-4-5-20250929")
	if err != nil {
		t.Fatalf("Failed to get Claude model: %v", err)
	}

	if model.CostPer1MIn <= 0 {
		t.Errorf("Expected positive input cost, got %f", model.CostPer1MIn)
	}
	if model.CostPer1MOut <= 0 {
		t.Errorf("Expected positive output cost, got %f", model.CostPer1MOut)
	}

	t.Logf("Claude Sonnet 4.5 pricing: $%.2f in / $%.2f out per 1M tokens",
		model.CostPer1MIn, model.CostPer1MOut)
}

func TestProviderConfigStructure(t *testing.T) {
	// Force reload to ensure we have the embedded config, not test override
	registry := GetRegistry()
	if err := registry.Load(); err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	config := registry.GetConfig()

	if config.Version == "" {
		t.Error("Config version is empty")
	}
	if config.UpdatedAt == "" {
		t.Error("Config updated_at is empty")
	}
	if config.Source == "" {
		t.Error("Config source is empty")
	}

	t.Logf("Provider config: version=%s, updated=%s, source=%s",
		config.Version, config.UpdatedAt, config.Source)
}
