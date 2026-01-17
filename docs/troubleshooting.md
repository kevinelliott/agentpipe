# Troubleshooting Guide

This guide helps diagnose and resolve common issues with AgentPipe.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Configuration Issues](#configuration-issues)
- [Agent Communication Issues](#agent-communication-issues)
- [Performance Issues](#performance-issues)
- [TUI Issues](#tui-issues)
- [Logging Issues](#logging-issues)
- [Build and Development Issues](#build-and-development-issues)
- [Getting Help](#getting-help)

## Installation Issues

### Issue: `go: version 1.25 required`

**Symptoms:**
```
go: version 1.25 required
```

**Solution:**
```bash
# Check Go version
go version

# Install Go 1.25+ from https://golang.org/dl/
# Or use version manager
brew install go@1.25  # macOS

# Verify installation
go version  # Should show 1.25 or higher
```

###

 Issue: `command not found: agentpipe`

**Symptoms:**
```
bash: agentpipe: command not found
```

**Solutions:**

1. **If installed via Homebrew:**
```bash
# Check if installed
brew list agentpipe

# If not installed
brew install kevinelliott/tap/agentpipe

# Check PATH
echo $PATH | grep -o '/usr/local/bin'
```

2. **If built from source:**
```bash
# Build and install
go install github.com/kevinelliott/agentpipe@latest

# Ensure GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Add to ~/.bashrc or ~/.zshrc
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
```

### Issue: Permission denied when running

**Symptoms:**
```
permission denied: ./agentpipe
```

**Solution:**
```bash
# Make executable
chmod +x agentpipe

# Or run with go
go run . <command>
```

## Configuration Issues

### Issue: `configuration validation failed`

**Symptoms:**
```
Error: configuration validation failed: no agents configured
```

**Solutions:**

1. **Check YAML syntax:**
```bash
# Validate YAML
cat agentpipe.yaml | python -c 'import yaml, sys; yaml.safe_load(sys.stdin)'

# Or use yq
yq eval agentpipe.yaml
```

2. **Verify required fields:**
```yaml
version: "1.0"  # Required

agents:  # At least one agent required
  - id: agent-1  # Required, must be unique
    type: claude  # Required
    name: Claude  # Required
    prompt: "You are helpful"  # Required

orchestrator:
  mode: round-robin  # Required (round-robin, reactive, or free-form)
```

3. **Check for duplicate agent IDs:**
```yaml
# BAD: Duplicate IDs
agents:
  - id: agent-1
    type: claude
  - id: agent-1  # ERROR: Duplicate!
    type: gemini

# GOOD: Unique IDs
agents:
  - id: claude-1
    type: claude
  - id: gemini-1
    type: gemini
```

### Issue: `invalid mode`

**Symptoms:**
```
Error: unknown conversation mode: round-robbin
```

**Solution:**
```yaml
# Valid modes are:
orchestrator:
  mode: round-robin  # Fixed order
  # OR
  mode: reactive     # Random (no repeats)
  # OR
  mode: free-form    # All agents per turn
```

### Issue: `failed to parse duration`

**Symptoms:**
```
Error: time: invalid duration "30"
```

**Solution:**
```yaml
# BAD: Missing unit
orchestrator:
  turn_timeout: 30
  response_delay: 1

# GOOD: Include unit
orchestrator:
  turn_timeout: 30s   # s, m, h
  response_delay: 1s
```

## Agent Communication Issues

### Issue: Agent CLI not found

**Symptoms:**
```
Error: agent claude is not available: claude CLI not found
```

**Solutions:**

1. **Check if CLI is installed:**
```bash
# For Claude
which claude

# For Gemini
which gemini

# For GitHub Copilot
gh copilot --help

# For Cursor
which cursor-agent
```

2. **Install missing CLI:**
```bash
# Claude
brew install anthropics/claude/claude

# GitHub Copilot
gh extension install github/gh-copilot

# Others: See agent-specific documentation
```

3. **Run doctor command:**
```bash
agentpipe doctor

# Output shows which agents are available
# ✅ claude: available
# ❌ gemini: CLI not found
```

### Issue: Agent health check timeout

**Symptoms:**
```
Error: health check failed for agent claude: context deadline exceeded
```

**Solutions:**

1. **Increase health check timeout:**
```bash
# Default is 5 seconds
agentpipe doctor --health-check-timeout 10
```

2. **Skip health check:**
```bash
agentpipe run --skip-health-check -c config.yaml
```

3. **Check agent CLI manually:**
```bash
# Test Claude CLI
echo "Hello" | claude

# Check for errors
claude --version
```

### Issue: Agent timeout during conversation

**Symptoms:**
```
[Error] Agent claude failed: context deadline exceeded
[Info] Continuing conversation with remaining agents...
```

**Solutions:**

1. **Increase turn timeout:**
```yaml
orchestrator:
  turn_timeout: 60s  # Increase from default 30s
```

2. **Enable retries:**
```yaml
orchestrator:
  max_retries: 3
  retry_initial_delay: 1s
  retry_max_delay: 30s
  retry_multiplier: 2.0
```

3. **Check network connectivity:**
```bash
# Test internet connection
ping anthropic.com

# Check if firewall blocking
sudo lsof -i -P | grep claude
```

### Issue: Rate limiting errors

**Symptoms:**
```
Error: rate limit wait failed: context deadline exceeded
```

**Solutions:**

1. **Adjust rate limits:**
```yaml
agents:
  - id: claude-1
    type: claude
    rate_limit: 5.0  # Reduce from 10.0
    rate_limit_burst: 2  # Reduce burst
```

2. **Disable rate limiting:**
```yaml
agents:
  - id: claude-1
    type: claude
    rate_limit: 0  # Unlimited
```

3. **Increase turn timeout:**
```yaml
orchestrator:
  turn_timeout: 120s  # Allow more time for rate limiting
```

## Performance Issues

### Issue: Slow conversation speed

**Solutions:**

1. **Reduce response delay:**
```yaml
orchestrator:
  response_delay: 500ms  # Reduce from default 1s
```

2. **Disable logging:**
```yaml
logging:
  enabled: false
```

3. **Use lightweight agents:**
```yaml
agents:
  - type: claude
    model: claude-3-5-haiku-20241022  # Faster than opus/sonnet
```

4. **Check system resources:**
```bash
# CPU usage
top -o cpu

# Memory usage
top -o mem

# Disk I/O
iostat -d 1
```

### Issue: High memory usage

**Solutions:**

1. **Limit conversation length:**
```yaml
orchestrator:
  max_turns: 10  # Limit turn count
```

2. **Monitor memory:**
```bash
# Run with profiling
go run -memprofile=mem.out . run -c config.yaml

# Analyze
go tool pprof mem.out
```

3. **Check for memory leaks:**
```bash
# Run with race detector
go run -race . run -c config.yaml
```

## TUI Issues

### Issue: TUI not rendering correctly

**Symptoms:**
- Garbled text
- Missing characters
- Incorrect colors

**Solutions:**

1. **Check terminal compatibility:**
```bash
# Test terminal type
echo $TERM

# Should be xterm-256color or similar
export TERM=xterm-256color
```

2. **Use non-TUI mode:**
```bash
# Run without TUI
agentpipe run -c config.yaml  # Without -t flag
```

3. **Update terminal:**
```bash
# macOS: Update Terminal.app or use iTerm2
brew install iterm2

# Linux: Try different terminal
sudo apt install terminator
```

### Issue: TUI freezes or hangs

**Solutions:**

1. **Check for blocking operations:**
```bash
# Send interrupt signal
Ctrl+C

# Force quit if needed
kill -9 $(pgrep agentpipe)
```

2. **Increase timeouts:**
```yaml
orchestrator:
  turn_timeout: 60s
```

3. **Run with debug output:**
```bash
# Enable verbose logging
agentpipe run -c config.yaml 2>&1 | tee debug.log
```

### Issue: Cannot scroll in TUI

**Solution:**
- Use mouse wheel to scroll in conversation panel
- Use arrow keys to navigate between panels
- Press `q` to quit TUI

## Logging Issues

### Issue: No log files created

**Symptoms:**
```
Expected log file in ~/.agentpipe/chats/ but directory is empty
```

**Solutions:**

1. **Check logging config:**
```yaml
logging:
  enabled: true  # Must be true
  chat_log_dir: ~/.agentpipe/chats  # Check path
```

2. **Verify directory permissions:**
```bash
# Check directory exists and is writable
ls -la ~/.agentpipe/chats/

# Create if missing
mkdir -p ~/.agentpipe/chats
chmod 755 ~/.agentpipe/chats
```

3. **Check disk space:**
```bash
df -h
```

### Issue: Log files growing too large

**Solutions:**

1. **Limit conversation length:**
```yaml
orchestrator:
  max_turns: 20
```

2. **Use JSON format (more compact):**
```yaml
logging:
  log_format: json  # Instead of text
```

3. **Rotate logs manually:**
```bash
# Move old logs
mv ~/.agentpipe/chats/*.txt ~/.agentpipe/chats/archive/

# Or delete old logs
find ~/.agentpipe/chats/ -name "*.txt" -mtime +30 -delete
```

### Issue: Cannot read log files

**Symptoms:**
```
Error: permission denied reading log file
```

**Solution:**
```bash
# Fix permissions
chmod 644 ~/.agentpipe/chats/*.txt

# Change ownership
sudo chown $USER ~/.agentpipe/chats/*.txt
```

## Build and Development Issues

### Issue: Tests failing

**Solutions:**

1. **Clean test cache:**
```bash
go clean -testcache
go test ./...
```

2. **Check Go version:**
```bash
go version  # Must be 1.25+
```

3. **Update dependencies:**
```bash
go mod tidy
go mod download
```

4. **Run specific test:**
```bash
go test -v -run TestName ./pkg/package/
```

### Issue: Linter errors

**Solutions:**

1. **Run linter:**
```bash
golangci-lint run --timeout=5m
```

2. **Auto-fix issues:**
```bash
golangci-lint run --fix
```

3. **Format code:**
```bash
gofmt -w .
goimports -local github.com/kevinelliott/agentpipe -w .
```

### Issue: Module issues

**Solutions:**

1. **Verify go.mod:**
```bash
go mod verify
```

2. **Download dependencies:**
```bash
go mod download
```

3. **Clean and rebuild:**
```bash
go clean -modcache
go mod tidy
go build ./...
```

## Debugging Tips

### Enable Verbose Logging

```bash
# Set log level
export LOG_LEVEL=debug
agentpipe run -c config.yaml

# Or redirect stderr
agentpipe run -c config.yaml 2> debug.log
```

### Reproduce Issues

```bash
# Minimal reproduction
agentpipe run -c minimal-config.yaml

# With single agent
# Create config with just one agent to isolate issue
```

### Collect Diagnostic Information

```bash
# System info
uname -a
go version
agentpipe version

# Check agent CLIs
which claude
which gemini
which gh

# Test configuration
agentpipe doctor --health-check-timeout 10

# Save output
agentpipe run -c config.yaml 2>&1 | tee diagnostics.log
```

## Getting Help

### Before Asking for Help

1. Search [existing issues](https://github.com/kevinelliott/agentpipe/issues)
2. Check this troubleshooting guide
3. Run `agentpipe doctor`
4. Try minimal configuration
5. Collect diagnostic information

### Creating an Issue

Include:
- AgentPipe version: `agentpipe version`
- Go version: `go version`
- Operating system: `uname -a`
- Configuration file (sanitized)
- Error message (full output)
- Steps to reproduce
- Expected vs actual behavior

### Community Support

- GitHub Issues: Bug reports and feature requests
- GitHub Discussions: Questions and general discussion
- Documentation: README and docs/ directory

### Security Issues

For security vulnerabilities, email security@example.com instead of creating a public issue.

## Common Error Messages

### `no agents configured`
→ Add at least one agent to your configuration

### `agent not available`
→ Install the agent's CLI tool

### `context deadline exceeded`
→ Increase timeout or enable retries

### `rate limit wait failed`
→ Adjust rate limits or increase timeout

### `failed to load config`
→ Check YAML syntax and file path

### `duplicate agent ID`
→ Ensure all agent IDs are unique

### `invalid mode`
→ Use: round-robin, reactive, or free-form

### `health check failed`
→ Increase timeout or skip health check

### `permission denied`
→ Check file permissions and ownership

### `command not found`
→ Ensure AgentPipe is in PATH

## Still Having Issues?

If you've tried the above solutions and still have problems:

1. Create a minimal reproduction case
2. Collect all diagnostic information
3. Open an issue on GitHub
4. Include configuration (sanitized)
5. Describe expected vs actual behavior

We're here to help! 🎉
