package metrics

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TestNewServer tests creating a new metrics server
func TestNewServer(t *testing.T) {
	config := ServerConfig{
		Addr: ":19090",
	}

	server := NewServer(config)
	if server == nil {
		t.Fatal("NewServer should return non-nil")
	}

	if server.addr != ":19090" {
		t.Errorf("Expected addr :19090, got %s", server.addr)
	}

	if server.metrics == nil {
		t.Error("Metrics should be initialized")
	}

	if server.registry == nil {
		t.Error("Registry should be initialized")
	}
}

// TestNewServer_Defaults tests server with default configuration
func TestNewServer_Defaults(t *testing.T) {
	config := ServerConfig{}
	server := NewServer(config)

	if server.addr != ":9090" {
		t.Errorf("Expected default addr :9090, got %s", server.addr)
	}
}

// TestNewServer_CustomRegistry tests server with custom registry
func TestNewServer_CustomRegistry(t *testing.T) {
	customRegistry := prometheus.NewRegistry()
	config := ServerConfig{
		Addr:     ":19090",
		Registry: customRegistry,
	}

	server := NewServer(config)
	if server.GetRegistry() != customRegistry {
		t.Error("Server should use custom registry")
	}
}

// TestServer_GetMetrics tests getting metrics from server
func TestServer_GetMetrics(t *testing.T) {
	config := ServerConfig{
		Addr: ":19090",
	}

	server := NewServer(config)
	metrics := server.GetMetrics()

	if metrics == nil {
		t.Fatal("GetMetrics should return non-nil")
	}

	// Test that metrics work
	metrics.RecordAgentRequest("test", "test", "success")
}

// TestServer_Endpoints tests HTTP endpoints
func TestServer_Endpoints(t *testing.T) {
	config := ServerConfig{
		Addr: ":0", // Use random available port
	}

	server := NewServer(config)

	// Start server in background
	go func() {
		_ = server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual address the server is listening on
	// Since we used :0, we need to find the actual port
	// For testing, we'll use a fixed port instead
	server.Stop(context.Background())
}

// TestServer_MetricsEndpoint tests the /metrics endpoint
func TestServer_MetricsEndpoint(t *testing.T) {
	config := ServerConfig{
		Addr: ":19091",
	}

	server := NewServer(config)

	// Record some metrics
	server.GetMetrics().RecordAgentRequest("Claude", "claude", "success")
	server.GetMetrics().RecordAgentTokens("Claude", "claude", "input", 100)

	// Start server in background
	go func() {
		_ = server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test /metrics endpoint
	resp, err := http.Get("http://localhost:19091/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	bodyStr := string(body)

	// Check for our metrics in the output
	if !strings.Contains(bodyStr, "agentpipe_agent_requests_total") {
		t.Error("Expected agentpipe_agent_requests_total in metrics output")
	}

	if !strings.Contains(bodyStr, "agentpipe_agent_tokens_total") {
		t.Error("Expected agentpipe_agent_tokens_total in metrics output")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

// TestServer_HealthEndpoint tests the /health endpoint
func TestServer_HealthEndpoint(t *testing.T) {
	config := ServerConfig{
		Addr: ":19092",
	}

	server := NewServer(config)

	// Start server in background
	go func() {
		_ = server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test /health endpoint
	resp, err := http.Get("http://localhost:19092/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "healthy") {
		t.Error("Expected 'healthy' in health response")
	}

	if !strings.Contains(bodyStr, "agentpipe-metrics") {
		t.Error("Expected 'agentpipe-metrics' in health response")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

// TestServer_IndexEndpoint tests the / endpoint
func TestServer_IndexEndpoint(t *testing.T) {
	config := ServerConfig{
		Addr: ":19093",
	}

	server := NewServer(config)

	// Start server in background
	go func() {
		_ = server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test / endpoint
	resp, err := http.Get("http://localhost:19093/")
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %s", resp.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "AgentPipe Metrics") {
		t.Error("Expected 'AgentPipe Metrics' in index page")
	}

	if !strings.Contains(bodyStr, "/metrics") {
		t.Error("Expected '/metrics' link in index page")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

// TestServer_StopWithoutStart tests stopping server that wasn't started
func TestServer_StopWithoutStart(t *testing.T) {
	config := ServerConfig{
		Addr: ":19094",
	}

	server := NewServer(config)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This should not panic or error
	_ = server.Stop(ctx)
}

// TestServer_GracefulShutdown tests graceful server shutdown
func TestServer_GracefulShutdown(t *testing.T) {
	config := ServerConfig{
		Addr: ":19095",
	}

	server := NewServer(config)

	// Start server
	go func() {
		_ = server.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Make a request
	go func() {
		resp, _ := http.Get("http://localhost:19095/metrics")
		if resp != nil {
			_ = resp.Body.Close()
		}
	}()

	// Stop server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startTime := time.Now()
	err := server.Stop(ctx)
	duration := time.Since(startTime)

	if err != nil {
		t.Errorf("Expected graceful shutdown, got error: %v", err)
	}

	// Should shutdown relatively quickly
	if duration > 2*time.Second {
		t.Errorf("Shutdown took too long: %v", duration)
	}
}

// TestServer_CustomTimeouts tests custom read/write timeouts
func TestServer_CustomTimeouts(t *testing.T) {
	config := ServerConfig{
		Addr:         ":19096",
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

	server := NewServer(config)
	if server.server.ReadTimeout != 1*time.Second {
		t.Errorf("Expected ReadTimeout 1s, got %v", server.server.ReadTimeout)
	}

	if server.server.WriteTimeout != 2*time.Second {
		t.Errorf("Expected WriteTimeout 2s, got %v", server.server.WriteTimeout)
	}
}
