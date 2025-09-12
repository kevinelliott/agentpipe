# AgentPipe 🚀

[![CI](https://github.com/kevinelliott/agentpipe/actions/workflows/test.yml/badge.svg)](https://github.com/kevinelliott/agentpipe/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/kevinelliott/agentpipe)](https://github.com/kevinelliott/agentpipe/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/kevinelliott/agentpipe)](https://github.com/kevinelliott/agentpipe/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/kevinelliott/agentpipe)](https://goreportcard.com/report/github.com/kevinelliott/agentpipe)

AgentPipe is a CLI and TUI application that orchestrates conversations between multiple AI agents. It allows different AI CLI tools (like Claude, Gemini, Qwen) to communicate with each other in a shared "room", creating dynamic multi-agent conversations.

## Screenshots

### Enhanced TUI Interface
![AgentPipe TUI](screenshots/tui/tui1.png)
*Enhanced TUI with multi-panel layout: agent list with status indicators, conversation view with metrics, statistics panel showing turns and total cost, configuration panel, and user input area*

## Features

- **Multi-Agent Conversations**: Connect multiple AI agents in a single conversation
- **Multiple Conversation Modes**:
  - `round-robin`: Agents take turns in a fixed order
  - `reactive`: Agents respond based on conversation dynamics
  - `free-form`: Agents participate freely as they see fit
- **Flexible Configuration**: Use command-line flags or YAML configuration files
- **Enhanced TUI Interface**: 
  - Beautiful panelized layout with agent list, conversation view, and user input
  - Color-coded agent messages with custom badges
  - Real-time agent activity indicators (green/grey dots)
  - Real-time metrics display (duration, tokens, cost)
  - Modal system for agent details
  - User participation in conversations
  - Topic panel showing initial prompt
  - Statistics panel with turn/agent counters
  - Configuration panel showing active settings
- **Chat Logging**: Automatic conversation logging to `~/.agentpipe/chats/`
- **Response Metrics**: Track response time, token usage, and estimated costs
- **Health Checks**: Automatic agent health verification before conversations
- **Agent Detection**: Built-in doctor command to check installed AI CLIs
- **Customizable Agents**: Configure prompts, models, and behaviors for each agent

## What's New 🎉

### Latest Features (v0.0.7)
- **Enhanced TUI Interface**: 
  - Beautiful multi-panel layout with dedicated sections for agents, chat, stats, and config
  - Real-time agent activity indicators showing when agents are thinking/responding
  - Consolidated message headers (only shown when speaker changes)
  - Proper multi-paragraph message handling
  - Topic panel displaying the initial conversation prompt
  - Statistics showing current/max turns, connected/configured agents, and total cost
  - Configuration panel showing all active settings including config file path
- **Response Metrics**: Real-time tracking with inline display in chat (X.Xs, XXX tokens, $X.XXXX)
- **Improved Message Formatting**: Better handling of multi-line agent responses
- **Chat Logging**: Dual output support - logs to file while displaying in TUI
- **Configuration Honoring**: TUI mode now properly respects all config settings
- **User Participation**: Seamless integration allowing users to join agent conversations

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew tap kevinelliott/tap
brew install agentpipe
```

### Using the install script

```bash
curl -sSL https://raw.githubusercontent.com/kevinelliott/agentpipe/main/install.sh | bash
```

### Using Go

```bash
go install github.com/kevinelliott/agentpipe@latest
```

### Build from source

```bash
git clone https://github.com/kevinelliott/agentpipe.git
cd agentpipe
go build -o agentpipe .
```

## Prerequisites

AgentPipe requires at least one AI CLI tool to be installed:

- [Claude Code CLI](https://github.com/anthropics/claude-code) - `claude`
- [Gemini CLI](https://github.com/google/generative-ai-cli) - `gemini`
- [Qwen Code CLI](https://github.com/QwenLM/qwen-code) - `qwen`
- [Codex CLI](https://github.com/openai/codex-cli) - `codex` (OpenAI's agentic CLI)
- [Ollama](https://github.com/ollama/ollama) - `ollama`

Check which agents are available on your system:

```bash
agentpipe doctor
```

## Quick Start

### Simple conversation with command-line flags

```bash
# Start a conversation between Claude and Gemini
agentpipe run -a claude:Alice -a gemini:Bob -p "Let's discuss AI ethics"

# Use TUI mode for a better experience
agentpipe run -a claude:Poet -a gemini:Scientist --tui

# Configure conversation parameters
agentpipe run -a claude:Agent1 -a gemini:Agent2 \
  --mode reactive \
  --max-turns 10 \
  --timeout 45 \
  --prompt "What is consciousness?"
```

### Using configuration files

```bash
# Run with a configuration file
agentpipe run -c examples/simple-conversation.yaml

# Run a debate between three agents
agentpipe run -c examples/debate.yaml --tui

# Brainstorming session with multiple agents
agentpipe run -c examples/brainstorm.yaml
```

## Configuration

### YAML Configuration Format

```yaml
version: "1.0"

agents:
  - id: agent-1
    type: claude  # Agent type (claude, gemini, qwen, etc.)
    name: "Friendly Assistant"
    prompt: "You are a helpful and friendly assistant."
    announcement: "Hello everyone! I'm here to help!"
    model: claude-3-sonnet  # Optional: specific model
    temperature: 0.7        # Optional: response randomness
    max_tokens: 1000        # Optional: response length limit

  - id: agent-2
    type: gemini
    name: "Technical Expert"
    prompt: "You are a technical expert who loves explaining complex topics."
    announcement: "Technical Expert has joined the chat!"
    temperature: 0.5

orchestrator:
  mode: round-robin       # Conversation mode
  max_turns: 10          # Maximum conversation turns
  turn_timeout: 30s      # Timeout per agent response
  response_delay: 2s     # Delay between responses
  initial_prompt: "Let's start our discussion!"

logging:
  enabled: true          # Enable chat logging
  chat_log_dir: ~/.agentpipe/chats  # Custom log path (optional)
  show_metrics: true     # Display response metrics in TUI
  log_format: text      # Log format (text or json)
```

### Conversation Modes

- **round-robin**: Agents speak in a fixed rotation
- **reactive**: Agents respond based on who spoke last
- **free-form**: Agents decide when to participate

## Commands

### `agentpipe run`

Start a conversation between agents.

**Flags:**
- `-c, --config`: Path to YAML configuration file
- `-a, --agents`: List of agents (format: `type:name`)
- `-m, --mode`: Conversation mode (default: round-robin)
- `--max-turns`: Maximum conversation turns (default: 10)
- `--timeout`: Response timeout in seconds (default: 30)
- `--delay`: Delay between responses in seconds (default: 1)
- `-p, --prompt`: Initial conversation prompt
- `-t, --tui`: Use enhanced TUI interface with panels and user input
- `--log-path`: Custom path for chat logs (default: ~/.agentpipe/chats)
- `--no-log`: Disable chat logging
- `--show-metrics`: Display response metrics (duration, tokens, cost)
- `--skip-health-check`: Skip agent health checks (not recommended)

### `agentpipe doctor`

Check which AI CLI tools are installed and available.

```bash
agentpipe doctor
```

## Examples

### Poetry vs Science Debate

```yaml
# Save as poetry-science.yaml
version: "1.0"
agents:
  - id: poet
    type: claude
    name: "The Poet"
    prompt: "You speak in beautiful metaphors and see the world through an artistic lens."
    temperature: 0.9
    
  - id: scientist
    type: gemini
    name: "The Scientist"
    prompt: "You explain everything through logic, data, and scientific principles."
    temperature: 0.3

orchestrator:
  mode: round-robin
  initial_prompt: "Is love just chemistry or something more?"
```

Run with: `agentpipe run -c poetry-science.yaml --tui`

### Creative Brainstorming

```bash
agentpipe run \
  -a claude:IdeaGenerator \
  -a gemini:CriticalThinker \
  -a qwen:Implementer \
  -a codex:TechAdvisor \
  --mode free-form \
  --max-turns 15 \
  --show-metrics \
  -p "How can we make education more engaging?"
```

## TUI Interface

The enhanced TUI provides a rich, interactive experience for managing multi-agent conversations:

### Layout
The TUI is divided into multiple panels:
- **Agents Panel** (Left): Shows all connected agents with real-time status indicators
- **Chat Panel** (Center): Displays the conversation with color-coded messages
- **Topic Panel** (Top Right): Shows the initial conversation prompt
- **Statistics Panel** (Right): Displays turn count, agent statistics, and total conversation cost
- **Configuration Panel** (Right): Shows active settings and config file path
- **User Input Panel** (Bottom): Allows you to participate in the conversation

### Visual Features
- **Agent Status Indicators**: Green dot (🟢) for active/responding, grey dot (⚫) for idle
- **Color-Coded Messages**: Each agent gets a unique color for easy tracking
- **Consolidated Headers**: Message headers only appear when the speaker changes
- **Metrics Display**: Response time (seconds), token count, and cost shown inline when enabled
- **Multi-Paragraph Support**: Properly formatted multi-line agent responses

### Controls
- `Tab`: Switch between panels (Agents, Chat, User Input)
- `↑↓`: Navigate in active panel
- `Enter`: Send message when in User Input panel
- `i`: Show agent info modal (when in Agents panel)
- `Ctrl+C` or `q`: Quit
- `PageUp/PageDown`: Scroll conversation
- Active agent indicators: 🟢 (responding) / ⚫ (idle)

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/kevinelliott/agentpipe.git
cd agentpipe

# Build the binary
go build -o agentpipe .

# Or build with version information
VERSION=v0.0.7 make build

# Run tests
go test ./...
```

### Project Structure

```
agentpipe/
├── cmd/              # CLI commands
│   ├── root.go      # Root command
│   ├── run.go       # Run conversation command
│   └── doctor.go    # Doctor diagnostic command
├── pkg/
│   ├── agent/       # Agent interface and registry
│   ├── adapters/    # Agent implementations
│   │   ├── claude.go   # Claude adapter
│   │   ├── gemini.go   # Gemini adapter
│   │   ├── qwen.go     # Qwen adapter
│   │   ├── codex.go    # Codex (OpenAI) adapter
│   │   └── ollama.go   # Ollama adapter
│   ├── config/      # Configuration handling
│   ├── orchestrator/# Conversation orchestration
│   ├── logger/      # Chat logging and output
│   └── tui/         # Terminal UI
│       ├── basic.go    # Basic TUI
│       └── enhanced.go # Enhanced panelized TUI
├── examples/        # Example configurations
│   ├── simple-conversation.yaml
│   ├── brainstorm.yaml
│   └── codex-brainstorm.yaml
└── main.go
```

### Adding New Agent Types

1. Create a new adapter in `pkg/adapters/`
2. Implement the `Agent` interface
3. Register the factory in `init()`

```go
type MyAgent struct {
    agent.BaseAgent
}

func init() {
    agent.RegisterFactory("myagent", NewMyAgent)
}
```

## Troubleshooting

### Agent Health Check Failed
If you encounter health check failures:
1. Verify the CLI is properly installed: `which <agent-name>`
2. Check if the CLI requires authentication or API keys
3. Try running the CLI manually to ensure it works
4. Use `--skip-health-check` flag as a last resort (not recommended)

### Qwen Code CLI Issues
The Qwen Code CLI uses a different interface than other agents:
- Use `qwen --prompt "your prompt"` for non-interactive mode
- The CLI may open an interactive session if not properly configured
- Full documentation: https://github.com/QwenLM/qwen-code

### Gemini Model Not Found
If you get a 404 error with Gemini:
- Check your model name in the configuration
- Ensure you have access to the specified model
- Try without specifying a model to use the default

### Chat Logs Location
Chat logs are saved by default to:
- macOS/Linux: `~/.agentpipe/chats/`
- Windows: `%USERPROFILE%\.agentpipe\chats\`

You can override this with `--log-path` or disable logging with `--no-log`.

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.