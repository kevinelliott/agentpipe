# Add Support for Aider CLI Agent

## Overview
Add support for [Aider](https://github.com/paul-gauthier/aider), the most popular open-source AI coding assistant with git integration.

## Why Add Aider?
- ⭐ **18,000+ GitHub stars** - Extremely popular in developer community
- 🎯 **Git-aware coding** - Unique capability not present in current agents
- 💬 **Excellent CLI interface** - Well-documented stdin/stdout support
- 🔄 **Auto-commits changes** - Streamlined workflow
- 📝 **Multi-file editing** - Handles complex refactoring tasks
- 🌐 **Multi-LLM support** - Works with GPT-4, Claude, etc.

## Current AgentPipe Support
AgentPipe currently supports 14 CLI-based agents:
- Amp, Claude, Codex, Copilot, Crush, Cursor, Factory, Gemini, Groq, Kimi, OpenCode, Qoder, Qwen, Ollama

Aider would be a valuable addition as a **git-aware coding specialist**.

## Technical Details

### Installation
```bash
pip install aider-chat
# or
pipx install aider-chat
```

### CLI Interface
```bash
# Non-interactive mode
aider --message "refactor this function to use async/await"

# Stdin support
echo "add error handling to main.py" | aider

# Model selection
aider --model gpt-4 --message "optimize this code"

# Git integration (automatic)
aider --auto-commits
```

### Integration Approach
- **Adapter Location:** `pkg/adapters/aider.go`
- **Command Detection:** `which aider`
- **Execution Mode:** Non-interactive via `--message` flag or stdin
- **Model Support:** Via `--model` flag (optional)
- **Pattern:** Similar to Claude adapter (simple stdin/stdout)

### Example Agent Configuration
```yaml
agents:
  - id: aider-dev
    type: aider
    name: "Git-Aware Developer"
    prompt: "You are an expert developer who makes careful, well-tested changes"
    model: gpt-4  # optional
    temperature: 0.7
```

## Implementation Checklist

### Phase 1: Basic Integration
- [ ] Create `pkg/adapters/aider.go`
  - [ ] Implement `Initialize()` method
  - [ ] Implement `IsAvailable()` method (check for `aider` in PATH)
  - [ ] Implement `HealthCheck()` method
  - [ ] Implement `GetCLIVersion()` method
- [ ] Add to agent registry in `init()` function
- [ ] Create unit tests in `pkg/adapters/aider_test.go`

### Phase 2: Message Handling
- [ ] Implement `SendMessage()` method
  - [ ] Filter relevant messages (exclude own messages)
  - [ ] Build structured prompt (3-part pattern)
  - [ ] Execute `aider --message` with prompt
  - [ ] Parse response
  - [ ] Handle errors
- [ ] Implement `StreamMessage()` method (if needed)
- [ ] Add comprehensive logging

### Phase 3: Advanced Features
- [ ] Model selection support (via `--model` flag)
- [ ] Temperature configuration (if supported)
- [ ] Git integration configuration
- [ ] Auto-commit settings
- [ ] Multi-file editing support

### Phase 4: Testing & Documentation
- [ ] Integration tests
- [ ] Add to `doctor` command detection
- [ ] Update README.md
  - [ ] Add to "Supported AI Agents" section
  - [ ] Add installation instructions
  - [ ] Add example configuration
- [ ] Create example config: `examples/aider-coding.yaml`
- [ ] Add troubleshooting section
- [ ] Update CHANGELOG.md

### Phase 5: Quality Assurance
- [ ] Run linter: `golangci-lint run --timeout=5m`
- [ ] Run tests: `go test -v -race ./...`
- [ ] Build: `go build -o agentpipe .`
- [ ] Manual testing:
  - [ ] Doctor command detection
  - [ ] Single message test
  - [ ] Multi-turn conversation
  - [ ] Round-robin mode
  - [ ] Reactive mode
  - [ ] Error handling
  - [ ] Timeout scenarios

## Example Use Cases

### Use Case 1: Code Review Team
```yaml
# Multi-agent code review with git-aware Aider
agents:
  - type: aider
    name: "Refactorer"
    prompt: "Focus on improving code structure and patterns"
  
  - type: claude
    name: "Reviewer"
    prompt: "Review code for bugs and edge cases"
  
  - type: qoder
    name: "Documenter"
    prompt: "Add comprehensive documentation"

orchestrator:
  mode: round-robin
  initial_prompt: "Review and improve the authentication module"
```

### Use Case 2: Git-Aware Development
```yaml
# Leverage Aider's git integration
agents:
  - type: aider
    name: "Git Developer"
    prompt: "Make changes and commit them with clear messages"
    model: gpt-4

orchestrator:
  mode: reactive
  initial_prompt: "Add input validation to all API endpoints"
```

## Success Criteria
- [ ] Aider CLI detected by `agentpipe doctor`
- [ ] Can send and receive messages in conversations
- [ ] Works in round-robin, reactive, and free-form modes
- [ ] Proper error handling and timeouts
- [ ] Documentation complete
- [ ] All tests passing
- [ ] Linting clean

## Related Issues
- Research document: `docs/research-ai-agent-clis.md`
- Summary: `docs/missing-ai-agents-summary.md`

## References
- Aider GitHub: https://github.com/paul-gauthier/aider
- Aider Docs: https://aider.chat/
- AgentPipe Agent Interface: `pkg/agent/agent.go`
- Existing Adapter Examples: `pkg/adapters/claude.go`, `pkg/adapters/amp.go`

## Priority
**HIGH** - Most requested missing agent based on research

## Estimated Effort
**LOW-MEDIUM** - 1-2 days
- Simple CLI interface (stdin/stdout)
- Well-documented tool
- Similar to existing adapters

## Labels
- `enhancement`
- `agent-support`
- `high-priority`
- `good-first-issue` (for experienced contributors)
