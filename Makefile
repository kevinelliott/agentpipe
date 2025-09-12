.PHONY: build test clean install release-build release

# Variables
BINARY_NAME=agentpipe
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/kevinelliott/agentpipe/internal/version.Version=$(VERSION) \
	-X github.com/kevinelliott/agentpipe/internal/version.CommitHash=$(COMMIT_HASH) \
	-X github.com/kevinelliott/agentpipe/internal/version.BuildDate=$(BUILD_DATE) -s -w"
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Install locally
install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

# Build for all platforms
release-build: clean
	mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build $(LDFLAGS) \
			-o dist/$(BINARY_NAME)_$${platform%/*}_$${platform#*/}$(if $(findstring windows,$${platform}),.exe,) .; \
		if [ "$${platform%/*}" != "windows" ]; then \
			tar -czf dist/$(BINARY_NAME)_$${platform%/*}_$${platform#*/}.tar.gz \
				-C dist $(BINARY_NAME)_$${platform%/*}_$${platform#*/}; \
		else \
			cd dist && zip $(BINARY_NAME)_$${platform%/*}_$${platform#*/}.zip \
				$(BINARY_NAME)_$${platform%/*}_$${platform#*/}.exe && cd ..; \
		fi; \
	done

# Create a new release (requires gh CLI)
release: release-build
	@if [ -z "$(VERSION)" ]; then \
		echo "Please specify VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	gh release create $(VERSION) dist/*.tar.gz dist/*.zip \
		--title "AgentPipe $(VERSION)" \
		--notes "Release $(VERSION)" \
		--draft

# Development commands
run:
	go run . doctor

run-example:
	go run . run -c examples/simple-conversation.yaml

run-tui:
	go run . run -c examples/brainstorm.yaml --enhanced-tui

version: build
	./$(BINARY_NAME) version

check-version: build
	./$(BINARY_NAME) -V

# Linting (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	gofumpt -w .

# Update dependencies
deps:
	go mod tidy
	go mod download

# Check for security vulnerabilities
audit:
	go list -json -deps ./... | nancy sleuth

# Generate documentation
docs:
	go doc -all > API.md

help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Build and install locally"
	@echo "  release-build - Build for all platforms"
	@echo "  release       - Create a new GitHub release"
	@echo "  run           - Run doctor command"
	@echo "  run-example   - Run simple conversation example"
	@echo "  run-tui       - Run enhanced TUI example"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  deps          - Update dependencies"
	@echo "  audit         - Check for vulnerabilities"
	@echo "  docs          - Generate documentation"