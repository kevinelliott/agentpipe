# AgentPipe Architecture

## Overview

AgentPipe is a multi-agent orchestration platform that enables AI agents from different providers to communicate and collaborate in structured conversations. The system is built with modularity, extensibility, and production-readiness in mind.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────┐     │
│  │   init   │ │  doctor  │ │   run    │ │  version  │     │
│  └──────────┘ └──────────┘ └──────────┘ └───────────┘     │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator Layer                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Orchestrator (manages conversation flow)           │   │
│  │  • Round-robin mode                                 │   │
│  │  • Reactive mode                                    │   │
│  │  • Free-form mode                                   │   │
│  │  • Retry logic with exponential backoff             │   │
│  │  • Rate limiting per agent                          │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Agent Layer                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │  Claude  │ │  Gemini  │ │ Copilot  │ │  Cursor  │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
│  ┌──────────┐ ┌──────────┐                                 │
│  │   Qwen   │ │  Codex   │  ... (extensible)               │
│  └──────────┘ └──────────┘                                 │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                   Supporting Systems                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │  Logger  │ │  Config  │ │   Rate   │ │  Utils   │      │
│  │          │ │          │ │ Limiter  │ │          │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
│  ┌──────────┐ ┌──────────┐                                 │
│  │  Errors  │ │   TUI    │                                 │
│  └──────────┘ └──────────┘                                 │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Orchestrator (`pkg/orchestrator`)

The orchestrator is the central coordinator that manages multi-agent conversations.

**Responsibilities:**
- Agent registration and lifecycle management
- Turn-taking coordination across different modes
- Message history management
- Retry logic with exponential backoff
- Rate limiting enforcement
- Error handling and recovery
- Metrics collection

**Orchestration Modes:**

1. **Round-Robin Mode**
   - Agents take turns in a fixed circular order
   - Fair distribution of speaking opportunities
   - Predictable conversation flow

2. **Reactive Mode**
   - Random agent selection
   - No agent speaks twice in a row
   - More dynamic conversations

3. **Free-Form Mode**
   - All agents can participate each turn
   - Most flexible but can be chaotic
   - Good for collaborative scenarios

**Key Features:**
- Context-aware message passing
- Graceful degradation on agent failure
- Configurable turn limits and timeouts
- Initial prompt support
- Thread-safe operations

### 2. Agent Abstraction (`pkg/agent`)

The agent layer provides a unified interface for different AI providers.

**Agent Interface:**
```go
type Agent interface {
    GetID() string
    GetName() string
    GetType() string
    GetModel() string
    GetRateLimit() float64
    GetRateLimitBurst() int
    Initialize(config AgentConfig) error
    SendMessage(ctx context.Context, messages []Message) (string, error)
    StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error
    Announce() string
    IsAvailable() bool
    HealthCheck(ctx context.Context) error
}
```

**Supported Agents:**
- Claude (via `claude` CLI)
- Gemini (via `gemini` CLI)
- GitHub Copilot (via `gh copilot` CLI)
- Cursor (via `cursor-agent` CLI)
- Qwen (via `qwen` CLI)
- Codex (via `codex` CLI)

**BaseAgent:**
- Common functionality shared across all agents
- Default implementations for standard methods
- Reduces boilerplate in agent adapters

### 3. Rate Limiting (`pkg/ratelimit`)

Production-ready rate limiting to prevent API abuse.

**Algorithm:** Token Bucket
- Allows burst traffic
- Maintains average rate limit
- Thread-safe implementation

**Features:**
- Configurable rate (requests/second)
- Configurable burst size
- Dynamic rate adjustment
- Statistics monitoring
- Zero-cost when disabled

**Usage:**
```go
limiter := ratelimit.NewLimiter(10.0, 5) // 10 req/s, burst of 5
err := limiter.Wait(ctx) // Block until token available
allowed := limiter.Allow() // Non-blocking check
```

### 4. Retry Logic

Exponential backoff with configurable parameters.

**Configuration:**
- `MaxRetries`: Maximum retry attempts (default: 3)
- `RetryInitialDelay`: Initial delay before first retry (default: 1s)
- `RetryMaxDelay`: Maximum delay between retries (default: 30s)
- `RetryMultiplier`: Backoff multiplier (default: 2.0)

**Formula:**
```
delay = InitialDelay * (Multiplier ^ attempt)
capped at MaxDelay
```

**Example:**
- Attempt 1: 1s delay
- Attempt 2: 2s delay
- Attempt 3: 4s delay
- Attempt 4: 8s delay
- Attempt 5: 16s delay (capped at MaxDelay)

### 5. Configuration System (`pkg/config`)

YAML-based configuration with validation.

**Structure:**
```yaml
version: "1.0"

agents:
  - id: claude-1
    type: claude
    name: Claude
    prompt: "You are a helpful assistant"
    model: "claude-sonnet-4.5"
    rate_limit: 10.0
    rate_limit_burst: 5

orchestrator:
  mode: round-robin
  max_turns: 10
  turn_timeout: 30s
  response_delay: 1s
  initial_prompt: "Welcome!"

logging:
  enabled: true
  chat_log_dir: ~/.agentpipe/chats
  log_format: json
  show_metrics: true
```

**Features:**
- Schema validation
- Default value application
- Duplicate ID detection
- Type safety
- Hot-reload support (planned)

### 6. Logging System (`pkg/logger`)

Dual-output logging with metrics support.

**Outputs:**
1. **Console/TUI**: Real-time display with color coding
2. **File**: Persistent storage in text or JSON format

**Features:**
- Agent-specific color coding
- Metrics display (duration, tokens, cost)
- Text wrapping for terminal width
- JSON format for machine parsing
- Message history preservation

**Log Formats:**

