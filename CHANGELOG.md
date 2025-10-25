# Changelog

All notable changes to AgentPipe will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.4.7] - 2025-10-25

### Changed
- **Kimi CLI Version Detection**: Updated to use GitHub releases API
  - Changed package manager from "uv" to "github" in registry
  - Package name now points to "MoonshotAI/kimi-cli" repository
  - Uses standard GitHub releases endpoint for version checking
  - More reliable than previous custom parsing approach
  - Consistent with other GitHub-based agents (Qwen, etc.)

### Technical Details
- Reuses existing `getGitHubLatestRelease()` infrastructure
- Fetches from `https://api.github.com/repos/MoonshotAI/kimi-cli/releases/latest`
- Properly detects and compares versions for update notifications
- Tested with `agentpipe agents list --outdated` command

### Benefits
- Improved reliability for version detection
- Simpler implementation using existing code
- Better alignment with standard practices
- More accurate update notifications for users

## [v0.4.6] - 2025-10-25

### Added
- **Groq Code CLI Agent Support**
  - New adapter for Groq Code CLI (`groq` command)
  - Powered by Groq's Lightning Processing Units (LPUs) for ultra-fast inference
  - Installation via npm: `npm install -g groq-code-cli@latest`
  - Supports temperature configuration via agent config
  - Interactive CLI with stdin-based message passing
  - Complete integration with standardized three-part prompt system
  - Comprehensive logging and error handling
  - Registry entry with install/uninstall/upgrade commands for all platforms
  - Authentication via GROQ_API_KEY environment variable or `/login` command

### Technical Details
- Implements all required Agent interface methods
- Follows established patterns from Claude and Gemini adapters
- Message filtering to prevent echo in multi-agent conversations
- Output cleaning to remove system messages and authentication prompts
- Health check via `--version` flag
- Version detection via registry system
- Supports both SendMessage and StreamMessage modes

## [v0.4.5] - 2025-10-25

### Changed
- **Kimi Installation**: Enhanced install and upgrade commands to explicitly specify Python 3.13
  - Install: `uv tool install --python 3.13 kimi-cli`
  - Upgrade: `uv tool upgrade kimi-cli --python 3.13 --no-cache`
  - Ensures correct Python version is used for Kimi CLI deployment
  - Updated README with explicit Python version specification
  - Prevents version conflicts with other Python installations

## [v0.4.4] - 2025-10-25

### Added
- **Dedicated Security Workflows**: Trivy and CodeQL now have their own workflows
  - Separated security scanning from test workflow
  - Improved workflow organization and clarity

### Changed
- **README Badges**: Enhanced badge section with additional metrics
  - Added downloads badge showing total release downloads
  - Added GitHub stars badge with icon
  - Improved visual hierarchy with consistent color coding
  - All badges properly linked for easy navigation

### Fixed
- **Windows Test Failure**: Fixed TestIsInstallable for platform-specific installations
  - Removed Ollama from test expectations (Windows-only instructions)
  - Test now only checks agents with actual install commands across all platforms
  - Resolves GitHub Actions test failures on Windows runners

### Improved
- **CI/CD Organization**: Cleaner separation of concerns
  - Test workflow focuses on testing and linting
  - Security workflows handle vulnerability scanning
  - Reduced workflow complexity and interdependencies

## [v0.4.3] - 2025-10-25

### Added
- **Kimi CLI Agent Support** (Moonshot AI)
  - New adapter for Kimi CLI (`kimi` command)
  - Installation via `uv tool install kimi-cli` (requires Python 3.13+)
  - Support for uv package manager-based installation
  - Upgrade via `uv tool upgrade kimi-cli --no-cache`
  - Blue-gradient ASCII logo in branding for visual distinction
  - Health checks and agent verification
  - Interactive-aware error handling with helpful authentication guidance
  - Structured prompt building for multi-agent conversations
  - Message filtering and context management
  - Stream message support with best-effort implementation

