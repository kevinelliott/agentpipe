# Add Support for GPT Engineer CLI Agent

## Overview
Add support for [GPT Engineer](https://github.com/gpt-engineer-org/gpt-engineer), a tool that generates entire codebases from specifications.

## Why Add GPT Engineer?
- ⭐ **52,000+ GitHub stars** - One of the most popular AI coding tools
- 🏗️ **Whole-project generation** - Unique use case (not code editing)
- 💬 **CLI-first design** - Built for terminal use
- 🔄 **Iterative development** - Refine generated code through conversation
- 🎯 **Different workflow** - Complements existing editing-focused agents
- 🌐 **Multi-LLM support** - Works with various providers

## Differentiation from Other Agents
- **Aider/Mentat:** Edit existing code
- **Claude/Cursor:** General coding assistance
- **GPT Engineer:** Generate new projects from scratch
- Use case: Bootstrapping new projects, prototypes, MVPs

## Technical Details

### Installation
```bash
pip install gpt-engineer
```

### CLI Interface
```bash
# Generate project from prompt
gpte "Create a REST API for a todo list with FastAPI"

# Or using full command
gpt-engineer "Build a React dashboard with charts"

# Specify model
gpte --model gpt-4 "Create a CLI tool in Python"

# Iterative mode
gpte --improve
```

### Integration Approach
- **Adapter Location:** `pkg/adapters/gpt_engineer.go`
- **Command Detection:** `which gpte` or `which gpt-engineer`
- **Execution Mode:** CLI with prompt argument
- **Model Support:** Via `--model` flag (optional)
- **Pattern:** Similar to Codex adapter (exec with arguments)

### Example Agent Configuration
```yaml
agents:
  - id: gpte-architect
    type: gpt-engineer
    name: "Project Generator"
    prompt: "You generate complete, well-structured projects"
    model: gpt-4  # optional
    temperature: 0.7
```

## Implementation Checklist

### Phase 1: Basic Integration
- [ ] Create `pkg/adapters/gpt_engineer.go`
  - [ ] Implement `Initialize()` method
  - [ ] Implement `IsAvailable()` method (check for `gpte` or `gpt-engineer`)
  - [ ] Implement `HealthCheck()` method
  - [ ] Implement `GetCLIVersion()` method
- [ ] Add to agent registry
- [ ] Create unit tests

### Phase 2: Message Handling
- [ ] Implement `SendMessage()` method
  - [ ] Filter relevant messages
  - [ ] Build structured prompt
  - [ ] Execute gpte with prompt
  - [ ] Parse response (may be file/directory output)
  - [ ] Handle errors
- [ ] Consider output handling (files vs. text)
- [ ] Add logging

### Phase 3: Advanced Features
- [ ] Model selection support
- [ ] Temperature configuration
- [ ] Iterative mode support (`--improve`)
- [ ] Output directory configuration
- [ ] Handle multi-file output

### Phase 4: Testing & Documentation
- [ ] Integration tests
- [ ] Add to `doctor` command
- [ ] Update README.md
- [ ] Create example config: `examples/gpt-engineer-project.yaml`
- [ ] Add troubleshooting section
- [ ] Update CHANGELOG.md

### Phase 5: Quality Assurance
- [ ] Linting clean
- [ ] Tests passing
- [ ] Manual testing complete

## Example Use Cases

### Use Case 1: Project Bootstrapping
```yaml
agents:
  - type: gpt-engineer
    name: "Architect"
    prompt: "Generate well-structured project foundations"
  
  - type: claude
    name: "Reviewer"
    prompt: "Review generated code for best practices"
  
  - type: qoder
    name: "Refiner"
    prompt: "Refine and optimize the generated code"

orchestrator:
  mode: round-robin
  initial_prompt: "Create a microservices architecture for an e-commerce platform"
```

### Use Case 2: MVP Generation
```yaml
agents:
  - type: gpt-engineer
    name: "Builder"
    prompt: "Build functional MVPs quickly"
  
  - type: cursor
    name: "Enhancer"
    prompt: "Add polish and features to the MVP"

orchestrator:
  mode: reactive
  initial_prompt: "Build a SaaS landing page with auth and payment"
```

### Use Case 3: Prototype Discussion
```yaml
# Interesting: GPT Engineer generates, others critique
agents:
  - type: gpt-engineer
    name: "Prototyper"
    prompt: "Generate quick prototypes for discussion"
  
  - type: gemini
    name: "Critic 1"
    prompt: "Critique architecture and design choices"
  
  - type: claude
    name: "Critic 2"
    prompt: "Suggest improvements and alternatives"

orchestrator:
  mode: round-robin
  max_turns: 6
  initial_prompt: "Design a real-time collaborative editor"
```

## Implementation Considerations

### Output Handling
GPT Engineer generates **files/directories**, not just text:
- May need to capture file output descriptions
- Consider how to represent generated files in conversation
- Possibly return file tree summary instead of full content

### Working Directory
- GPT Engineer works in current directory
- May need to:
  - Create temporary directory for output
  - Or configure output directory
  - Return summary of generated files

### Iterative Mode
- GPT Engineer supports `--improve` for refinement
- Could enable multi-turn improvements in conversation
- Track context between iterations

## Success Criteria
- [ ] GPT Engineer CLI detected by `agentpipe doctor`
- [ ] Can participate in conversations (even with file output)
- [ ] Handles project generation gracefully
- [ ] Proper error handling
- [ ] Documentation complete
- [ ] All tests passing

## Priority
**MEDIUM-HIGH** - Top 3 missing agent, but different workflow

## Estimated Effort
**MEDIUM** - 2-3 days
- Need to handle file/directory output
- Different workflow than other agents
- May need special handling for conversation context

## References
- GPT Engineer GitHub: https://github.com/gpt-engineer-org/gpt-engineer
- GPT Engineer Docs: https://gpt-engineer.readthedocs.io/

## Labels
- `enhancement`
- `agent-support`
- `high-priority`
- `unique-workflow`
