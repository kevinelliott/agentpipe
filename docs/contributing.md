# Contributing to AgentPipe

Thank you for your interest in contributing to AgentPipe! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Adding New Features](#adding-new-features)
- [Documentation](#documentation)
- [Release Process](#release-process)

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on what is best for the community
- Show empathy towards other community members

### Unacceptable Behavior

- Harassment or discrimination of any kind
- Trolling or insulting comments
- Publishing others' private information
- Other conduct inappropriate in a professional setting

## Getting Started

### Prerequisites

- **Go 1.25+**: Required for building the project
- **Git**: For version control
- **golangci-lint**: For code linting
- **make**: For build automation (optional)

### Development Setup

1. **Fork the repository**
   ```bash
   # Click "Fork" on GitHub, then clone your fork
   git clone https://github.com/YOUR_USERNAME/agentpipe.git
   cd agentpipe
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/kevinelliott/agentpipe.git
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Install development tools**
   ```bash
   # Install golangci-lint
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

   # Install goimports
   go install golang.org/x/tools/cmd/goimports@latest
   ```

5. **Verify setup**
   ```bash
   go test ./...
   golangci-lint run
   ```

## Development Workflow

### 1. Create a Branch

```bash
# Update your fork
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
```

### Branch Naming Conventions

- `feature/description`: New features
- `fix/description`: Bug fixes
- `docs/description`: Documentation changes
- `refactor/description`: Code refactoring
- `test/description`: Test additions/changes
- `chore/description`: Build/tooling changes

### 2. Make Changes

- Write clear, idiomatic Go code
- Follow existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run unit tests
go test ./...

# Run with race detection
go test -race ./...

# Run integration tests
go test ./test/integration/

# Run benchmarks
go test -bench=. ./test/benchmark/

# Run linter
golangci-lint run

# Format code
gofmt -w .
goimports -local github.com/kevinelliott/agentpipe -w .
```

### 4. Commit Your Changes

```bash
git add .
git commit -m "feat: add new feature"
```

See [Commit Guidelines](#commit-guidelines) below.

### 5. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Coding Standards

### Go Style Guide

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

### Key Principles

1. **Simplicity**: Simple code is better than clever code
2. **Readability**: Code is read more than it's written
3. **Consistency**: Follow existing patterns
4. **Documentation**: Public APIs must be documented
5. **Error Handling**: Always handle errors explicitly

### Code Organization

```
pkg/
├── agent/          # Agent interfaces and base types
├── adapters/       # Agent implementations
├── orchestrator/   # Conversation orchestration
├── config/         # Configuration management
├── logger/         # Logging system
├── ratelimit/      # Rate limiting
├── errors/         # Error types
├── tui/            # Terminal UI
└── utils/          # Utility functions
```

### Naming Conventions

**Packages:**
- All lowercase
- Short, descriptive names
- No underscores

**Files:**
- Lowercase with underscores
- `agent.go`, `orchestrator_test.go`

**Types:**
- PascalCase for exported types
- camelCase for unexported types

**Functions:**
- PascalCase for exported functions
- camelCase for unexported functions
- Descriptive verb names: `GetMessages`, `SendMessage`

**Constants:**
- PascalCase for exported constants
- camelCase for unexported constants

**Variables:**
- camelCase for all variables
- Short names in small scopes
- Descriptive names in large scopes

### Error Handling

```go
// Good: Return errors, don't panic
func DoSomething() error {
    if err := validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}

// Bad: Panic on errors
func DoSomething() {
    if err := validate(); err != nil {
        panic(err)
    }
}
```

### Context Usage

```go
// Good: Accept context as first parameter
func SendMessage(ctx context.Context, msg string) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Do work
    }
    return nil
}
```

### Interface Design

```go
// Good: Small, focused interfaces
type Reader interface {
    Read([]byte) (int, error)
}

// Bad: Large, unfocused interfaces
type Everything interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    Sync() error
    // ... many more methods
}
```

## Testing Guidelines

### Test Organization

- Unit tests: `*_test.go` in same package
- Integration tests: `test/integration/`
- Benchmark tests: `test/benchmark/`

### Test Naming

```go
func TestFunctionName(t *testing.T)           // Basic test
func TestFunctionName_EdgeCase(t *testing.T)  // Specific case
func BenchmarkFunctionName(b *testing.B)      // Benchmark
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
        wantErr bool
    }{
        {"valid input", "test", true, false},
        {"empty input", "", false, true},
        {"invalid input", "!", false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Validate() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Coverage

- **Minimum**: 80% coverage for new code
- **Target**: 90%+ coverage
- **Critical paths**: 100% coverage

```bash
# Check coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Mocking

Use interfaces for mockability:

```go
// Define interface
type Storage interface {
    Save(data []byte) error
    Load() ([]byte, error)
}

// Mock in tests
type MockStorage struct {
    SaveFunc func([]byte) error
    LoadFunc func() ([]byte, error)
}

func (m *MockStorage) Save(data []byte) error {
    return m.SaveFunc(data)
}
```

## Commit Guidelines

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Build/tooling changes
- `perf`: Performance improvements

### Examples

```
feat(orchestrator): add retry logic with exponential backoff

Implements configurable retry logic with exponential backoff for
failed agent responses. Supports MaxRetries, InitialDelay, MaxDelay,
and Multiplier configuration.

Closes #123
```

```
fix(ratelimit): prevent race condition in token refill

Added mutex protection around token refill calculation to prevent
race condition when multiple goroutines check availability.

Fixes #456
```

### Rules

1. Use present tense ("add feature" not "added feature")
2. Use imperative mood ("move cursor" not "moves cursor")
3. First line ≤ 72 characters
4. Separate subject from body with blank line
5. Wrap body at 72 characters
6. Reference issues in footer

## Pull Request Process

### Before Submitting

- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] Linter shows no issues
- [ ] Documentation updated
- [ ] CHANGELOG updated (for significant changes)
- [ ] Commits are clean and well-formatted

### PR Description Template

```markdown
## Description
Brief description of changes

## Motivation
Why is this change needed?

## Changes
- Change 1
- Change 2
- Change 3

## Testing
How was this tested?

## Screenshots (if applicable)

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] CHANGELOG updated
- [ ] Breaking changes noted
```

### Review Process

1. **Automated Checks**: CI must pass
2. **Code Review**: At least one approval required
3. **Testing**: Reviewer verifies functionality
4. **Merge**: Squash merge to main branch

### Addressing Feedback

```bash
# Make changes based on feedback
git add .
git commit -m "address review feedback"
git push origin feature/your-feature-name
```

## Adding New Features

### New Agent Type

1. Create adapter in `pkg/adapters/`
2. Implement `Agent` interface
3. Add tests in `pkg/adapters/*_test.go`
4. Add integration test in `test/integration/`
5. Update documentation in `README.md` and `docs/`
6. Add example configuration

### New Orchestration Mode

1. Add mode constant in `pkg/orchestrator/orchestrator.go`
2. Implement mode logic
3. Add comprehensive tests
4. Update configuration schema
5. Document in `docs/architecture.md`

### New Command

1. Create command file in `cmd/`
2. Register with Cobra
3. Add tests
4. Update help text
5. Document in `README.md`

## Documentation

### Code Documentation

All exported functions, types, and constants must have godoc comments:

```go
// Agent represents an AI agent that can participate in conversations.
// It provides methods for sending messages, health checking, and configuration.
type Agent interface {
    // GetID returns the unique identifier of the agent.
    GetID() string

    // SendMessage sends a message to the agent and returns the response.
    // The context can be used to cancel the operation or set timeouts.
    SendMessage(ctx context.Context, messages []Message) (string, error)
}
```

### README Updates

Update `README.md` when adding:
- New features
- New configuration options
- New commands
- Breaking changes

### CHANGELOG Updates

Add entries to `CHANGELOG.md` for:
- New features
- Bug fixes
- Breaking changes
- Deprecations

Format:
```markdown
## [Unreleased]

### Added
- New feature description

### Changed
- Changed behavior description

### Fixed
- Bug fix description

### Deprecated
- Deprecated feature description
```

## Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. Update version in `internal/version/version.go`
2. Update `CHANGELOG.md`
3. Create git tag: `git tag v1.2.3`
4. Push tag: `git push origin v1.2.3`
5. GitHub Actions will create release and build binaries
6. Update Homebrew formula

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue
- **Security**: Email security@example.com
- **Chat**: Join our Discord/Slack

## Recognition

Contributors are recognized in:
- `CONTRIBUTORS.md`
- Release notes
- GitHub contributors page

Thank you for contributing to AgentPipe! 🎉
