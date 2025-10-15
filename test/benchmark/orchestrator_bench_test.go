package benchmark

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
)

// BenchmarkAgent is a minimal agent for benchmarking
type BenchmarkAgent struct {
	agent.BaseAgent
}

func (a *BenchmarkAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	return "benchmark response", nil
}

func (a *BenchmarkAgent) StreamMessage(ctx context.Context, messages []agent.Message, writer io.Writer) error {
	_, err := writer.Write([]byte("benchmark response"))
	return err
}

func (a *BenchmarkAgent) IsAvailable() bool {
	return true
}

func (a *BenchmarkAgent) HealthCheck(ctx context.Context) error {
	return nil
}

// BenchmarkGetMessages benchmarks message history retrieval
func BenchmarkGetMessages(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run("Size"+string(rune('0'+size/10)), func(b *testing.B) {
			orch := setupOrchestratorWithMessages(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = orch.GetMessages()
			}
		})
	}
}

// BenchmarkAddAgent benchmarks agent registration
func BenchmarkAddAgent(b *testing.B) {
	config := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		TurnTimeout:   30 * time.Second,
		ResponseDelay: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		orch := orchestrator.NewOrchestrator(config, io.Discard)
		testAgent := &BenchmarkAgent{
			BaseAgent: agent.BaseAgent{
				ID:   "bench-agent",
				Name: "BenchAgent",
				Type: "benchmark",
			},
		}
		b.StartTimer()

		orch.AddAgent(testAgent)
	}
}

// BenchmarkBackoffCalculation benchmarks backoff delay calculation
func BenchmarkBackoffCalculation(b *testing.B) {
	config := orchestrator.OrchestratorConfig{
		Mode:              orchestrator.ModeRoundRobin,
		MaxRetries:        5,
		RetryInitialDelay: 1 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		RetryMultiplier:   2.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate backoff calculation
		delay := float64(config.RetryInitialDelay)
		for attempt := 1; attempt <= 5; attempt++ {
			delay = delay * config.RetryMultiplier
			if delay > float64(config.RetryMaxDelay) {
				delay = float64(config.RetryMaxDelay)
			}
			_ = time.Duration(delay)
		}
	}
}

// BenchmarkMessageCopy benchmarks the cost of copying message slices
func BenchmarkMessageCopy(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		messages := make([]agent.Message, size)
		for i := 0; i < size; i++ {
			messages[i] = agent.Message{
				AgentID:   "agent-1",
				AgentName: "Agent1",
				Content:   "This is message number " + string(rune('0'+i%10)),
				Timestamp: time.Now().Unix(),
				Role:      "agent",
			}
		}

		b.Run("Size"+string(rune('0'+size/10)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				copied := make([]agent.Message, len(messages))
				copy(copied, messages)
			}
		})
	}
}

// BenchmarkConversationSmall benchmarks a small conversation
func BenchmarkConversationSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		config := orchestrator.OrchestratorConfig{
			Mode:          orchestrator.ModeRoundRobin,
			MaxTurns:      3,
			TurnTimeout:   5 * time.Second,
			ResponseDelay: 1 * time.Millisecond,
		}
		orch := orchestrator.NewOrchestrator(config, io.Discard)

		agent1 := &BenchmarkAgent{
			BaseAgent: agent.BaseAgent{
				ID:   "agent-1",
				Name: "Agent1",
				Type: "benchmark",
			},
		}
		orch.AddAgent(agent1)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		b.StartTimer()

		_ = orch.Start(ctx)

		b.StopTimer()
		cancel()
	}
}

// BenchmarkConversationMultiAgent benchmarks multi-agent conversations
func BenchmarkConversationMultiAgent(b *testing.B) {
	agentCounts := []int{2, 5, 10}

	for _, count := range agentCounts {
		b.Run("Agents"+string(rune('0'+count)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				config := orchestrator.OrchestratorConfig{
					Mode:          orchestrator.ModeRoundRobin,
					MaxTurns:      2,
					TurnTimeout:   5 * time.Second,
					ResponseDelay: 1 * time.Millisecond,
				}
				orch := orchestrator.NewOrchestrator(config, io.Discard)

				for j := 0; j < count; j++ {
					agent := &BenchmarkAgent{
						BaseAgent: agent.BaseAgent{
							ID:   "agent-" + string(rune('0'+j)),
							Name: "Agent" + string(rune('0'+j)),
							Type: "benchmark",
						},
					}
					orch.AddAgent(agent)
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				b.StartTimer()

				_ = orch.Start(ctx)

				b.StopTimer()
				cancel()
			}
		})
	}
}

// Helper function to create orchestrator with pre-populated messages
func setupOrchestratorWithMessages(count int) *orchestrator.Orchestrator {
	config := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		TurnTimeout:   30 * time.Second,
		ResponseDelay: 1 * time.Second,
	}
	orch := orchestrator.NewOrchestrator(config, io.Discard)

	// Add a dummy agent to populate messages
	testAgent := &BenchmarkAgent{
		BaseAgent: agent.BaseAgent{
			ID:   "test-agent",
			Name: "TestAgent",
			Type: "test",
		},
	}
	orch.AddAgent(testAgent)

	// Run a quick conversation to populate messages
	config2 := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ModeRoundRobin,
		MaxTurns:      count,
		TurnTimeout:   5 * time.Second,
		ResponseDelay: 1 * time.Millisecond,
	}
	orch2 := orchestrator.NewOrchestrator(config2, io.Discard)
	orch2.AddAgent(testAgent)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = orch2.Start(ctx)

	return orch2
}
