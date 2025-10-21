package bridge

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clean environment
	os.Unsetenv("AGENTPIPE_STREAM_ENABLED")
	os.Unsetenv("AGENTPIPE_STREAM_URL")
	os.Unsetenv("AGENTPIPE_STREAM_API_KEY")
	os.Unsetenv("AGENTPIPE_ENV")

	// Reset viper
	viper.Reset()

	config := LoadConfig()

	// Test defaults
	if config.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if config.TimeoutMs != 10000 {
		t.Errorf("Expected TimeoutMs=10000, got %d", config.TimeoutMs)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("Expected RetryAttempts=3, got %d", config.RetryAttempts)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected LogLevel=info, got %s", config.LogLevel)
	}

	// URL should be the default (depends on build tag)
	if config.URL == "" {
		t.Error("Expected URL to be set to default")
	}

	t.Logf("Default URL: %s", config.URL)
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Clean environment first
	os.Unsetenv("AGENTPIPE_ENV")
	viper.Reset()

	// Set environment variables
	os.Setenv("AGENTPIPE_STREAM_ENABLED", "true")
	os.Setenv("AGENTPIPE_STREAM_URL", "https://example.com")
	os.Setenv("AGENTPIPE_STREAM_API_KEY", "sk_test_key")

	defer func() {
		os.Unsetenv("AGENTPIPE_STREAM_ENABLED")
		os.Unsetenv("AGENTPIPE_STREAM_URL")
		os.Unsetenv("AGENTPIPE_STREAM_API_KEY")
	}()

	config := LoadConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true from env var")
	}

	if config.URL != "https://example.com" {
		t.Errorf("Expected URL=https://example.com, got %s", config.URL)
	}

	if config.APIKey != "sk_test_key" {
		t.Errorf("Expected APIKey=sk_test_key, got %s", config.APIKey)
	}
}

func TestLoadConfig_ViperConfig(t *testing.T) {
	// Clean environment
	os.Unsetenv("AGENTPIPE_STREAM_ENABLED")
	os.Unsetenv("AGENTPIPE_STREAM_URL")
	os.Unsetenv("AGENTPIPE_STREAM_API_KEY")
	os.Unsetenv("AGENTPIPE_ENV")

	// Reset and configure viper
	viper.Reset()
	viper.Set("bridge.enabled", true)
	viper.Set("bridge.url", "https://viper.example.com")
	viper.Set("bridge.api_key", "sk_viper_key")
	viper.Set("bridge.timeout_ms", 15000)
	viper.Set("bridge.retry_attempts", 5)
	viper.Set("bridge.log_level", "debug")

	defer viper.Reset()

	config := LoadConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true from viper")
	}

	if config.URL != "https://viper.example.com" {
		t.Errorf("Expected URL=https://viper.example.com, got %s", config.URL)
	}

	if config.APIKey != "sk_viper_key" {
		t.Errorf("Expected APIKey=sk_viper_key, got %s", config.APIKey)
	}

	if config.TimeoutMs != 15000 {
		t.Errorf("Expected TimeoutMs=15000, got %d", config.TimeoutMs)
	}

	if config.RetryAttempts != 5 {
		t.Errorf("Expected RetryAttempts=5, got %d", config.RetryAttempts)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel=debug, got %s", config.LogLevel)
	}
}

func TestLoadConfig_EnvironmentOverridesViper(t *testing.T) {
	// Configure viper
	viper.Reset()
	viper.Set("bridge.enabled", false)
	viper.Set("bridge.url", "https://viper.example.com")
	viper.Set("bridge.api_key", "sk_viper_key")

	// Set environment variables (should override viper)
	os.Setenv("AGENTPIPE_STREAM_ENABLED", "true")
	os.Setenv("AGENTPIPE_STREAM_URL", "https://env.example.com")
	os.Setenv("AGENTPIPE_STREAM_API_KEY", "sk_env_key")

	defer func() {
		os.Unsetenv("AGENTPIPE_STREAM_ENABLED")
		os.Unsetenv("AGENTPIPE_STREAM_URL")
		os.Unsetenv("AGENTPIPE_STREAM_API_KEY")
		viper.Reset()
	}()

	config := LoadConfig()

	// Environment variables should win
	if !config.Enabled {
		t.Error("Expected Enabled=true from env var (should override viper)")
	}

	if config.URL != "https://env.example.com" {
		t.Errorf("Expected URL from env var, got %s", config.URL)
	}

	if config.APIKey != "sk_env_key" {
		t.Errorf("Expected APIKey from env var, got %s", config.APIKey)
	}
}

func TestCleanBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com/", "https://example.com"},
		{"https://example.com/api/ingest", "https://example.com"},
		{"https://example.com/api/ingest/", "https://example.com"},
		{"http://localhost:3000", "http://localhost:3000"},
		{"http://localhost:3000/", "http://localhost:3000"},
		{"http://localhost:3000/api/ingest", "http://localhost:3000"},
	}

	for _, tt := range tests {
		result := cleanBaseURL(tt.input)
		if result != tt.expected {
			t.Errorf("cleanBaseURL(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestGetDefaultURL(t *testing.T) {
	// Clean environment
	os.Unsetenv("AGENTPIPE_ENV")

	url := getDefaultURL()

	// Should return either localhost or production URL depending on build tag
	if url != "http://localhost:3000" && url != "https://agentpipe.ai" {
		t.Errorf("Unexpected default URL: %s", url)
	}

	t.Logf("Default URL (no env var): %s", url)

	// Test with environment variable override
	os.Setenv("AGENTPIPE_ENV", "production")
	defer os.Unsetenv("AGENTPIPE_ENV")

	url = getDefaultURL()
	if url != "https://agentpipe.ai" {
		t.Errorf("Expected production URL when AGENTPIPE_ENV=production, got %s", url)
	}

	os.Setenv("AGENTPIPE_ENV", "development")
	url = getDefaultURL()
	if url != "http://localhost:3000" {
		t.Errorf("Expected development URL when AGENTPIPE_ENV=development, got %s", url)
	}
}

func TestLoadConfig_EnabledVariations(t *testing.T) {
	// Test different values for enabled flag
	tests := []struct {
		envValue string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"", false}, // empty should use default (false)
	}

	viper.Reset()
	defer os.Unsetenv("AGENTPIPE_STREAM_ENABLED")

	for _, tt := range tests {
		t.Run("enabled="+tt.envValue, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("AGENTPIPE_STREAM_ENABLED", tt.envValue)
			} else {
				os.Unsetenv("AGENTPIPE_STREAM_ENABLED")
			}

			config := LoadConfig()
			if config.Enabled != tt.expected {
				t.Errorf("With AGENTPIPE_STREAM_ENABLED=%s, expected Enabled=%v, got %v",
					tt.envValue, tt.expected, config.Enabled)
			}
		})
	}
}
