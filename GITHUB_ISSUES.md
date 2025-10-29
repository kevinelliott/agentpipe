# GitHub Issues to Create

This file contains 12 issues identified in the AgentPipe codebase. Copy each issue section and create it as a GitHub issue with the specified labels.

---

## Issue 1: Race Condition in Orchestrator Message History

**Labels:** `bug`, `concurrency`, `medium-priority`

**Title:** Race Condition in Orchestrator Message History

**Description:**

### Severity
Medium

### Category  
Concurrency Bug

### Description
The `getAgentResponse` method in `pkg/orchestrator/orchestrator.go` has a race condition when accessing message history and updating the current turn number. Messages are read without a lock (via `getMessages()` at line 767), then after processing, the lock is acquired to append new messages (lines 956-962). Between these operations, concurrent access could lead to inconsistent state.

### Location
- **File:** `pkg/orchestrator/orchestrator.go`
- **Lines:** 746-1000

### Code Example
```go
// Line 767: Messages read without lock (inside getMessages)
messages := o.getMessages()

// ... processing happens ...

// Lines 956-962: Lock acquired here, but state might have changed
o.mu.Lock()
o.messages = append(o.messages, msg)
currentTurn := o.currentTurnNumber
o.currentTurnNumber++
bridgeEmitter := o.bridgeEmitter
o.mu.Unlock()
```

### Impact
- Incorrect message ordering in multi-threaded scenarios
- Turn number mismatches in metrics/logging
- Potential data races when using streaming bridge

### Recommended Fix
Hold the lock throughout the entire operation or ensure atomicity of read-modify-write cycles. The lock should protect both the read and write operations to prevent race conditions.

### Risk Assessment
**Medium**: While orchestrator methods are typically called sequentially, the concurrent-safe design with RWMutex suggests multi-threaded use is intended. The race detector would catch this in integration tests.

---

## Issue 2: Security - API Key Logging Risk in Bridge Client

**Labels:** `security`, `high-priority`, `bug`

**Title:** Security: API Key Logging Risk in Bridge Client

**Description:**

### Severity
**HIGH**

### Category
Security Vulnerability

### Description
The bridge client in `internal/bridge/client.go` includes API keys in HTTP headers without explicit protection against logging. While the documentation claims "API keys never logged" (CLAUDE.md line 146), there's no actual safeguard preventing accidental exposure in error messages or debug output.

### Location
- **File:** `internal/bridge/client.go`
- **Lines:** 108-136 (sendRequest method), 324-327 (in pkg/client/openai_compat.go)

### Code Example
```go
// Line 117: API key set in header
req.Header.Set("Authorization", "Bearer "+c.apiKey)

// Line 131: Error body read and could be logged
bodyBytes, _ := io.ReadAll(resp.Body)
return &httpError{
    statusCode: resp.StatusCode,
    message:    string(bodyBytes),
}
```

### Problems
1. No redaction of API keys in error messages
2. HTTP requests/responses could leak keys in debug logs
3. Error types include full error messages which might contain headers

### Impact - CRITICAL
API keys could be exposed in:
- Log files (`~/.agentpipe/chats/`)
- Error output to stderr/stdout
- Debug logs when `LogLevel: "debug"`
- Crash dumps or panic traces
- This could lead to unauthorized API access and abuse

### Recommended Fix
1. Add explicit API key redaction in error messages:
```go
func redactAPIKey(msg string, apiKey string) string {
    if apiKey != "" && len(apiKey) > 4 {
        redacted := apiKey[:4] + "****"
        return strings.ReplaceAll(msg, apiKey, redacted)
    }
    return msg
}
```

2. Never log full requests/responses that include headers
3. Add unit tests that verify keys are never in error strings
4. Consider using structured logging with field-level redaction

### Risk Assessment
**HIGH**: API key exposure is a serious security vulnerability that could lead to unauthorized access and API abuse. This should be fixed immediately.

---

## Issue 3: Memory Leak - Rate Limiter Map Never Cleaned Up

**Labels:** `bug`, `memory-leak`, `medium-priority`

**Title:** Memory Leak: Rate Limiter Map Never Cleaned Up

