// Package metrics provides Prometheus metrics for AgentPipe.
// It tracks agent requests, durations, tokens, costs, and errors.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// Namespace is the Prometheus namespace for all AgentPipe metrics
	Namespace = "agentpipe"
)

// Metrics contains all Prometheus metrics for AgentPipe.
type Metrics struct {
	// AgentRequests counts total agent requests by agent name and status (success/error)
	AgentRequests *prometheus.CounterVec

	// AgentRequestDuration tracks agent request duration in seconds
	AgentRequestDuration *prometheus.HistogramVec

	// AgentTokens counts tokens consumed by agent and type (input/output)
	AgentTokens *prometheus.CounterVec

	// AgentCost tracks estimated costs in USD by agent
	AgentCost *prometheus.CounterVec

	// AgentErrors counts errors by agent and error type
	AgentErrors *prometheus.CounterVec

	// ActiveConversations tracks the number of active conversations
	ActiveConversations prometheus.Gauge

	// ConversationTurns counts total conversation turns by mode
	ConversationTurns *prometheus.CounterVec

	// MessageSize tracks message size distribution in bytes
	MessageSize *prometheus.HistogramVec

	// RetryAttempts counts retry attempts by agent
	RetryAttempts *prometheus.CounterVec

	// RateLimitHits counts rate limit hits by agent
	RateLimitHits *prometheus.CounterVec
}

var (
	// DefaultMetrics is the default global metrics instance
	DefaultMetrics *Metrics

	// DefaultRegistry is the default Prometheus registry
	DefaultRegistry *prometheus.Registry
)

func init() {
	DefaultRegistry = prometheus.NewRegistry()
	DefaultMetrics = NewMetrics(DefaultRegistry)
}

// NewMetrics creates a new Metrics instance with the given registry.
// If registry is nil, the default Prometheus registry is used.
func NewMetrics(registry prometheus.Registerer) *Metrics {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	m := &Metrics{
		AgentRequests: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "agent_requests_total",
				Help:      "Total number of agent requests by agent name and status",
			},
			[]string{"agent_name", "agent_type", "status"},
		),

		AgentRequestDuration: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "agent_request_duration_seconds",
				Help:      "Agent request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"agent_name", "agent_type"},
		),

		AgentTokens: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "agent_tokens_total",
				Help:      "Total number of tokens consumed by agent and type",
			},
			[]string{"agent_name", "agent_type", "token_type"},
		),

		AgentCost: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "agent_cost_usd_total",
				Help:      "Total estimated cost in USD by agent",
			},
			[]string{"agent_name", "agent_type", "model"},
		),

		AgentErrors: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "agent_errors_total",
				Help:      "Total number of agent errors by agent and error type",
			},
			[]string{"agent_name", "agent_type", "error_type"},
		),

		ActiveConversations: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Name:      "active_conversations",
				Help:      "Current number of active conversations",
			},
		),

		ConversationTurns: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "conversation_turns_total",
				Help:      "Total number of conversation turns by mode",
			},
			[]string{"mode"},
		),

		MessageSize: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "message_size_bytes",
				Help:      "Message size distribution in bytes",
				Buckets:   []float64{100, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000},
			},
			[]string{"agent_name", "direction"},
		),

		RetryAttempts: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "retry_attempts_total",
				Help:      "Total number of retry attempts by agent",
			},
			[]string{"agent_name", "agent_type"},
		),

		RateLimitHits: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "rate_limit_hits_total",
				Help:      "Total number of rate limit hits by agent",
			},
			[]string{"agent_name"},
		),
	}

	return m
}

// RecordAgentRequest records an agent request with its result.
func (m *Metrics) RecordAgentRequest(agentName, agentType, status string) {
	m.AgentRequests.WithLabelValues(agentName, agentType, status).Inc()
}

// RecordAgentDuration records the duration of an agent request in seconds.
func (m *Metrics) RecordAgentDuration(agentName, agentType string, durationSeconds float64) {
	m.AgentRequestDuration.WithLabelValues(agentName, agentType).Observe(durationSeconds)
}

// RecordAgentTokens records tokens consumed by an agent.
func (m *Metrics) RecordAgentTokens(agentName, agentType, tokenType string, count int) {
	m.AgentTokens.WithLabelValues(agentName, agentType, tokenType).Add(float64(count))
}

// RecordAgentCost records the estimated cost of an agent request in USD.
func (m *Metrics) RecordAgentCost(agentName, agentType, model string, cost float64) {
	m.AgentCost.WithLabelValues(agentName, agentType, model).Add(cost)
}

// RecordAgentError records an agent error.
func (m *Metrics) RecordAgentError(agentName, agentType, errorType string) {
	m.AgentErrors.WithLabelValues(agentName, agentType, errorType).Inc()
}

// IncrementActiveConversations increments the active conversations gauge.
func (m *Metrics) IncrementActiveConversations() {
	m.ActiveConversations.Inc()
}

// DecrementActiveConversations decrements the active conversations gauge.
func (m *Metrics) DecrementActiveConversations() {
	m.ActiveConversations.Dec()
}

// RecordConversationTurn records a conversation turn.
func (m *Metrics) RecordConversationTurn(mode string) {
	m.ConversationTurns.WithLabelValues(mode).Inc()
}

// RecordMessageSize records the size of a message in bytes.
func (m *Metrics) RecordMessageSize(agentName, direction string, sizeBytes int) {
	m.MessageSize.WithLabelValues(agentName, direction).Observe(float64(sizeBytes))
}

// RecordRetryAttempt records a retry attempt.
func (m *Metrics) RecordRetryAttempt(agentName, agentType string) {
	m.RetryAttempts.WithLabelValues(agentName, agentType).Inc()
}

// RecordRateLimitHit records a rate limit hit.
func (m *Metrics) RecordRateLimitHit(agentName string) {
	m.RateLimitHits.WithLabelValues(agentName).Inc()
}

// Reset resets all metrics. Useful for testing.
func (m *Metrics) Reset() {
	m.AgentRequests.Reset()
	m.AgentRequestDuration.Reset()
	m.AgentTokens.Reset()
	m.AgentCost.Reset()
	m.AgentErrors.Reset()
	m.ActiveConversations.Set(0)
	m.ConversationTurns.Reset()
	m.MessageSize.Reset()
	m.RetryAttempts.Reset()
	m.RateLimitHits.Reset()
}
