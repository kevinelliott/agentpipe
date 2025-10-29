package bridge

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// ZerologJSONWriter is a zerolog writer that emits log entries as log.entry JSON events
type ZerologJSONWriter struct {
	emitter *StdoutEmitter
	mu      sync.Mutex
}

// NewZerologJSONWriter creates a new zerolog writer that emits JSON events
func NewZerologJSONWriter(emitter *StdoutEmitter) *ZerologJSONWriter {
	return &ZerologJSONWriter{
		emitter: emitter,
	}
}

// Write implements io.Writer for zerolog
func (w *ZerologJSONWriter) Write(p []byte) (n int, err error) {
	// Parse the zerolog JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		// If we can't parse it, just write raw to stderr as fallback
		fmt.Fprint(os.Stderr, string(p))
		return len(p), nil
	}

	// Extract standard zerolog fields
	level, _ := logEntry["level"].(string)
	message, _ := logEntry["message"].(string)

	// Build metadata from remaining fields
	metadata := make(map[string]interface{})
	for k, v := range logEntry {
		// Skip standard fields that we handle separately
		if k != "level" && k != "message" && k != "time" && k != "timestamp" {
			metadata[k] = v
		}
	}

	// Emit as log.entry event
	w.mu.Lock()
	defer w.mu.Unlock()

	w.emitter.EmitLogEntry(
		level,        // level (debug, info, warn, error, etc.)
		"",           // agent_id (not applicable for system logs)
		"",           // agent_name (not applicable for system logs)
		"",           // agent_type (not applicable for system logs)
		message,      // content
		"diagnostic", // role (use "diagnostic" to distinguish from agent messages)
		nil,          // metrics
		metadata,     // metadata (all other fields from zerolog)
	)

	return len(p), nil
}

var _ io.Writer = (*ZerologJSONWriter)(nil)
