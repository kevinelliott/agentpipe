# AgentPipe Codebase Issues Analysis

This document contains a comprehensive analysis of bugs, security issues, and performance problems identified in the AgentPipe codebase.

## Summary

Total Issues Identified: 12
- High Severity: 1
- Medium Severity: 6  
- Low Severity: 5

## Issue List

### Issue 1: Race Condition in Orchestrator Message History

**Severity:** Medium  
**Category:** Concurrency Bug  
**Location:** `pkg/orchestrator/orchestrator.go`, lines 746-1000

#### Description
The `getAgentResponse` method has a race condition when accessing message history and updating the current turn number. Messages are read without a lock, then after processing, the lock is acquired to append new messages. Between these operations, concurrent access could lead to inconsistent state.

#### Code Location
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

#### Impact
- Incorrect message ordering in multi-threaded scenarios
- Turn number mismatches in metrics/logging
- Potential data races when using streaming bridge

#### Recommended Fix
Hold the lock throughout the entire operation or ensure atomicity of read-modify-write cycles.

---

### Issue 2: Insecure API Key Logging Risk in Bridge Client

**Severity:** High  
**Category:** Security Vulnerability  
**Location:** `internal/bridge/client.go`, lines 108-136, 324-327

#### Description
The bridge client includes API keys in HTTP headers without explicit protection against logging. While the documentation claims "API keys never logged", there's no actual safeguard preventing accidental exposure in error messages or debug output.

#### Code Location
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

#### Impact
API keys could be exposed in:
- Log files (`~/.agentpipe/chats/`)
- Error output to stderr/stdout
- Debug logs when `LogLevel: "debug"`
- Crash dumps or panic traces

#### Recommended Fix
1. Add explicit API key redaction in error messages
2. Never log full requests/responses that include headers
3. Add unit tests that verify keys are never in error strings

---

### Issue 3: Memory Leak in Rate Limiter Map

**Severity:** Medium  
**Category:** Memory Leak / Resource Management  
**Location:** `pkg/orchestrator/orchestrator.go`, lines 457-461

#### Description
The orchestrator creates rate limiters for each agent but has no mechanism to remove them. Rate limiters accumulate indefinitely in the `rateLimiters` map.

#### Code Location
```go
o.rateLimiters[a.GetID()] = ratelimit.NewLimiter(rateLimit, rateLimitBurst)
```

#### Impact
- Memory leak in long-running processes
- Unlimited growth of rateLimiters map
- No cleanup/finalization method

#### Recommended Fix
Add a `RemoveAgent()` method and/or a `Close()` method to clean up resources.

---

### Issue 4: Potential Integer Overflow in Token Calculation

**Severity:** Low  
**Category:** Arithmetic Bug  
**Location:** `pkg/utils/tokens.go`, lines 20-24

#### Description
The `EstimateTokens` function performs arithmetic that could overflow with very large inputs.

#### Code Location
```go
wordEstimate := len(words) * 4 / 3  // Could overflow on huge inputs
charEstimate := chars / 4
return (wordEstimate + charEstimate) / 2
```

#### Impact
- Silent overflow leading to negative token counts
- Incorrect cost estimates for large conversations
- Potential panic in edge cases

#### Recommended Fix
Use int64 or add bounds checking with a maximum input size.

---

### Issue 5: Unchecked Error in File Operations

**Severity:** Medium  
**Category:** Error Handling Bug  
**Location:** `pkg/logger/logger.go`, lines 420-426

#### Description
The `writeToFile` method prints errors to stderr but continues execution, leading to silent failures.

#### Code Location
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

#### Impact
- Lost conversation logs
- Incomplete records for debugging
- Users unaware of logging failures
- Disk full conditions silently fail

#### Recommended Fix
Return errors and handle them at call sites, or track error count and warn users.

---

### Issue 6: Unseeded Random Number Generator

**Severity:** Low  
**Category:** Concurrency Bug / Determinism  
**Location:** `pkg/orchestrator/orchestrator.go`, lines 1025-1053

#### Description
The `selectNextAgent` method uses `rand.Intn` without seeding, resulting in deterministic "random" selection.

#### Code Location
```go
targetIndex := rand.Intn(availableCount)
```

#### Impact
- Predictable agent selection in "reactive" mode
- Same sequence every run
- Potential race in concurrent calls (global rand state)

#### Recommended Fix
Seed the random number generator or use `math/rand/v2`:
```go
func init() {
    rand.Seed(time.Now().UnixNano())
}
```

---

### Issue 7: Event Store File Permissions Too Permissive

**Severity:** Medium  
**Category:** Security - File Permissions  
**Location:** `internal/bridge/eventstore.go`, line 31

#### Description
Event log files are created with 0644 permissions (world-readable), potentially exposing conversation content.

#### Code Location
```go
file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
```

