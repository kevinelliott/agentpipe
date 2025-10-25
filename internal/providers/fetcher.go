package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// CatwalkBaseURL is the base URL for Catwalk's provider configs on GitHub
	CatwalkBaseURL = "https://raw.githubusercontent.com/charmbracelet/catwalk/main/internal/providers/configs"
	// SchemaVersion is the current version of the provider config schema
	SchemaVersion = "1.0"
)

// ProviderFileNames lists all available provider config files from Catwalk
var ProviderFileNames = []string{
	"aihubmix.json",
	"anthropic.json",
	"azure.json",
	"bedrock.json",
	"cerebras.json",
	"chutes.json",
	"deepseek.json",
	"gemini.json",
	"groq.json",
	"huggingface.json",
	"openai.json",
	"openrouter.json",
	"venice.json",
	"vertexai.json",
	"xai.json",
	"zai.json",
}

// FetchProvidersFromCatwalk fetches all provider configs from Catwalk's GitHub repository
// and returns a consolidated ProviderConfig.
func FetchProvidersFromCatwalk() (*ProviderConfig, error) {
	providers := make([]Provider, 0, len(ProviderFileNames))
	client := &http.Client{Timeout: 30 * time.Second}

	for _, filename := range ProviderFileNames {
		url := fmt.Sprintf("%s/%s", CatwalkBaseURL, filename)

		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", filename, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch %s: HTTP %d", filename, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		var provider Provider
		if err := json.Unmarshal(body, &provider); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
		}

		providers = append(providers, provider)
	}

	config := &ProviderConfig{
		Version:   SchemaVersion,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    "https://github.com/charmbracelet/catwalk",
		Providers: providers,
	}

	return config, nil
}

// FetchProviderFromCatwalk fetches a single provider config from Catwalk's GitHub repository.
func FetchProviderFromCatwalk(filename string) (*Provider, error) {
	url := fmt.Sprintf("%s/%s", CatwalkBaseURL, filename)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", filename, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filename, err)
	}

	var provider Provider
	if err := json.Unmarshal(body, &provider); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	return &provider, nil
}
