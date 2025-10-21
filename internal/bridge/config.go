package bridge

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the configuration for the bridge streaming functionality
type Config struct {
	Enabled       bool   `mapstructure:"enabled"`
	URL           string `mapstructure:"url"`
	APIKey        string `mapstructure:"api_key"`
	TimeoutMs     int    `mapstructure:"timeout_ms"`
	RetryAttempts int    `mapstructure:"retry_attempts"`
	LogLevel      string `mapstructure:"log_level"`
}

// LoadConfig loads bridge configuration from viper, environment variables, and defaults
// Precedence: environment variables > viper config > defaults
func LoadConfig() *Config {
	config := &Config{
		Enabled:       false, // Disabled by default
		URL:           getDefaultURL(),
		TimeoutMs:     10000,
		RetryAttempts: 3,
		LogLevel:      "info",
	}

	// Load from viper config file if available
	if viper.IsSet("bridge.enabled") {
		config.Enabled = viper.GetBool("bridge.enabled")
	}
	if viper.IsSet("bridge.url") {
		config.URL = cleanBaseURL(viper.GetString("bridge.url"))
	}
	if viper.IsSet("bridge.api_key") {
		config.APIKey = viper.GetString("bridge.api_key")
	}
	if viper.IsSet("bridge.timeout_ms") {
		config.TimeoutMs = viper.GetInt("bridge.timeout_ms")
	}
	if viper.IsSet("bridge.retry_attempts") {
		config.RetryAttempts = viper.GetInt("bridge.retry_attempts")
	}
	if viper.IsSet("bridge.log_level") {
		config.LogLevel = viper.GetString("bridge.log_level")
	}

	// Override with environment variables (highest priority)
	if enabled := os.Getenv("AGENTPIPE_STREAM_ENABLED"); enabled == "true" || enabled == "1" {
		config.Enabled = true
	} else if enabled == "false" || enabled == "0" {
		config.Enabled = false
	}

	if url := os.Getenv("AGENTPIPE_STREAM_URL"); url != "" {
		config.URL = cleanBaseURL(url)
	}

	if apiKey := os.Getenv("AGENTPIPE_STREAM_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	return config
}

// cleanBaseURL removes trailing /api/ingest if present and trailing slashes
func cleanBaseURL(url string) string {
	// Remove trailing /api/ingest if user accidentally included it
	url = strings.TrimSuffix(url, "/api/ingest/")
	url = strings.TrimSuffix(url, "/api/ingest")
	url = strings.TrimSuffix(url, "/")
	return url
}

// getDefaultURL returns the default URL based on build-time configuration
// and runtime environment variable override
func getDefaultURL() string {
	// Check for runtime environment override
	if env := os.Getenv("AGENTPIPE_ENV"); env == "production" {
		return "https://agentpipe.ai"
	} else if env == "development" {
		return "http://localhost:3000"
	}

	// Otherwise use build-time default from build tags
	return DefaultURL
}
