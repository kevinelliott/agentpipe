package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kevinelliott/agentpipe/pkg/log"
)

// Server is an HTTP server that exposes Prometheus metrics.
type Server struct {
	addr     string
	server   *http.Server
	registry *prometheus.Registry
	metrics  *Metrics
}

// ServerConfig contains configuration for the metrics server.
type ServerConfig struct {
	// Addr is the address to listen on (e.g., ":9090")
	Addr string

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes
	WriteTimeout time.Duration

	// Registry is the Prometheus registry to use (if nil, a new one is created)
	Registry *prometheus.Registry
}

// NewServer creates a new metrics server with the given configuration.
func NewServer(config ServerConfig) *Server {
	if config.Addr == "" {
		config.Addr = ":9090"
	}

	if config.ReadTimeout == 0 {
		config.ReadTimeout = 5 * time.Second
	}

	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}

	registry := config.Registry
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	metrics := NewMetrics(registry)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", indexHandler)

	server := &http.Server{
		Addr:         config.Addr,
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return &Server{
		addr:     config.Addr,
		server:   server,
		registry: registry,
		metrics:  metrics,
	}
}

// Start starts the metrics server.
// This method blocks until the server is stopped or encounters an error.
func (s *Server) Start() error {
	log.WithField("addr", s.addr).Info("starting metrics server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).Error("metrics server failed")
		return fmt.Errorf("metrics server failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the metrics server.
func (s *Server) Stop(ctx context.Context) error {
	log.Info("stopping metrics server")

	if err := s.server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("metrics server shutdown failed")
		return fmt.Errorf("metrics server shutdown failed: %w", err)
	}

	log.Info("metrics server stopped")
	return nil
}

// GetMetrics returns the metrics instance for recording.
func (s *Server) GetMetrics() *Metrics {
	return s.metrics
}

// GetRegistry returns the Prometheus registry.
func (s *Server) GetRegistry() *prometheus.Registry {
	return s.registry
}

// healthHandler handles health check requests.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","service":"agentpipe-metrics"}`)
}

// indexHandler handles requests to the root path.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>AgentPipe Metrics</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .endpoint { margin: 20px 0; padding: 15px; background-color: #f5f5f5; border-left: 4px solid #0066cc; }
        code { background-color: #e8e8e8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>AgentPipe Metrics Server</h1>
    <p>This server exposes Prometheus metrics for AgentPipe.</p>

    <div class="endpoint">
        <h2><a href="/metrics">/metrics</a></h2>
        <p>Prometheus metrics endpoint in OpenMetrics format.</p>
    </div>

    <div class="endpoint">
        <h2><a href="/health">/health</a></h2>
        <p>Health check endpoint. Returns JSON with service status.</p>
    </div>

    <h2>Available Metrics</h2>
    <ul>
        <li><code>agentpipe_agent_requests_total</code> - Total agent requests by agent name, type, and status</li>
        <li><code>agentpipe_agent_request_duration_seconds</code> - Agent request duration histogram</li>
        <li><code>agentpipe_agent_tokens_total</code> - Total tokens consumed by agent and type</li>
        <li><code>agentpipe_agent_cost_usd_total</code> - Total estimated cost in USD</li>
        <li><code>agentpipe_agent_errors_total</code> - Total errors by agent and error type</li>
        <li><code>agentpipe_active_conversations</code> - Current number of active conversations</li>
        <li><code>agentpipe_conversation_turns_total</code> - Total conversation turns by mode</li>
        <li><code>agentpipe_message_size_bytes</code> - Message size distribution</li>
        <li><code>agentpipe_retry_attempts_total</code> - Total retry attempts by agent</li>
        <li><code>agentpipe_rate_limit_hits_total</code> - Total rate limit hits</li>
    </ul>

    <h2>Example Prometheus Configuration</h2>
    <pre><code>scrape_configs:
  - job_name: 'agentpipe'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s</code></pre>
</body>
</html>`)
}