**Description:**

### Severity
Medium

### Category
Memory Leak / Resource Management

### Description
The orchestrator creates rate limiters for each agent in `AddAgent()` but has no mechanism to remove them. Rate limiters accumulate indefinitely in the `rateLimiters` map, causing a memory leak.

### Location
- **File:** `pkg/orchestrator/orchestrator.go`
- **Lines:** 457-461

### Code Example
```go
o.rateLimiters[a.GetID()] = ratelimit.NewLimiter(rateLimit, rateLimitBurst)
```

### Problem
There's no mechanism to:
- Remove rate limiters when agents are removed
- Clean up resources when orchestrator is destroyed
- Prevent unlimited growth of the rateLimiters map

### Impact
- Memory leak in long-running processes
- Unlimited growth of rateLimiters map
- No cleanup/finalization method
- Could be significant in services that create/destroy many orchestrators

### Recommended Fix
1. Add a `RemoveAgent(agentID string)` method
2. Add a `Close()` method to clean up all resources

### Risk Assessment
**Medium**: Affects long-running processes and services. Not critical for CLI usage but important for server deployments.

---

## Issue 4: Potential Integer Overflow in Token Calculation

**Labels:** `bug`, `low-priority`, `enhancement`

**Title:** Potential Integer Overflow in Token Calculation

**Description:**

### Severity
Low

### Category
Arithmetic Bug

### Description
The `EstimateTokens` function in `pkg/utils/tokens.go` performs arithmetic that could overflow with very large inputs.

### Location
- **File:** `pkg/utils/tokens.go`
- **Lines:** 20-24

### Code Example
```go
wordEstimate := len(words) * 4 / 3  // Could overflow on huge inputs
charEstimate := chars / 4
return (wordEstimate + charEstimate) / 2
```

### Problem
For very large texts (e.g., entire codebases, very long conversations), `len(words) * 4` could overflow an `int`, leading to negative or incorrect token counts.

### Impact
- Silent overflow leading to negative token counts
- Incorrect cost estimates for large conversations
- Potential panic in some edge cases
- Misleading metrics

### Recommended Fix
Use int64 or add bounds checking with a maximum input size.

### Risk Assessment
**Low**: Unlikely to occur in normal usage, but possible with extremely large inputs.

---

## Issue 5: Silent Failure - Unchecked Errors in File Operations

**Labels:** `bug`, `error-handling`, `medium-priority`

**Title:** Silent Failure: Unchecked Errors in File Operations

**Description:**

### Severity
Medium

### Category
Error Handling Bug

### Description
The `writeToFile` method in `pkg/logger/logger.go` prints errors to stderr but continues execution, leading to silent failures that users are unaware of.

### Location
- **File:** `pkg/logger/logger.go`
- **Lines:** 420-426

### Code Example
```go
func (l *ChatLogger) writeToFile(content string) {
    if l.logFile != nil {
        if _, err := l.logFile.WriteString(content); err != nil {
            fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
        }
        if err := l.logFile.Sync(); err != nil {
            fmt.Fprintf(os.Stderr, "Error syncing log file: %v\n", err)
        }
    }
}
```

### Problems
1. Errors are printed to stderr but execution continues
2. No retry logic for transient failures
3. Disk full conditions silently fail
4. Partial writes not detected
5. Users may not notice stderr messages

### Impact
- Lost conversation logs (important for debugging and auditing)
- Incomplete records for debugging
- Users unaware of logging failures
- Data loss in disk full scenarios

### Recommended Fix
Return errors and handle them at call sites, or at minimum track error count and warn users when threshold exceeded.

### Risk Assessment
**Medium**: Important for data integrity and user awareness, especially in production environments.

---

## Issue 6: Unseeded Random Number Generator Causes Deterministic Behavior

**Labels:** `bug`, `low-priority`, `enhancement`

**Title:** Unseeded Random Number Generator Causes Deterministic Behavior

**Description:**

### Severity
Low

### Category
Concurrency Bug / Determinism

### Description
The `selectNextAgent` method uses `rand.Intn` without seeding the random number generator, resulting in deterministic "random" selection that's the same every run.

