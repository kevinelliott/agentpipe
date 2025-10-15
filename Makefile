.PHONY: help build test lint clean docker-build docker-run docker-push install dev

# Variables
BINARY_NAME=agentpipe
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DOCKER_IMAGE=agentpipe
DOCKER_TAG?=latest
DOCKER_REGISTRY?=docker.io/kevinelliott

# Go build flags
LDFLAGS=-ldflags="-w -s \
	-X github.com/kevinelliott/agentpipe/internal/version.Version=$(VERSION) \
	-X github.com/kevinelliott/agentpipe/internal/version.Commit=$(COMMIT) \
	-X github.com/kevinelliott/agentpipe/internal/version.Date=$(DATE)"

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

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.DEFAULT_GOAL := help
