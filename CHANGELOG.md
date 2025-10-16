# Changelog

All notable changes to AgentPipe will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
- **Amp CLI Agent Support**: Integration with Sourcegraph's Amp coding agent
  - Execute mode and streaming support
  - Autonomous coding capabilities
  - IDE integration support

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

[Unreleased]: https://github.com/kevinelliott/agentpipe/compare/v0.0.16...HEAD
[v0.0.16]: https://github.com/kevinelliott/agentpipe/compare/v0.0.15...v0.0.16
[v0.0.15]: https://github.com/kevinelliott/agentpipe/compare/v0.0.9...v0.0.15
[v0.0.9]: https://github.com/kevinelliott/agentpipe/compare/v0.0.8...v0.0.9
[v0.0.8]: https://github.com/kevinelliott/agentpipe/releases/tag/v0.0.8