### Changed
- Updated registry test to expect 12 supported agents (added Kimi)
- Updated README with Kimi agent documentation and installation instructions

## [v0.4.2] - 2025-10-24

### Fixed
- **Qoder Installation/Upgrade**: Added `--force` flag to Qoder install and upgrade commands
  - Allows Qoder to be installed/upgraded even if version already exists
  - Prevents "Version X.X.X already exists" errors

## [v0.4.1] - 2025-10-22

### Added
- **Conversation Summarization**: Automatic AI-generated summaries at conversation completion
  - Configurable summary agent (default: Gemini, supports all agent types)
  - `--no-summary` flag to disable summaries
  - `--summary-agent` flag to override configured agent
  - Summary configuration in YAML config file
  - Summary metadata includes agent type, model, tokens, cost, and duration
  - Summary tokens and cost factored into conversation totals
  - Smart prompt design avoiding meta-commentary

- **Unique Agent IDs**: Enhanced agent identification for multiple agents of same type
  - Agent IDs now unique per instance: `{agentType}-{index}` (e.g., `claude-0`, `claude-1`)
  - AgentID included in all bridge streaming events
  - Allows tracking of multiple agents with same type in single conversation
  - AgentID in conversation.started participants list
  - AgentID in all message.created events

- **Event Store**: Local event persistence
  - Events saved to `~/.agentpipe/events/` directory
  - One JSON Lines file per conversation
  - Non-blocking async operation
  - Debug logging for storage errors

### Changed
- **Bridge Events**: Enhanced ConversationCompletedData structure
  - Summary field now contains full SummaryMetadata (instead of plain string)
  - Includes summary agent type, model, tokens, cost, duration
  - Total tokens and cost now include summary metrics
  - Duration does not include summary generation time

- **Streaming Protocol**: Updated message event structure
  - EmitMessageCreated now requires agentID as first parameter
  - MessageCreatedData includes agent_id field
  - AgentParticipant includes agent_id field

### Fixed
- Agent identification for conversations with multiple agents of same type
- Cost tracking to include summary generation costs in totals
- Thread-safe access to bridge emitter in orchestrator

## [v0.4.0] - 2025-10-21

### Added
- **Bridge Connection Events**: Automatic connection announcement
  - Emit bridge.connected event on emitter initialization
  - System info included in connection event
  - Synchronous sending ensures reliability

- **Cancellation Detection**: Detect and report conversation interruption
  - Emit conversation.completed with status="interrupted" on Ctrl+C
  - Distinguish between normal completion and cancellation
  - Proper error propagation in orchestrator

### Changed
- **Event Reliability**: Improved critical event delivery
  - Use synchronous SendEvent for completion and error events
  - Prevent truncated JSON payloads on program exit

## [v0.3.0] - 2025-10-21

### Added
- **Streaming Bridge**: Opt-in real-time conversation streaming to AgentPipe Web
  - Stream live conversation events to AgentPipe Web for browser viewing
  - Four event types: `conversation.started`, `message.created`, `conversation.completed`, `conversation.error`
  - Non-blocking async HTTP implementation that never blocks conversations
  - CLI commands for easy configuration:
    - `agentpipe bridge setup` - Interactive configuration wizard
    - `agentpipe bridge status` - View current bridge configuration (with `--json` flag support)
    - `agentpipe bridge test` - Test connection to AgentPipe Web
    - `agentpipe bridge disable` - Disable streaming
  - System info collection: OS, version, architecture, AgentPipe version, Go version
  - Configuration via viper config file or environment variables
  - Build-tag conditional defaults (dev vs production URLs)
  - Privacy-first design: disabled by default, API keys never logged, clear disclosure
  - Production-ready with retry logic (exponential backoff) and comprehensive tests (>80% coverage)
  - Agent participants tracked with CLI version information
  - Conversation metrics: turns, tokens, cost, duration

