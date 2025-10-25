package providers

import (
	"testing"
)

// TestFetchProviderFromCatwalk tests fetching a single provider config
// This is a network test and will be skipped in offline environments
func TestFetchProviderFromCatwalk(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	provider, err := FetchProviderFromCatwalk("anthropic.json")
	if err != nil {
		t.Fatalf("Failed to fetch anthropic.json: %v", err)
	}

	if provider.ID != "anthropic" {
		t.Errorf("Expected provider ID 'anthropic', got '%s'", provider.ID)
	}

	if provider.Name == "" {
		t.Error("Provider name is empty")
	}

	if len(provider.Models) == 0 {
		t.Error("Provider has no models")
	}

	t.Logf("Fetched provider: %s with %d models", provider.Name, len(provider.Models))
}

// TestFetchProvidersFromCatwalk tests fetching all provider configs
// This is a network test and will be skipped in offline environments
func TestFetchProvidersFromCatwalk(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	config, err := FetchProvidersFromCatwalk()
	if err != nil {
		t.Fatalf("Failed to fetch providers: %v", err)
	}

	if config.Version == "" {
		t.Error("Config version is empty")
	}

	if config.UpdatedAt == "" {
		t.Error("Config updated_at is empty")
	}

	if config.Source == "" {
		t.Error("Config source is empty")
	}

	if len(config.Providers) == 0 {
		t.Fatal("No providers fetched")
	}

	t.Logf("Fetched %d providers from Catwalk", len(config.Providers))

	// Verify key providers exist
	providerIDs := make(map[string]bool)
	for _, p := range config.Providers {
		providerIDs[p.ID] = true
	}

	expectedProviders := []string{"anthropic", "openai", "gemini", "deepseek"}
	for _, id := range expectedProviders {
		if !providerIDs[id] {
			t.Errorf("Expected provider '%s' not found in fetched configs", id)
		}
	}
}
