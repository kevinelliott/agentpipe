# Add Support for Mentat CLI Agent

## Overview
Add support for [Mentat](https://github.com/AbanteAI/mentat), an AI coding assistant that writes code directly to files with git integration.

## Why Add Mentat?
- ⭐ **2,500+ GitHub stars** - Growing popularity in developer community
- 📝 **Direct file editing** - Unique workflow compared to other agents
- 💬 **CLI-first design** - Built for terminal use
- 🔄 **Git integration** - Automatic change tracking
- 🎯 **Multi-file changes** - Handles complex refactoring
- 🌐 **Multi-LLM support** - Works with various API providers

## Differentiation from Aider
While similar in concept, Mentat has a different approach:
- **Aider:** Interactive git commits, focuses on collaborative editing
- **Mentat:** Direct file writing, focuses on autonomous changes
- Both are valuable for different workflows

## Technical Details

### Installation
```bash
pip install mentat
```

### CLI Interface
```bash
# Interactive mode with message
mentat

# Via stdin
echo "refactor authentication to use JWT" | mentat

# Model selection
mentat --model gpt-4

# Specific files
mentat file1.py file2.py
```

### Integration Approach
- **Adapter Location:** `pkg/adapters/mentat.go`
- **Command Detection:** `which mentat`
- **Execution Mode:** Interactive with stdin
- **Model Support:** Via `--model` flag (optional)
- **Pattern:** Similar to Kimi adapter (interactive CLI with stdin)

### Example Agent Configuration
```yaml
agents:
  - id: mentat-dev
    type: mentat
    name: "File Editor"
    prompt: "You make precise, well-tested changes to code files"
    model: gpt-4  # optional
    temperature: 0.5
```

## Implementation Checklist

### Phase 1: Basic Integration
- [ ] Create `pkg/adapters/mentat.go`
  - [ ] Implement `Initialize()` method
  - [ ] Implement `IsAvailable()` method
  - [ ] Implement `HealthCheck()` method
  - [ ] Implement `GetCLIVersion()` method
- [ ] Add to agent registry
- [ ] Create unit tests

### Phase 2: Message Handling
- [ ] Implement `SendMessage()` method
  - [ ] Filter relevant messages
  - [ ] Build structured prompt
  - [ ] Execute mentat with stdin
  - [ ] Parse response
  - [ ] Handle errors
- [ ] Implement `StreamMessage()` method (if needed)
- [ ] Add logging

### Phase 3: Advanced Features
- [ ] Model selection support
- [ ] Temperature configuration
- [ ] File targeting options
- [ ] Git integration configuration

### Phase 4: Testing & Documentation
- [ ] Integration tests
- [ ] Add to `doctor` command
- [ ] Update README.md
- [ ] Create example config: `examples/mentat-coding.yaml`
- [ ] Add troubleshooting section
- [ ] Update CHANGELOG.md

### Phase 5: Quality Assurance
- [ ] Linting clean
- [ ] Tests passing
- [ ] Manual testing complete

## Example Use Cases

### Use Case 1: Multi-File Refactoring
```yaml
agents:
  - type: mentat
    name: "Refactorer"
    prompt: "Refactor code to improve structure and maintainability"
  
  - type: claude
    name: "Reviewer"
    prompt: "Review changes for correctness and best practices"

orchestrator:
  mode: round-robin
  initial_prompt: "Refactor the API layer to use dependency injection"
```

### Use Case 2: Feature Implementation
```yaml
agents:
  - type: mentat
    name: "Developer"
    prompt: "Implement features with clean, tested code"
  
  - type: qoder
    name: "Documenter"
    prompt: "Add documentation for new features"

orchestrator:
  mode: reactive
  initial_prompt: "Add user authentication with OAuth2"
```

## Success Criteria
- [ ] Mentat CLI detected by `agentpipe doctor`
- [ ] Can participate in multi-agent conversations
- [ ] Proper error handling
- [ ] Documentation complete
- [ ] All tests passing

## Priority
**HIGH** - Top 3 missing agent based on research

## Estimated Effort
**LOW-MEDIUM** - 1-2 days

## References
- Mentat GitHub: https://github.com/AbanteAI/mentat
- Mentat Docs: https://www.mentat.ai/

## Labels
- `enhancement`
- `agent-support`
- `high-priority`