#### Impact
- Conversation content visible to other users on shared systems
- Potential privacy violation
- Sensitive data exposure

#### Recommended Fix
Use more restrictive permissions (0600):
```go
file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
```

---

### Issue 8: Configuration Directory Permissions Not Enforced

**Severity:** Medium  
**Category:** Security - File Permissions  
**Location:** `pkg/config/config.go`, line 145

#### Description
`SaveConfig` creates files with correct permissions but doesn't verify parent directory permissions.

#### Code Location
```go
if err := os.WriteFile(path, data, 0600); err != nil {
```

#### Impact
- Parent directory might be world-readable
- Config files with sensitive data exposed
- Security audit failure

#### Recommended Fix
Ensure parent directory has correct permissions:
```go
dir := filepath.Dir(path)
if err := os.MkdirAll(dir, 0700); err != nil {
    return err
}
```

---

### Issue 9: Panic Risk from Negative Terminal Width

**Severity:** Low  
**Category:** Defensive Programming  
**Location:** `pkg/logger/logger.go`, lines 234, 444-449

#### Description
The logger uses terminal width without bounds checking, which could cause a panic if the terminal width is negative.

#### Code Location
```go
output.WriteString(separatorStyle.Render(strings.Repeat("─", min(l.termWidth, 80))))
```

#### Impact
- Crash on malformed terminal size
- Poor error handling

#### Recommended Fix
Add bounds checking:
```go
width := max(0, min(l.termWidth, 80))
output.WriteString(separatorStyle.Render(strings.Repeat("─", width)))
```

---

### Issue 10: Resource Exhaustion via Unbounded Retry

**Severity:** Medium  
**Category:** DoS / Resource Exhaustion  
**Location:** `pkg/orchestrator/orchestrator.go`, lines 789-846

#### Description
Retry logic has no maximum total time limit, only a maximum number of retries. With exponential backoff, this could block for extremely long periods.

#### Code Location
```go
for attempt := 0; attempt <= o.config.MaxRetries; attempt++ {
    delay := o.calculateBackoffDelay(attempt)
    // ... wait and retry
}
```

#### Impact
- Conversation hangs indefinitely with high MaxRetries
- Resource exhaustion
- No absolute timeout

#### Recommended Fix
Add absolute timeout:
```go
retryCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
// Check retryCtx.Done() in retry loop
```

---

### Issue 11: Missing Validation for Model Names

**Severity:** Low  
**Category:** Input Validation  
**Location:** `pkg/adapters/openrouter.go`, lines 52-58

#### Description
Model names from configuration are accepted without format validation.

#### Code Location
```go
if o.Config.Model == "" {
    return fmt.Errorf("model must be specified for OpenRouter agent")
}
```

#### Impact
- API errors only caught at runtime
- Poor user experience
- Potential injection if model name concatenated into URLs

#### Recommended Fix
Add model name format validation:
```go
const modelPattern = `^[a-zA-Z0-9/_-]+$`
if !regexp.MustCompile(modelPattern).MatchString(o.Config.Model) {
    return fmt.Errorf("invalid model name format")
}
```

---

### Issue 12: HTTP Client Timeout Too Long for Non-Streaming Requests

**Severity:** Low  
**Category:** Performance / Availability  
**Location:** `pkg/client/openai_compat.go`, line 33

#### Description
HTTP client has a 120-second timeout, which is longer than the default turn timeout (30s).

#### Code Location
```go
httpClient: &http.Client{
    Timeout: 120 * time.Second,
},
```

#### Impact
- Long hangs on network issues
- Timeout longer than turn timeout
- Poor user experience

#### Recommended Fix
Use shorter timeout matching turn timeout:
```go
Timeout: 30 * time.Second, // Match default turn timeout
```

Or make it configurable based on context.

---

## Recommendations

### Immediate Actions (High Priority)
1. **Issue 2**: Add API key redaction to prevent exposure in logs
2. **Issue 7**: Fix event store file permissions to 0600
3. **Issue 8**: Enforce directory permissions for config files

### Short-term Improvements (Medium Priority)
1. **Issue 1**: Fix race condition in orchestrator
2. **Issue 3**: Add resource cleanup mechanism
3. **Issue 5**: Improve error handling in file operations
4. **Issue 10**: Add absolute timeout to retry logic

### Long-term Enhancements (Low Priority)
1. **Issue 4**: Add bounds checking for token calculation
2. **Issue 6**: Seed random number generator properly
3. **Issue 9**: Add defensive bounds checking
4. **Issue 11**: Validate model name format
5. **Issue 12**: Make HTTP timeout configurable

## Testing Recommendations

1. Add integration tests with race detector enabled
2. Test with very large inputs to verify integer overflow handling
3. Test file operation error scenarios (disk full, permission denied)
4. Add security tests for API key exposure
5. Test concurrent access patterns
6. Verify file permissions on all created files/directories