### Changed
- **BREAKING**: Extended Agent interface with new `GetCLIVersion()` method
  - All agent adapters now implement version detection
  - Uses internal registry for version lookup
  - Required for streaming bridge agent participant data
  - Custom agent implementations must add this method

### Improved
- **Configuration**: Added `BridgeConfig` struct to support streaming bridge settings
  - Bridge enabled status, URL, API key, timeout, retry attempts, log level
  - Defaults applied automatically in config parsing
  - Environment variable overrides supported

### Fixed
- **Thread Safety**: Added RWMutex for safe concurrent access to orchestrator bridge emitter
- **Linting**: Fixed non-constant format string error in orchestrator error handling

## [v0.2.2] - 2025-10-20

### Added
- **JSON Output Support**: Added `--json` flag to `agentpipe agents list` command
  - Regular list mode outputs structured JSON with agent details
  - Outdated mode outputs version comparison data in JSON format
  - Works with all existing filters: `--installed`, `--outdated`, `--current`
  - Clean JSON structure with appropriate omitempty fields
  - Example: `agentpipe agents list --json`
  - Example: `agentpipe agents list --outdated --json`
  - Useful for programmatic integration and automation

### Improved
- **Agent List Output**: Enhanced parallel version checking for both human-readable and JSON outputs
- **Code Organization**: Refactored version row type for better reusability

## [v0.2.1] - 2025-10-20

### Added
- **OpenCode CLI Agent**: Complete integration for SST's OpenCode terminal-native AI coding agent
  - Full adapter implementation with non-interactive `opencode run` mode
  - npm package support: `opencode-ai@latest`
  - Quiet flag for non-interactive execution
  - Comprehensive documentation and troubleshooting
  - Now 11 supported AI agent CLIs
- **Referral Links Section**: New dedicated section in README to support project development
  - Qoder referral link for users to support ongoing development
  - Clear explanation of how referral links help fund the project

### Fixed
- **Amp CLI**: Updated to support npm-based installation and automated upgrades
  - Changed from manual-only to `npm install -g @sourcegraph/amp`
  - Now supports `agentpipe agents upgrade amp`
- **Codex CLI**: Fixed npm package name for correct version detection
  - Corrected from `@openai/codex-cli` to `@openai/codex`
  - Added homebrew installation option: `brew install --cask codex`
  - Automated upgrades now work correctly

### Improved
- **Documentation**: Enhanced installation instructions for multiple agents
  - Added npm and homebrew options where applicable
  - Updated Prerequisites section with all installation methods
  - Added OpenCode to adapter reference implementations
- **Tests**: Updated registry tests to reflect Amp's new installable status

## [v0.2.0] - 2025-10-20

### Added
- **Agent Upgrade Command**: New `agentpipe agents upgrade` subcommand for easy updates
  - Upgrade individual agents: `agentpipe agents upgrade claude`
  - Upgrade multiple agents: `agentpipe agents upgrade claude ollama gemini`
  - Upgrade all installed agents: `agentpipe agents upgrade --all`
  - Automatic detection of installed agents for selective upgrades
  - User confirmation prompts before performing upgrades
  - Cross-platform support (darwin, linux, windows)
- **Automated Version Detection**: Complete version checking for all 10 supported agents
  - npm registry integration (Claude, Codex, Gemini, Copilot, Amp, Qwen)
  - Homebrew Formulae API integration (Ollama)
  - GitHub Releases API integration (Qwen fallback)
  - Shell script parsing for version extraction (Factory, Cursor)
  - JSON manifest parsing (Qoder)
  - Replaces all "manual install" placeholders with actual version numbers
- **Multiple Package Manager Support**: Extensible version checking architecture
  - `npm`: Query npm registry API for latest versions
  - `homebrew`: Query Homebrew Formulae API for latest versions
  - `github`: Query GitHub Releases API for latest releases
  - `script`: Parse shell install scripts for VER= or DOWNLOAD_URL= version patterns
  - `manifest`: Fetch and parse JSON manifests with "latest" field
