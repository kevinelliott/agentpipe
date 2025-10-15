# AgentPipe Project Memory

## Project Overview
AgentPipe is a CLI and TUI application that orchestrates conversations between multiple AI agent CLIs (Claude, Gemini, Qwen, Codex, Ollama). It allows different AI tools to communicate in a shared "room" with various conversation modes.

## Key Technical Details

### Go Version
- **IMPORTANT**: Requires Go 1.24+ (go.mod specifies 1.24.0)
- GitHub Actions workflows must use Go 1.24
- Some dependencies (bubbletea v1.3.9) require Go 1.24+

### Health Check Configuration
- Default timeout: 5 seconds (increased from 2 seconds)
- Claude CLI needs longer startup time
- Flag: `--health-check-timeout` to customize
- Flag: `--skip-health-check` to bypass

### Directory Structure
- Chat logs: `~/.agentpipe/chats/` (default)
- Homebrew formula: `Formula/` (NOT Formulae/)
- Config examples: `examples/`

### CI/CD Configuration

#### Linting
- Use golangci-lint-action@v6 (NOT v8)
- Use `version: v2` for golangci-lint (v2 format)
- Configuration file format: golangci-lint v2 (with `version: "2"` string)
- Formatters configured in separate `formatters:` section
- Exclusions under `linters.exclusions.rules:` (not `issues.exclude-rules`)
- Cognitive complexity threshold: 30
- Excluded from complexity checks: pkg/tui/, pkg/adapters/

#### Testing
- Windows test fix: Single-line command (no multiline)
- Command: `go test -v -race ./...`
- No coverage profile to avoid Windows parsing issues

#### Releases
- Triggered on tags: `v[0-9]+.[0-9]+.[0-9]+`
- Requires `HOMEBREW_TAP_TOKEN` secret for formula updates
- Token needs `repo` scope for cross-repo access

### Common Issues & Fixes

1. **Claude CLI health check timeout**
   - Solution: Increased default timeout to 5 seconds
   - User can use `--health-check-timeout 10` for longer

2. **Windows test failures**
   - Issue: Multiline commands break on Windows
   - Solution: Use single-line test command

3. **Homebrew formula updates failing**
   - Issue: GitHub Actions bot lacks permissions
   - Solution: Add `HOMEBREW_TAP_TOKEN` secret with repo scope

4. **Linting errors**
   - Empty branches: Add comment or `_ = err`
   - Imports: Use `goimports -local github.com/kevinelliott/agentpipe`
   - Deprecated methods: Updated viewport scroll methods

### Agent Adapters
Each agent adapter must implement:
- `Initialize(config)` - Setup with config
- `IsAvailable()` - Check if CLI exists
- `HealthCheck(ctx)` - Verify CLI works
- `SendMessage(ctx, messages)` - Send and receive
- `GetMetrics()` - Return usage metrics

### TUI Features
- Three panels: agents list, conversation, user input
- Color-coded agent messages with badges
- Real-time metrics display (duration, tokens, cost)
- Modal system for agent details
- User participation with 'u' key

### Configuration
YAML config supports:
- Multiple agents with custom prompts
- Orchestrator modes: round-robin, reactive, free-form
- Logging configuration
- Turn limits and timeouts

## Development Commands

```bash
# Build
go build -o agentpipe .

# Test
go test -v -race ./...

# Lint
golangci-lint run --timeout=5m

# Format
gofmt -w .
goimports -local github.com/kevinelliott/agentpipe -w .

# Run with TUI
./agentpipe run -t -c examples/brainstorm.yaml

# Check agent health
./agentpipe doctor
```

## Recent Changes Log
- Increased health check timeout: 2s → 5s
- Fixed Windows CI test command
- Updated to Go 1.24 requirement
- Fixed Homebrew formula path: Formulae → Formula
- Added badges to README
- Fixed all linting issues for CI