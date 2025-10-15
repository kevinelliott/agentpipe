package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// TestNew tests creating a new logger
func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf)

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	logger.Info("test message")

	if buf.Len() == 0 {
		t.Error("Expected output to be written")
	}
}

// TestNewWithLevel tests creating a logger with a specific level
func TestNewWithLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithLevel(buf, zerolog.ErrorLevel)

	// Debug message should not be logged
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should not be logged at Error level")
	}

	// Error message should be logged
	logger.Error("error message")
	if buf.Len() == 0 {
		t.Error("Error message should be logged")
	}
}

// TestLogLevels tests all log levels
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(*Logger, string)
		level   string
	}{
		{"Debug", func(l *Logger, msg string) { l.Debug(msg) }, "debug"},
		{"Info", func(l *Logger, msg string) { l.Info(msg) }, "info"},
		{"Warn", func(l *Logger, msg string) { l.Warn(msg) }, "warn"},
		{"Error", func(l *Logger, msg string) { l.Error(msg) }, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(buf)

			tt.logFunc(logger, "test message")

			if buf.Len() == 0 {
				t.Error("Expected output to be written")
			}

			// Parse JSON output
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			if logEntry["level"] != tt.level {
				t.Errorf("Expected level %s, got %v", tt.level, logEntry["level"])
			}

			if logEntry["message"] != "test message" {
				t.Errorf("Expected message 'test message', got %v", logEntry["message"])
			}
		})
	}
}

// TestLogFormattedMessages tests formatted log messages
func TestLogFormattedMessages(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(*Logger, string, ...interface{})
	}{
		{"Debugf", func(l *Logger, format string, args ...interface{}) { l.Debugf(format, args...) }},
		{"Infof", func(l *Logger, format string, args ...interface{}) { l.Infof(format, args...) }},
		{"Warnf", func(l *Logger, format string, args ...interface{}) { l.Warnf(format, args...) }},
		{"Errorf", func(l *Logger, format string, args ...interface{}) { l.Errorf(format, args...) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(buf)

			tt.logFunc(logger, "test %s %d", "message", 42)

			if buf.Len() == 0 {
				t.Error("Expected output to be written")
			}

			output := buf.String()
			if !strings.Contains(output, "test message 42") {
				t.Errorf("Expected formatted message, got: %s", output)
			}
		})
	}
}

// TestWithField tests adding fields to logger context
func TestWithField(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf).WithField("user", "testuser")

	logger.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["user"] != "testuser" {
		t.Errorf("Expected user field 'testuser', got %v", logEntry["user"])
	}
}

// TestWithFields tests adding multiple fields
func TestWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf).WithFields(map[string]interface{}{
		"user":   "testuser",
		"action": "login",
		"count":  42,
	})

	logger.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["user"] != "testuser" {
		t.Errorf("Expected user field 'testuser', got %v", logEntry["user"])
	}
	if logEntry["action"] != "login" {
		t.Errorf("Expected action field 'login', got %v", logEntry["action"])
	}
	if logEntry["count"] != float64(42) {
		t.Errorf("Expected count field 42, got %v", logEntry["count"])
	}
}

// TestWithError tests adding error field
func TestWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	testErr := &testError{msg: "test error"}
	logger := New(buf).WithError(testErr)

	logger.Error("operation failed")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["error"] != "test error" {
		t.Errorf("Expected error field 'test error', got %v", logEntry["error"])
	}
}

// TestGlobalLogger tests global logger functions
func TestGlobalLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	SetGlobalLogger(New(buf))

	Info("global test message")

	if buf.Len() == 0 {
		t.Error("Expected output from global logger")
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["message"] != "global test message" {
		t.Errorf("Expected message 'global test message', got %v", logEntry["message"])
	}
}

// TestParseLevel tests level parsing
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zerolog.Level
	}{
		{"trace", zerolog.TraceLevel},
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"warning", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"panic", zerolog.PanicLevel},
		{"unknown", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := ParseLevel(tt.input)
			if level != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, level)
			}
		})
	}
}

// TestInitLogger tests logger initialization
func TestInitLogger(t *testing.T) {
	buf := &bytes.Buffer{}

	// Test JSON output (not pretty)
	InitLogger(buf, zerolog.InfoLevel, false)
	Info("test message")

	if buf.Len() == 0 {
		t.Error("Expected output after InitLogger")
	}

	// Should be valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Expected JSON output, got error: %v", err)
	}

	// Test pretty output
	buf.Reset()
	InitLogger(buf, zerolog.InfoLevel, true)
	Info("pretty message")

	if buf.Len() == 0 {
		t.Error("Expected output after InitLogger with pretty=true")
	}

	// Pretty output should contain the message
	output := buf.String()
	if !strings.Contains(output, "pretty message") {
		t.Errorf("Expected pretty output to contain message, got: %s", output)
	}
}

// TestChainedContext tests chaining multiple context additions
func TestChainedContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf).
		WithField("user", "alice").
		WithField("action", "login").
		WithError(&testError{msg: "auth failed"})

	logger.Error("authentication error")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["user"] != "alice" {
		t.Errorf("Expected user field, got %v", logEntry["user"])
	}
	if logEntry["action"] != "login" {
		t.Errorf("Expected action field, got %v", logEntry["action"])
	}
	if logEntry["error"] != "auth failed" {
		t.Errorf("Expected error field, got %v", logEntry["error"])
	}
}

// Helper types for testing

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Benchmark tests

func BenchmarkInfo(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkInfof(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infof("benchmark message %d", i)
	}
}

func BenchmarkWithFields(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(map[string]interface{}{
			"user":   "testuser",
			"action": "benchmark",
			"count":  i,
		}).Info("benchmark message")
	}
}

func BenchmarkWithError(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(buf)
	testErr := &testError{msg: "benchmark error"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithError(testErr).Error("benchmark error message")
	}
}