### Location
- **File:** `pkg/orchestrator/orchestrator.go`
- **Lines:** 1025-1053

### Code Example
```go
targetIndex := rand.Intn(availableCount)
```

### Problems
1. No `rand.Seed()` call = deterministic "random" selection
2. Not thread-safe (uses global rand state)
3. Same sequence every run

### Impact
- Predictable agent selection in "reactive" mode
- Same sequence every run (not truly random)
- Potential race condition in concurrent calls

### Recommended Fix
Seed the random number generator or use `math/rand/v2` (Go 1.24+).

### Risk Assessment
**Low**: Affects randomness quality but doesn't cause crashes. More of a quality issue than a bug.

---

## Issue 7: Security - Event Store Files Created with World-Readable Permissions

**Labels:** `security`, `medium-priority`, `bug`

**Title:** Security: Event Store Files Created with World-Readable Permissions

**Description:**

### Severity
Medium

### Category
Security - File Permissions

### Description
Event log files in `internal/bridge/eventstore.go` are created with 0644 permissions (world-readable), potentially exposing conversation content containing sensitive information.

### Location
- **File:** `internal/bridge/eventstore.go`
- **Line:** 31

### Code Example
```go
file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
```

### Problem
Files are created with permissions `0644` (owner: read+write, group: read, others: read), allowing any user on the system to read conversation logs.

### Impact
- Conversation content visible to other users on shared systems
- Potential privacy violation
- Sensitive data exposure
- Non-compliance with security best practices

### Recommended Fix
Use more restrictive permissions (0600):
```go
file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
```

Also ensure the parent directory has appropriate permissions (0700).

### Risk Assessment
**Medium**: Important security issue on multi-user systems. Should be fixed to protect user privacy.

---

## Issue 8: Security - Configuration Directory Permissions Not Enforced

**Labels:** `security`, `medium-priority`, `bug`

**Title:** Security: Configuration Directory Permissions Not Enforced

**Description:**

### Severity
Medium

### Category
Security - File Permissions

### Description
`SaveConfig` in `pkg/config/config.go` creates files with correct permissions (0600) but doesn't verify or enforce parent directory permissions, potentially leaving sensitive configuration data accessible.

### Location
- **File:** `pkg/config/config.go`
- **Line:** 145

### Code Example
```go
if err := os.WriteFile(path, data, 0600); err != nil {
    return fmt.Errorf("failed to write config file: %w", err)
}
```

### Problems
1. Parent directory might be world-readable (default 0755)
2. No check that `~/.agentpipe` has correct permissions (should be 0700)
3. Config files could contain sensitive data (API keys, tokens)

### Impact
- Config files with sensitive data could be accessed via directory listing
- Non-compliance with security best practices
- Potential exposure of API keys if stored in config

### Recommended Fix
Ensure parent directory has correct permissions before writing files.

### Risk Assessment
**Medium**: Important security issue that could expose sensitive configuration data on shared systems.

---

## Issue 9: Panic Risk - Negative Terminal Width Not Handled

**Labels:** `bug`, `low-priority`, `enhancement`

**Title:** Panic Risk: Negative Terminal Width Not Handled

**Description:**

### Severity
Low

### Category
Defensive Programming

### Description
The logger uses terminal width without bounds checking in `pkg/logger/logger.go`. If terminal width is negative or invalid, `strings.Repeat` will panic.

### Location
- **File:** `pkg/logger/logger.go`
- **Lines:** 234, 444-449

### Code Example
```go
output.WriteString(separatorStyle.Render(strings.Repeat("─", min(l.termWidth, 80))))
```

### Problem
If `l.termWidth` is negative, the `min()` function could return a negative value, causing `strings.Repeat` to panic.

### Impact
- Crash on malformed terminal size
- No defensive programming
- Application crash instead of graceful degradation

### Recommended Fix
Add bounds checking with a max function to ensure non-negative values.

### Risk Assessment
**Low**: Unlikely to occur in normal usage, but good defensive programming practice.

---

## Issue 10: Resource Exhaustion - Unbounded Retry Without Total Timeout

**Labels:** `bug`, `resource-exhaustion`, `medium-priority`

