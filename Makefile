.PHONY: help build test lint clean docker-build docker-run docker-push install uninstall dev

# Variables
BINARY_NAME=agentpipe
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DOCKER_IMAGE=agentpipe
DOCKER_TAG?=latest
DOCKER_REGISTRY?=docker.io/kevinelliott

# Installation paths
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
INSTALL?=install

# Go build flags
LDFLAGS=-ldflags "-w -s \
	-X github.com/kevinelliott/agentpipe/internal/version.Version=$(VERSION) \
	-X github.com/kevinelliott/agentpipe/internal/version.CommitHash=$(COMMIT) \
	-X github.com/kevinelliott/agentpipe/internal/version.BuildDate=$(DATE)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test: ## Run tests
	@echo "Running tests..."
	go test -v -race ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run --timeout=5m

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	go clean -cache -testcache

install: build ## Install binary to system (default: /usr/local/bin, override with PREFIX)
	@echo "Installing $(BINARY_NAME) to $(BINDIR)..."
	@mkdir -p $(BINDIR)
	$(INSTALL) -m 755 $(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(BINDIR)/$(BINARY_NAME)"
	@echo "Run '$(BINARY_NAME) --help' to get started"

uninstall: ## Uninstall binary from system
	@echo "Uninstalling $(BINARY_NAME) from $(BINDIR)..."
	rm -f $(BINDIR)/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.DEFAULT_GOAL := help
