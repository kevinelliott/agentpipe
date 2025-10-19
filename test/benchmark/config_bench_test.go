package benchmark

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
)

// BenchmarkConfigValidate benchmarks configuration validation
func BenchmarkConfigValidate(b *testing.B) {
	cfg := createTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}

// BenchmarkConfigMarshal benchmarks YAML marshaling
func BenchmarkConfigMarshal(b *testing.B) {
	cfg := createTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = yaml.Marshal(cfg)
	}
}

// BenchmarkConfigUnmarshal benchmarks YAML unmarshaling
func BenchmarkConfigUnmarshal(b *testing.B) {
	yamlData := []byte(`version: "1.0"
agents:
  - id: agent-1
    type: claude
    name: Agent1
    prompt: "You are a helpful assistant"
    rate_limit: 10.0
    rate_limit_burst: 5
  - id: agent-2
    type: gemini
    name: Agent2
    prompt: "You are another assistant"

orchestrator:
  mode: round-robin
  max_turns: 10
  turn_timeout: 30s
  response_delay: 1s

logging:
  enabled: true
  chat_log_dir: /tmp/logs
  log_format: json
  show_metrics: true
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg config.Config
		_ = yaml.Unmarshal(yamlData, &cfg)
	}
}

// BenchmarkConfigLoadFromFile benchmarks loading config from file
func BenchmarkConfigLoadFromFile(b *testing.B) {
	// Create temporary config file
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench-config.yaml")

	cfg := createTestConfig()
	data, _ := yaml.Marshal(cfg)
	_ = os.WriteFile(configPath, data, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.LoadConfig(configPath)
	}
}

// BenchmarkConfigCreationWithDefaults benchmarks creating config with defaults
func BenchmarkConfigCreationWithDefaults(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := config.NewDefaultConfig()
		_ = cfg.Validate()
	}
}

// BenchmarkNewDefaultConfig benchmarks default config creation
func BenchmarkNewDefaultConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.NewDefaultConfig()
	}
}

// Helper function to create a test configuration
func createTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.Agents = []agent.AgentConfig{
		{
			ID:             "agent-1",
			Type:           "claude",
			Name:           "Claude",
			Prompt:         "You are a helpful assistant",
			RateLimit:      10.0,
			RateLimitBurst: 5,
		},
		{
			ID:     "agent-2",
			Type:   "gemini",
			Name:   "Gemini",
			Prompt: "You are another assistant",
		},
	}
	return cfg
}