**Title:** Resource Exhaustion: Unbounded Retry Without Total Timeout

**Description:**

### Severity
Medium

### Category
DoS / Resource Exhaustion

### Description
The retry logic in `pkg/orchestrator/orchestrator.go` has no maximum total time limit, only a maximum number of retries. With exponential backoff and high MaxRetries, this could block for extremely long periods.

### Location
- **File:** `pkg/orchestrator/orchestrator.go`
- **Lines:** 789-846

### Code Example
```go
for attempt := 0; attempt <= o.config.MaxRetries; attempt++ {
    delay := o.calculateBackoffDelay(attempt)
    // ... exponential backoff ...
}
```

### Problem
- No absolute maximum time for all retries combined
- With high MaxRetries, total time could exceed hours
- Only per-attempt timeout, not total retry timeout

### Impact
- Conversation hangs indefinitely with high MaxRetries configured
- Resource exhaustion (goroutines blocked)
- Denial of service if MaxRetries misconfigured

### Recommended Fix
Add absolute timeout for the entire retry process (e.g., 5 minutes maximum).

### Risk Assessment
**Medium**: Could cause availability issues and poor user experience if misconfigured.

---

## Issue 11: Missing Input Validation for Model Names

**Labels:** `enhancement`, `low-priority`, `validation`

**Title:** Missing Input Validation for Model Names

**Description:**

### Severity
Low

### Category
Input Validation

### Description
Model names from configuration in `pkg/adapters/openrouter.go` are accepted without format validation, which could lead to runtime errors or security issues.

### Location
- **File:** `pkg/adapters/openrouter.go`
- **Lines:** 52-58

### Code Example
```go
if o.Config.Model == "" {
    return fmt.Errorf("model must be specified for OpenRouter agent")
}
```

### Problems
1. No format validation (could be arbitrary string, special characters)
2. Errors only caught at API call time (late failure)
3. Potential injection if model name used in URLs

### Impact
- API errors only caught at runtime
- Poor user experience
- No early validation

### Recommended Fix
Add model name format validation using regex pattern matching.

### Risk Assessment
**Low**: Mainly affects user experience and error reporting. Security risk is minimal if model names are only used in API calls.

---

## Issue 12: Performance - HTTP Client Timeout Too Long for Non-Streaming

**Labels:** `enhancement`, `low-priority`, `performance`

**Title:** Performance: HTTP Client Timeout Too Long for Non-Streaming Requests

**Description:**

### Severity
Low

### Category
Performance / Availability

### Description
The HTTP client in `pkg/client/openai_compat.go` has a 120-second timeout, which is significantly longer than the default turn timeout (30s) and could block conversation flow.

### Location
- **File:** `pkg/client/openai_compat.go`
- **Line:** 33

### Code Example
```go
httpClient: &http.Client{
    Timeout: 120 * time.Second,
},
```

### Problem
- 120 second timeout is 4x longer than default turn timeout (30s)
- Could cause conversations to hang for very long periods
- Inconsistent timeout behavior between different layers

### Impact
- Long hangs on network issues
- Timeout longer than turn timeout creates confusing behavior
- Poor user experience

### Recommended Fix
Use a configurable timeout that matches the turn timeout, or make it configurable based on context.

### Risk Assessment
**Low**: Affects user experience but doesn't cause data loss or security issues.

---

## Summary

**Total Issues:** 12
- **High Severity:** 1 (Security - API Key Logging)
- **Medium Severity:** 6
- **Low Severity:** 5

### Priority Order for Fixes

1. **Issue 2** - API Key Logging (HIGH - Security)
2. **Issue 7** - Event Store File Permissions (MEDIUM - Security)
3. **Issue 8** - Config Directory Permissions (MEDIUM - Security)
4. **Issue 1** - Race Condition (MEDIUM - Concurrency)
5. **Issue 3** - Memory Leak (MEDIUM - Resource Management)
6. **Issue 5** - File Operation Errors (MEDIUM - Error Handling)
7. **Issue 10** - Unbounded Retry (MEDIUM - Resource Exhaustion)
8. **Issues 4, 6, 9, 11, 12** - Low priority enhancements