Text:
```
[Agent1|250ms|150t|0.0045] Response message here
```

JSON:
```json
{
  "agent_id": "agent-1",
  "agent_name": "Agent1",
  "content": "Response message",
  "timestamp": 1234567890,
  "role": "agent",
  "metrics": {
    "duration": "250ms",
    "input_tokens": 100,
    "output_tokens": 50,
    "total_tokens": 150,
    "cost": 0.0045
  }
}
```

### 7. Error Handling (`pkg/errors`)

Structured error types for better error handling.

**Error Types:**
- `AgentError`: Agent-specific failures
- `ConfigError`: Configuration issues
- `InitializationError`: Setup failures
- `CommunicationError`: Network/IPC issues
- `ValidationError`: Invalid input
- `OrchestratorError`: Orchestration failures

**Benefits:**
- Error wrapping with context
- Type-safe error handling
- Better error messages
- Unwrap support for error chains

### 8. TUI (`pkg/tui`)

Terminal User Interface built with Bubble Tea.

**Features:**
- Three-panel layout (agents, conversation, input)
- Real-time message display
- Agent status indicators
- Metrics display
- Modal system for details
- User participation mode

**Keybindings:**
- `q`: Quit
- `u`: User participation mode
- Arrow keys: Navigate panels
- Enter: View details

## Data Flow

### Message Flow

```
1. User Input / Initial Prompt
        ↓
2. Orchestrator selects next agent
        ↓
3. Rate limiter check
        ↓
4. Agent receives message history
        ↓
5. Agent calls external CLI
        ↓
6. Response returned (with retry if failed)
        ↓
7. Metrics calculated (tokens, cost, duration)
        ↓
8. Message added to history
        ↓
9. Logger writes to file and console
        ↓
10. TUI updates display
        ↓
11. Repeat from step 2
```

### Configuration Flow

```
1. Load YAML config file
        ↓
2. Parse and unmarshal
        ↓
3. Validate configuration
        ↓
4. Apply defaults
        ↓
5. Create orchestrator
        ↓
6. Initialize agents
        ↓
7. Health check agents
        ↓
8. Start conversation
```

## Design Patterns

### 1. Interface-Based Design
All agents implement the `Agent` interface, enabling polymorphism and easy extension.

### 2. Factory Pattern
Agent creation is abstracted through factory functions based on agent type.

### 3. Strategy Pattern
Different orchestration modes implement different turn-taking strategies.

### 4. Observer Pattern
Logger acts as observer of orchestrator events.

### 5. Command Pattern
CLI commands encapsulate operations with clean separation.

### 6. Builder Pattern
Configuration objects are built incrementally with validation.

## Concurrency Model

### Thread Safety

**Orchestrator:**
- `sync.RWMutex` for message history access
- Concurrent-safe agent calls
- Context-based cancellation

**Rate Limiter:**
- `sync.Mutex` for token bucket operations
- Lock-free reads where possible
- Parallel-safe operations

**Logger:**
- Concurrent writes to file and console
- Buffered I/O for performance
- No shared mutable state

### Context Propagation

All operations support `context.Context` for:
- Timeout enforcement
- Cancellation propagation
- Request-scoped values

## Performance Considerations

### Optimizations

1. **Message Copying**
   - Copy-on-read for message history
   - Prevents data races without excessive locking

2. **Token Estimation**
   - Simple word-based heuristic
   - O(n) complexity, very fast
   - Good enough for cost estimation

3. **Rate Limiting**
   - Minimal overhead (~60ns per check)
   - Zero cost when disabled
   - Lock-free fast path for available tokens

4. **Configuration**
   - Parsed once at startup
   - Cached in memory
   - O(1) access for agent configs

### Scalability

- **Agents**: Tested with up to 10 agents
- **Message History**: Efficient with 1000+ messages
- **Concurrent Operations**: Thread-safe with parallel agent calls
- **Rate Limiting**: Supports high throughput (100k+ req/s)

## Extension Points

### Adding New Agents

1. Implement `Agent` interface
2. Add adapter in `pkg/adapters/`
3. Register in agent factory
4. Add health check logic
5. Update documentation

### Adding New Orchestration Modes

1. Add new mode constant in `pkg/orchestrator`
2. Implement mode-specific logic
3. Add tests for new mode
4. Update configuration schema

### Adding New Features

- Middleware: Hook into message processing
- Plugins: Dynamic agent loading
- Metrics: Export to Prometheus/StatsD
- Persistence: Save/resume conversations

## Security Considerations

### Input Validation
- All user input is validated before processing
- Agent responses are sanitized
- Path traversal prevention in file operations

### Resource Limits
- Turn timeouts prevent infinite conversations
- Rate limiting prevents API abuse
- Memory limits through bounded history

### Secrets Management
- API keys passed to agent CLIs
- No secrets in logs
- Secure temporary file handling

## Testing Strategy

### Unit Tests (86 tests)
- Component isolation
- Mock dependencies
- Edge case coverage

### Integration Tests (15 tests)
- End-to-end flows
- Multi-component interaction
- Error scenarios

### Benchmark Tests (25+ benchmarks)
- Performance baselines
- Regression detection
- Scalability validation

### Test Coverage
- Orchestrator: 100%
- Adapters: 100%
- Rate Limiter: 100%
- Logger: 95%+
- Config: 90%+

## Future Architecture

### Planned Enhancements

1. **Plugin System**: Dynamic agent loading
2. **Middleware**: Request/response processing pipeline
3. **Metrics**: Prometheus exporter
4. **Persistence**: Conversation database
5. **Streaming**: Real-time response streaming
6. **Distributed**: Multi-node orchestration
7. **Hot Reload**: Configuration updates without restart

### API Evolution

Current: CLI-based
Future: REST API + gRPC for programmatic access