- **Parallel Version Checking**: Dramatically improved performance with concurrent API calls
  - Goroutine-based concurrent version fetching
  - Buffered channels for result collection
  - Performance improvement: ~10+ seconds → ~3.7 seconds for 10 agents
  - Thread-safe result aggregation

### Fixed
- **npm 404 Errors**: Corrected package names for npm-based agents
  - Claude: `@anthropic-ai/claude-cli` → `@anthropic-ai/claude-code`
  - Codex: `@openai/codex-cli` → `@openai/codex`
  - Gemini: `@google/generative-ai-cli` → `@google/gemini-cli`
- **Ollama Version Detection**: Enhanced to work without running Ollama instance
  - Now parses version from warning messages (e.g., "Warning: client version is 0.12.5")
  - Improved `containsVersion()` and `extractVersionNumber()` logic
  - No longer requires Ollama server to be running

### Improved
- **UI/UX Enhancements**: Better table display for agent version information
  - Removed redundant "Status" column from outdated agents table
  - Rebalanced column widths for better readability
    - Agent: 15 → 12 characters
    - Installed Version: 20 → 24 characters
    - Latest Version: 20 → 24 characters
    - Total width: 80 → 85 characters
  - Changed upgrade instructions from "install" to "upgrade" for clarity
- **Agent Registry Metadata**: Complete package manager information for all agents
  - Factory: Uses script-based version detection from https://app.factory.ai/cli
  - Amp: Uses npm registry @sourcegraph/amp
  - Cursor: Uses script-based version detection from https://cursor.com/install
  - Qoder: Uses manifest from qoder-ide.oss-ap-southeast-1.aliyuncs.com
  - All agents now have upgrade commands defined for current OS

## [v0.1.5] - 2025-10-19

### Fixed
- **Linting Errors**: Fixed golangci-lint errors in doctor.go
  - Fixed gofmt formatting (struct field alignment)
  - Fixed prealloc warning (pre-allocated slices with known capacity)
  - CI/CD pipeline now passes all quality checks

## [v0.1.4] - 2025-10-19

### Added
- **Doctor Command JSON Output**: Programmatic agent detection for web interfaces
  - `--json` flag for structured output in JSON format
  - Complete system diagnostics in machine-readable format
  - Agent detection with availability, authentication, and version info
  - Perfect for dynamic UI generation (e.g., agentpipe-web)
  - Outputs: `system_environment`, `supported_agents`, `available_agents`, `configuration`, `summary`
  - Each agent includes: name, command, path, version, install/upgrade commands, docs, auth status
  - Clean JSON output (logo suppressed when using `--json` flag)

### Improved
- **Documentation**: Added comprehensive JSON output format documentation to README
  - Usage examples for both human-readable and JSON modes
  - Field-by-field JSON structure explanation
  - Use cases for programmatic consumption

## [v0.1.3] - 2025-10-19

