# Development Guide

This guide provides detailed information for developing AgentPipe, including setup, workflows, debugging, and best practices.

## Table of Contents

- [Environment Setup](#environment-setup)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Debugging](#debugging)
- [Code Generation](#code-generation)
- [Performance Profiling](#performance-profiling)
- [Troubleshooting](#troubleshooting)

## Environment Setup

### Required Tools

```bash
# Go 1.24+ (required)
go version  # Must be 1.24 or higher

# golangci-lint (required for linting)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goimports (required for formatting)
go install golang.org/x/tools/cmd/goimports@latest

# Optional but recommended
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/godoc@latest
go install github.com/rakyll/gotest@latest
```

### Editor Setup

#### VS Code

Recommended extensions:
- `golang.go`: Official Go extension
- `GitHub.copilot`: AI assistance
- `streetsidesoftware.code-spell-checker`: Spell checking

`.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "goimports",
  "go.importOnSave": true,
  "go.testFlags": ["-v", "-race"],
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

#### GoLand/IntelliJ

1. Enable golangci-lint: Preferences → Tools → golangci-lint
2. Set Go version: Preferences → Go → GOROOT
3. Enable goimports: Preferences → Tools → File Watchers

### Agent CLIs Setup

For local testing, install the agent CLIs:

```bash
# Claude CLI
brew install anthropics/claude/claude

# Gemini CLI (if available)
# Installation method varies

# GitHub Copilot CLI
gh extension install github/gh-copilot

# Cursor CLI
# Installation from Cursor IDE

# Others as needed
```

## Project Structure

```
agentpipe/
├── cmd/                    # Command-line interface
│   ├── root.go            # Root command
│   ├── run.go             # Run conversation
│   ├── doctor.go          # Health check
│   ├── init.go            # Initialize config
│   └── version.go         # Version command
│
├── pkg/                    # Core library code
│   ├── agent/             # Agent interfaces
│   │   └── agent.go
│   ├── adapters/          # Agent implementations
│   │   ├── claude.go
│   │   ├── gemini.go
│   │   ├── copilot.go
│   │   └── ...
│   ├── orchestrator/      # Conversation orchestration
│   │   ├── orchestrator.go
│   │   └── orchestrator_test.go
│   ├── config/            # Configuration
│   │   ├── config.go
│   │   └── config_test.go
│   ├── logger/            # Logging system
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── ratelimit/         # Rate limiting
│   │   ├── ratelimit.go
│   │   └── ratelimit_test.go
│   ├── errors/            # Error types
│   │   ├── errors.go
│   │   └── errors_test.go
│   ├── tui/               # Terminal UI
│   │   └── tui.go
│   └── utils/             # Utilities
│       ├── tokens.go
│       └── cost.go
│
├── test/                   # Tests
│   ├── integration/       # Integration tests
│   │   ├── conversation_test.go
│   │   └── error_scenarios_test.go
│   └── benchmark/         # Benchmark tests
│       ├── utils_bench_test.go
│       ├── ratelimit_bench_test.go
│       ├── orchestrator_bench_test.go
│       └── config_bench_test.go
│
├── examples/              # Example configurations
│   ├── brainstorm.yaml
│   ├── debate.yaml
│   └── code-review.yaml
│
├── docs/                  # Documentation
│   ├── architecture.md
│   ├── contributing.md
│   ├── development.md
│   └── configuration.md
│
├── internal/              # Internal packages
│   └── version/          # Version info
│
├── .github/               # GitHub configuration
│   └── workflows/        # CI/CD workflows
│       ├── test.yml
│       ├── lint.yml
│       └── release.yml
│
├── .golangci.yml          # Linter configuration
├── go.mod                 # Go modules
├── go.sum                 # Dependency checksums
├── CLAUDE.md              # Project memory
└── README.md              # Main documentation
```

## Development Workflow

### 1. Iterative Development

```bash
# Watch mode for tests (requires gotest)
gotest -watch ./pkg/orchestrator/

# Or use entr for file watching
ls pkg/**/*.go | entr -c go test ./pkg/...

# Quick feedback loop
while true; do
    clear
    go test -run TestYourTest ./pkg/yourpackage/
    sleep 1
done
```

### 2. TDD Workflow

```bash
# 1. Write failing test
go test -run TestNewFeature ./pkg/yourpackage/

# 2. Implement feature
# Edit code...

# 3. Run tests until passing
go test -run TestNewFeature ./pkg/yourpackage/

# 4. Refactor
# Clean up code...

# 5. Run full test suite
go test ./...
```

### 3. Feature Branch Workflow

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make changes and test
go test ./...
golangci-lint run

# Commit incrementally
git add -p  # Stage hunks interactively
git commit -m "feat: add feature part 1"

# Push and create PR
git push -u origin feature/my-feature
```

## Building and Running

### Development Build

```bash
# Build binary
go build -o agentpipe .

# Build with debug info
go build -gcflags="all=-N -l" -o agentpipe .

# Build for specific OS
GOOS=linux GOARCH=amd64 go build -o agentpipe-linux .
GOOS=darwin GOARCH=arm64 go build -o agentpipe-mac .
GOOS=windows GOARCH=amd64 go build -o agentpipe.exe .
```

### Running Locally

```bash
# Run without building
go run . run -c examples/brainstorm.yaml

# Run with TUI
go run . run -t -c examples/brainstorm.yaml

# Run doctor command
go run . doctor

# Run init command
go run . init -o test-config.yaml
```

### Hot Reload Development

```bash
# Install air (hot reload tool)
go install github.com/cosmtrek/air@latest

# Create .air.toml
cat > .air.toml << 'EOF'
[build]
  cmd = "go build -o bin/agentpipe ."
  bin = "bin/agentpipe"
  include_ext = ["go"]
  exclude_dir = ["test", "vendor"]
EOF

# Run with hot reload
air
```

## Testing

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/orchestrator/

# Specific test
go test -run TestOrchestrator ./pkg/orchestrator/

# With coverage
go test -cover ./...

# With race detection
go test -race ./...

# Verbose output
go test -v ./...

# Integration tests
go test ./test/integration/

# Benchmarks
go test -bench=. ./test/benchmark/
go test -bench=BenchmarkEstimateTokens ./test/benchmark/

# Short mode (skip slow tests)
go test -short ./...
```

### Writing Tests

#### Unit Test Example

```go
func TestRoundRobinMode(t *testing.T) {
    // Setup
    config := orchestrator.OrchestratorConfig{
        Mode: orchestrator.ModeRoundRobin,
        MaxTurns: 2,
    }
    orch := orchestrator.NewOrchestrator(config, io.Discard)

    // Execute
    agent1 := &MockAgent{id: "a1", name: "Agent1"}
    agent2 := &MockAgent{id: "a2", name: "Agent2"}
    orch.AddAgent(agent1)
    orch.AddAgent(agent2)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := orch.Start(ctx)

    // Verify
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if agent1.callCount != 2 {
        t.Errorf("expected agent1 called 2 times, got %d", agent1.callCount)
    }
}
```

#### Benchmark Example

```go
func BenchmarkEstimateTokens(b *testing.B) {
    text := "This is a test message for benchmarking token estimation."

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = utils.EstimateTokens(text)
    }
}
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# Show coverage by function
go tool cover -func=coverage.out

# Coverage with minimum threshold
go test -cover ./... | grep -E 'coverage:.*[0-9]+\.[0-9]+%' | awk '{if ($2 < 80.0) exit 1}'
```

## Debugging

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug tests
dlv test ./pkg/orchestrator/ -- -test.run TestOrchestrator

# Debug binary
dlv exec ./agentpipe -- run -c examples/brainstorm.yaml

# Attach to running process
dlv attach $(pgrep agentpipe)
```

### Common Delve Commands

```
break main.main              # Set breakpoint
continue                     # Continue execution
next                         # Step over
step                         # Step into
print variableName          # Print variable
stack                        # Show stack trace
goroutines                   # List goroutines
exit                         # Exit debugger
```

### VS Code Debugging

`.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Current File",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${file}"
    },
    {
      "name": "Debug AgentPipe",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}",
      "args": ["run", "-c", "examples/brainstorm.yaml"]
    },
    {
      "name": "Debug Tests",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/pkg/orchestrator"
    }
  ]
}
```

### Logging for Debugging

```go
// Add temporary debug logging
import "log"

log.Printf("DEBUG: variable = %+v\n", variable)
log.Printf("DEBUG: entering function with args: %v\n", args)

// Or use fmt for quick debugging
import "fmt"

fmt.Printf("DEBUG: %#v\n", complexStruct)
```

### Race Detector

```bash
# Run with race detector
go run -race . run -c examples/brainstorm.yaml

# Test with race detector
go test -race ./...

# Build with race detector
go build -race -o agentpipe .
```

## Code Generation

### Mocks

```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate mocks
mockgen -source=pkg/agent/agent.go -destination=pkg/agent/mock_agent.go -package=agent

# Use in tests
import "github.com/kevinelliott/agentpipe/pkg/agent"

mockCtrl := gomock.NewController(t)
defer mockCtrl.Finish()

mockAgent := agent.NewMockAgent(mockCtrl)
mockAgent.EXPECT().GetID().Return("test-id").AnyTimes()
```

### Stringer

```bash
# Generate String() methods for enums
//go:generate stringer -type=ConversationMode

go generate ./...
```

## Performance Profiling

### CPU Profiling

```bash
# Profile specific test
go test -cpuprofile=cpu.out -bench=BenchmarkEstimateTokens ./test/benchmark/

# Analyze profile
go tool pprof cpu.out
(pprof) top10
(pprof) web  # Visualize (requires graphviz)
```

### Memory Profiling

```bash
# Profile memory allocations
go test -memprofile=mem.out -bench=BenchmarkEstimateTokens ./test/benchmark/

# Analyze profile
go tool pprof mem.out
(pprof) top10
(pprof) list functionName
```

### Tracing

```bash
# Generate execution trace
go test -trace=trace.out ./pkg/orchestrator/

# View trace
go tool trace trace.out
```

### Benchstat Comparison

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks and save results
go test -bench=. -count=10 ./test/benchmark/ > old.txt

# Make changes...

# Run benchmarks again
go test -bench=. -count=10 ./test/benchmark/ > new.txt

# Compare
benchstat old.txt new.txt
```

## Troubleshooting

### Common Issues

#### Module Issues

```bash
# Clean module cache
go clean -modcache

# Update dependencies
go get -u ./...
go mod tidy

# Verify dependencies
go mod verify
```

#### Build Issues

```bash
# Clean build cache
go clean -cache

# Rebuild everything
go build -a ./...
```

#### Test Issues

```bash
# Clean test cache
go clean -testcache

# Run tests without cache
go test -count=1 ./...
```

### Performance Issues

```bash
# Check for goroutine leaks
go test -race ./...

# Profile slow tests
go test -timeout=30s -cpuprofile=cpu.out ./...

# Check memory usage
go test -memprofile=mem.out -run=TestSlowTest ./...
```

### Debugging Failed CI

```bash
# Run same checks as CI locally
./scripts/ci-checks.sh

# Or run individually
go test -race ./...
golangci-lint run --timeout=5m
go build ./...
```

## Best Practices

### Code Organization

1. Keep packages focused and cohesive
2. Avoid circular dependencies
3. Use internal/ for private packages
4. Group related functionality

### Performance

1. Profile before optimizing
2. Use benchmarks to measure improvements
3. Prefer simple code over premature optimization
4. Cache expensive computations

### Error Handling

1. Wrap errors with context
2. Use typed errors for important cases
3. Don't ignore errors
4. Fail fast and loudly

### Testing

1. Test behavior, not implementation
2. Use table-driven tests
3. Mock external dependencies
4. Test error paths

### Documentation

1. Document all exported APIs
2. Include examples in godoc
3. Keep README up to date
4. Update CHANGELOG for changes

## Additional Resources

- [Official Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
