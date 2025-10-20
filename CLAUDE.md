# AgentPipe Project Memory

## Project Overview
AgentPipe is a CLI and TUI application that orchestrates conversations between multiple AI agent CLIs (Claude, Gemini, Qwen, Codex, Ollama). It allows different AI tools to communicate in a shared "room" with various conversation modes.

## Key Technical Details

### Go Version
- **IMPORTANT**: Requires Go 1.24+ (go.mod specifies 1.24.0)
- GitHub Actions workflows must use Go 1.24
- All dependencies are compatible with Go 1.24

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
- GitHub Action version parameter: `version: latest` (downloads latest golangci-lint v1.x)
- **IMPORTANT**: Config file (`.golangci.yml`) is compatible with golangci-lint v1.64.8 (GitHub Actions)
- **Local linting**: If you have golangci-lint v2.x installed locally, it will not work with the current config
  - Either downgrade to v1.x: `brew install golangci/tap/golangci-lint@1`
  - Or rely on GitHub Actions for linting verification
- Configuration structure: `linters-settings:` for linter config, `issues.exclude-rules:` for exclusions
- Cognitive complexity threshold: 30
- Excluded from complexity checks: pkg/tui/, pkg/adapters/, pkg/orchestrator/

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

2. **Windows test failures - multiline commands**
   - Issue: Multiline commands break on Windows
   - Solution: Use single-line test command

3. **Windows test failures - timer resolution**
   - Issue: `time.Since()` can return 0 for very fast operations due to Windows timer granularity (~15.6ms default)
   - Solution: Add delays (≥20ms) to mock agents in tests to ensure measurable durations
   - Example: `agent.sendDelay = 20 * time.Millisecond` in integration tests
   - Windows timer is much coarser than Unix (typically 1ms or better)

4. **Homebrew formula updates failing**
   - Issue: GitHub Actions bot lacks permissions
   - Solution: Add `HOMEBREW_TAP_TOKEN` secret with repo scope

5. **Linting errors**
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

## Quality Requirements

**IMPORTANT**: Before committing any changes, ALL of the following checks MUST pass:

1. **Linting**: `golangci-lint run --timeout=5m` (Note: Local v2.x won't work; verify via GitHub Actions)
2. **Testing**: `go test -v -race ./...`
3. **Build**: `go build -o agentpipe .`

No code should be committed if any of these checks fail. This ensures code quality, prevents regressions, and maintains CI/CD pipeline health.

**Note**: If local linting fails due to golangci-lint version mismatch, ensure tests and build pass locally, then verify linting on GitHub Actions.

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
- Downgraded to Go 1.24.0 (for golangci-lint compatibility)
- Fixed golangci-lint config for v1.64.8 (GitHub Actions)
- Added skip logic to adapter tests for missing CLI tools
- Increased health check timeout: 2s → 5s
- Fixed Windows CI test command
- Fixed Homebrew formula path: Formulae → Formula
- Added badges to README
- Fixed all linting issues for CI
- Before releasing, be sure that lints, tests, and build pass.