### Added
- **Factory CLI Agent Support**: Full integration with Factory.ai's Droid coding agent
  - Non-interactive exec mode with `droid exec` command
  - Autonomy level configuration (`--auto high`) for multi-agent conversations
  - Structured prompt delivery with clear context sections
  - Smart message filtering (excludes agent's own messages)
  - Comprehensive logging and error handling
  - Optional model specification via config
  - Agent-native software development with Code Droid and Knowledge Droid
  - Installation: `curl -fsSL https://app.factory.ai/cli | sh`
  - Documentation: https://docs.factory.ai/cli

### Improved
- **Doctor Command**: Added Factory CLI detection with installation and upgrade instructions
- **README**: Updated with Factory CLI support and troubleshooting section
- **Architecture Documentation**: Added Factory to supported agents list and visual diagrams
- **Agent Count**: Now supporting 10 AI agent CLIs (up from 9)

## [v0.1.1] - 2025-10-19

### Fixed
- **Windows Test Compatibility**: Fixed timer resolution issues causing test failures on Windows
  - Windows timer granularity (~15.6ms) caused `time.Since()` to return 0 for very fast operations
  - Increased mock agent delay to 20ms in TestConversationWithMetrics to ensure measurable durations
  - Test now passes reliably on all platforms (Windows, macOS, Linux)
- **Windows File Permission Tests**: Added runtime OS detection to skip Unix-specific permission checks
  - TestState_Save now correctly skips file permission verification on Windows
  - Tests properly handle platform differences in file permission models

### Changed
- **Go Version Requirement**: Downgraded from Go 1.25.3 to Go 1.24.0 for broader compatibility
  - Maintains compatibility with golangci-lint v1.64.8
  - All GitHub Actions workflows updated to use Go 1.24
  - go.mod updated to reflect Go 1.24 requirement

### Documentation
- Added comprehensive documentation of Windows-specific testing challenges in CLAUDE.md
- Documented timer resolution requirements for cross-platform test development
- Updated development guide with platform compatibility considerations

## [v0.1.0] - 2025-10-16

### Added
- **Agent Type Indicators**: Message badges now show agent type in parentheses (e.g., "Alice (qoder)")
  - Helps users quickly identify which agent type is responding
  - Displayed in all message badges in both TUI and CLI output
  - Agent type automatically populated from agent configuration
- **Branded TUI Logo**: Enhanced TUI with colored ASCII sunset gradient logo
  - Consistent branding across CLI and TUI modes
  - Shared branding package for code reuse
  - Beautiful sunset gradient colors using ANSI 24-bit color codes
- **TUI Search Feature**: Press Ctrl+F to search through conversation messages
  - Case-insensitive search through message content and agent names
  - n/N navigation between search results
  - Visual feedback with match count and position
  - Auto-scroll to current search result
- **Agent Filtering**: Use slash commands to filter messages by agent
  - `/filter <agent>` to show only specific agent's messages
  - `/clear` to remove active filter
  - System messages always visible regardless of filter
- **Help Modal**: Press `?` to view all keyboard shortcuts
  - Organized by category (General, Conversation, Search, Commands)
  - Toggle on/off with `?` or Esc
  - Comprehensive documentation of all keybindings
- **Amp CLI Agent Support**: Advanced integration with Sourcegraph's Amp coding agent
  - Thread management for efficient conversations
  - Smart message filtering (excludes agent's own messages)
  - Structured prompt delivery with clear context sections
  - Streaming support with thread continuation
  - Reduces API costs by 50-90% vs traditional approaches
- **Qoder CLI Agent Support**: Full integration with Qoder agentic coding platform
  - Non-interactive print mode with `qodercli --print`
  - Enhanced context engineering for comprehensive codebase understanding
  - Intelligent agents for systematic software development tasks
  - Built-in tools (Grep, Read, Write, Bash) for file operations
  - MCP integration support for extended functionality
  - Permission bypass with `--yolo` flag for automated execution

### Improved
- **Enhanced HOST vs SYSTEM Distinction**: Clearer visual separation in message display
  - HOST messages now formatted like agent messages with badge, newline, and indented content
  - SYSTEM messages remain inline format for announcements
  - HOST badge uses distinctive purple color (#99)
  - Makes conversation context clearer by distinguishing orchestrator prompts from system notifications
- **Gemini Adapter Reliability**: Improved error handling for process exit issues
  - Now accepts valid output even when Gemini CLI doesn't exit cleanly
  - Distinguishes between real API errors (404, 401) and harmless process termination
  - Enhanced output cleaning to filter error traces and stack dumps
  - Significantly reduces false failures in multi-agent conversations
- **Standardized Agent Introduction**: All agents now receive complete conversation history when first coming online
  - **Complete message delivery**: Agents receive ALL existing messages (system prompts + agent messages)
  - **No message loss**: Correctly identifies orchestrator's initial prompt vs. agent announcements
  - **Correct topic extraction**: Finds system message with AgentID="system" as the conversation topic
  - **Clear labeling**: System messages are explicitly labeled as "SYSTEM:" in conversation history
  - **Structured three-part prompt** format:
    1. **AGENT SETUP** (first): Agent's name, role, and custom instructions
    2. **CONVERSATION TOPIC** (second): Initial orchestrator prompt highlighted prominently
    3. **CONVERSATION SO FAR** (third): All existing messages (announcements + responses)
- **Amp Agent Context Awareness**: Restructured prompt delivery with thread management
  - Uses `amp thread new` and `amp thread continue` for efficient communication
  - **Smart message filtering**: Automatically excludes Amp's own messages from being sent back to it
  - Only sends messages from OTHER agents and system messages (Amp already knows what it said)
  - **Thread management**: Reduces API costs and response times by 50-90%
  - Automatic thread ID tracking and incremental message sending
  - Enhanced logging with prompt previews (first 300 chars) for debugging
- **Session Summary**: Now displayed for all conversation endings, not just CTRL-C interruptions
  - Shows summary when conversation completes normally (max turns reached)
  - Shows summary when interrupted with CTRL-C
  - Shows summary even when conversation ends with an error
  - Includes total messages, tokens, time spent, and cost for all endings

### Fixed
- **Inconsistent Agent Badge Colors**: Fixed race condition causing first message to have grey badge
  - Now ensures agent color is assigned before badge style is retrieved
  - Agent name badges now consistently show the assigned color from first message onward
  - Improved visual consistency in both TUI and CLI output
- **TUI Display Corruption**: Fixed stderr output interfering with TUI rendering
  - Removed all `fmt.Fprintf(os.Stderr, ...)` calls from TUI code
  - Metrics and logging no longer corrupt the TUI alt-screen display
  - Silent error handling in TUI mode while maintaining conversation panel visibility
- **Agent Prompt Response**: Fixed critical bug where agents weren't properly responding to orchestrator's initial prompt
  - Changed prompt header from passive "CONVERSATION TOPIC" to directive "YOUR TASK - PLEASE RESPOND TO THIS"
  - Makes it clear the initial prompt is a direct instruction, not passive context
  - Agents now immediately engage with the topic instead of asking "what would you like help with?"
- **Amp Thread Creation Pattern**: Fixed empty response issue with Amp agent
  - Previously: Created thread with prompt, received empty response
  - Now: Create empty thread first, then send prompt via `thread continue`
  - Matches Amp CLI's expected pattern where `thread new` only returns thread ID
  - Amp now correctly responds to initial prompts on first turn
- **Agent Introduction Logic**: Fixed orchestrator prompt detection
  - Correctly distinguishes between orchestrator messages (AgentID="system") and agent announcements
  - Agent announcements are system messages from specific agents, not the conversation topic
  - All agents now receive the orchestrator's initial prompt in the "YOUR TASK" section
- **Codex Non-Interactive Mode**: Fixed terminal compatibility errors with Codex agent
  - Uses `codex exec` subcommand for non-interactive execution
  - Parses JSON output to extract agent messages cleanly
  - Automatically bypasses approval prompts with safety flags
  - No more "stdout is not a terminal" errors in multi-agent conversations
- **Standardized All Adapters**: Applied consistent interaction pattern across all 8 adapters
  - All adapters (Amp, Claude, Codex, Copilot, Cursor, Gemini, Qoder, Qwen) now use identical:
    - Three-part structured prompts (Setup → Task → History)
    - Message filtering to exclude agent's own messages
    - Comprehensive structured logging with timing and metrics
    - Proper error handling with specific error detection
  - Ensures reliable, consistent behavior across all agent types
- **Orchestrator Identification**: Changed orchestrator messages from "System" to "HOST" for clarity
  - Initial conversation prompt now uses `AgentID="host"` and `AgentName="HOST"`
  - Distinguishes orchestrator messages from system announcements (agent joins, etc.)
  - All 8 adapters updated to recognize both "system"/"System" and "host"/"HOST" for backwards compatibility
  - Makes it clear who is presenting the initial task vs. system notifications

## [v0.0.16] - 2025-10-15

### Added - Production-Ready Features
- **Prometheus Metrics**: Comprehensive observability with 10+ metric types
  - HTTP server with `/metrics`, `/health`, and web UI endpoints
  - Ready for Grafana dashboards and Prometheus alerting
  - Track requests, durations, tokens, costs, errors, rate limits, retries
  - OpenMetrics format support
- **Middleware Pipeline**: Extensible message processing architecture
  - 10+ built-in middleware (logging, metrics, validation, filtering, sanitization)
  - Custom middleware support for transforms and filters
  - Error recovery and panic handling
  - Chain-of-responsibility pattern implementation
- **Conversation State Management**: Save/resume conversations
  - Save/resume conversations from JSON state files
  - `agentpipe resume` command with --list flag
  - Automatic state directory (~/.agentpipe/states/)
  - Full conversation history, config, and metadata preservation
- **Export Functionality**: Multi-format conversation export
  - Export to JSON, Markdown, or HTML formats
  - Professional HTML styling with responsive design
  - XSS prevention with HTML escaping
  - `agentpipe export` command

### Added - Reliability & Performance
- **Rate Limiting**: Token bucket algorithm per agent
  - Configurable rate and burst capacity
  - Thread-safe implementation with ~60ns overhead
  - Automatic rate limit hit tracking in metrics
- **Retry Logic**: Exponential backoff with smart defaults
  - 3 retries with 1s initial delay, 30s max, 2.0x multiplier
  - Configurable per orchestrator
  - Retry attempt tracking in metrics
- **Structured Error Handling**: Typed error system
  - AgentError, ConfigError, ValidationError, TimeoutError, etc.
  - Error wrapping with context
  - Better error classification for metrics
- **Config Hot-Reload**: Development workflow enhancement
  - Watch config files for changes with viper.WatchConfig
  - Thread-safe reload with callbacks
  - --watch-config flag for development mode

### Added - Docker Support
- Multi-stage Dockerfile (~50MB final image)
- docker-compose.yml with metrics server on port 9090
- Health checks and graceful shutdown
- Volume mounts for configs and logs
- Non-root user for security
- Complete documentation in docs/docker.md

### Added - Testing & Quality
- **Comprehensive Test Coverage**: 200+ tests
  - 86+ unit tests across orchestrator, adapters, logger, errors, ratelimit, config, conversation
  - 15 integration tests for end-to-end conversation flows
  - 25+ benchmark tests for performance regression detection
  - TUI component tests with race detection
  - All tests passing with concurrent access validation
- **Documentation**: Complete docs/ directory
  - architecture.md - System design and patterns
  - contributing.md - Contribution guidelines
  - development.md - Development setup and workflows
  - troubleshooting.md - Common issues and solutions
  - docker.md - Docker deployment guide

### Added - Developer Experience
- **Interactive Init Command**: Configuration wizard
  - Guided prompts for all configuration options
  - Agent selection and configuration
  - Orchestrator mode and settings
  - Automatic file creation
- **Structured Logging**: Zerolog-based logging
  - JSON and pretty console output
  - Contextual fields for debugging
  - Integration across orchestrator, adapters, and commands
  - Maintains fmt.Fprintf for TUI display
- **Enhanced CLI**: New commands and flags
  - `agentpipe export` - Export conversations
  - `agentpipe resume` - Resume saved conversations
  - `agentpipe init` - Interactive config wizard
  - --save-state, --state-file, --watch-config flags

### Improved
- Godoc comments on all exported types and functions
- 0 linting issues with golangci-lint
- Thread-safe implementations throughout
- Proper resource cleanup and leak prevention

## [v0.0.15] - 2025-10-14

### Added
- **GitHub Copilot CLI Integration**: Full support for GitHub's Copilot terminal agent
  - Non-interactive mode support using `--prompt` flag
  - Automatic tool permission handling with `--allow-all-tools`
  - Multi-model support (Claude Sonnet 4.5, GPT-5, etc.)
  - Authentication detection and helpful error messages
  - Subscription requirement validation

### Improved
- **Graceful CTRL-C Handling**: Interrupting a conversation now displays a session summary
  - Total messages (agent + system)
  - Total tokens used
  - Total time spent (intelligently formatted: ms/s/m:s)
  - Total conversation cost
  - All messages are properly logged before exit
- **Total Time Tracking in TUI**: Statistics panel now shows cumulative time for all agent requests

### Fixed
- **Resource Leak Fixes**:
  - Fixed timer leak in cursor adapter (using `time.NewTimer` with proper cleanup)
  - Message channel now properly closed on TUI exit
  - Dropped messages now logged to stderr with counts
  - Orchestrator goroutine lifecycle properly tracked with graceful shutdown

## [v0.0.9] - 2025-10-12

### Added
- **Cursor CLI Integration**: Full support for Cursor's AI agent
  - Automatic authentication detection
  - Intelligent retry logic for improved reliability
  - Optimized timeout handling for cursor-agent's longer response times
  - JSON stream parsing for real-time response streaming
  - Robust error recovery and process management

## [v0.0.8] - 2025-10-10

### Added - TUI Improvements
- **Real-time Activity Indicators**: Visual feedback showing which agent is currently responding
- **Enhanced Metrics Display**:
  - Response time shown in seconds with 1 decimal precision (e.g., 2.5s)
  - Token count for each response
  - Cost estimate per response (e.g., $0.0012)
  - Total conversation cost tracking in Statistics panel
- **Improved Message Formatting**:
  - Consolidated headers (timestamp and name only shown when speaker changes)
  - Proper multi-paragraph message handling
  - Clean spacing between messages
  - No extra newlines between paragraphs from same speaker
- **Configuration Improvements**:
  - TUI now properly honors all configuration settings
  - Config file path displayed in Configuration panel
  - Dual output support (logs to file while displaying in TUI)
  - Metrics display controlled by `show_metrics` config option

### Improved
- **Better Error Handling**: Clearer error messages for agent failures and timeouts
- **Improved Health Checks**: More robust agent verification before starting conversations
- **Cost Tracking**: Automatic calculation and accumulation of API costs
- **Metrics Pipeline**: End-to-end metrics flow from orchestrator to TUI display

### Performance & Reliability
- **Optimized Message Handling**: Reduced memory usage and improved message rendering performance
- **Better Concurrency**: Proper goroutine management and channel handling
- **Graceful Shutdowns**: Clean termination of agents and proper resource cleanup

### User Experience
- **Intuitive Panel Navigation**: Tab-based navigation between panels
- **Real-time Feedback**: Instant visual indicators for agent activity
- **Clean Message Display**: Smart consolidation of headers and proper paragraph formatting
- **Cost Transparency**: See exactly how much each conversation costs

[Unreleased]: https://github.com/kevinelliott/agentpipe/compare/v0.2.1...HEAD
[v0.2.1]: https://github.com/kevinelliott/agentpipe/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/kevinelliott/agentpipe/compare/v0.1.5...v0.2.0
[v0.1.5]: https://github.com/kevinelliott/agentpipe/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/kevinelliott/agentpipe/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/kevinelliott/agentpipe/compare/v0.1.1...v0.1.3
[v0.1.1]: https://github.com/kevinelliott/agentpipe/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/kevinelliott/agentpipe/compare/v0.0.16...v0.1.0
[v0.0.16]: https://github.com/kevinelliott/agentpipe/compare/v0.0.15...v0.0.16
[v0.0.15]: https://github.com/kevinelliott/agentpipe/compare/v0.0.9...v0.0.15
[v0.0.9]: https://github.com/kevinelliott/agentpipe/compare/v0.0.8...v0.0.9
[v0.0.8]: https://github.com/kevinelliott/agentpipe/releases/tag/v0.0.8
