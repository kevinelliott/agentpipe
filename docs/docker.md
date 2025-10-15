# Docker Guide

This guide covers using AgentPipe with Docker for containerized deployments.

## Table of Contents

- [Quick Start](#quick-start)
- [Building Images](#building-images)
- [Running Containers](#running-containers)
- [Docker Compose](#docker-compose)
- [Configuration](#configuration)
- [Development](#development)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Using Pre-built Image

```bash
# Pull the latest image
docker pull kevinelliott/agentpipe:latest

# Run with example configuration
docker run --rm -it \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  kevinelliott/agentpipe:latest run -c /home/agentpipe/config.yaml
```

### Using Docker Compose

```bash
# Clone repository
git clone https://github.com/kevinelliott/agentpipe.git
cd agentpipe

# Create configuration
cp examples/brainstorm.yaml config.yaml

# Start with docker-compose
docker-compose up
```

## Building Images

### Production Image

```bash
# Build with current version
docker build -t agentpipe:latest .

# Build with specific version
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  -t agentpipe:v1.0.0 \
  .

# Build using Makefile
make docker-build
```

### Development Image

```bash
# Build development image with hot reload
docker build -f Dockerfile.dev -t agentpipe:dev .

# Or using Makefile
make docker-build-dev
```

### Multi-platform Build

```bash
# Build for multiple architectures
docker buildx create --name agentpipe-builder --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t agentpipe:latest \
  --push \
  .
```

## Running Containers

### Basic Usage

```bash
# Show help
docker run --rm agentpipe:latest --help

# Show version
docker run --rm agentpipe:latest version

# Run doctor command
docker run --rm agentpipe:latest doctor
```

### With Configuration File

```bash
# Mount config from host
docker run --rm -it \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  agentpipe:latest run -c /home/agentpipe/config.yaml
```

### With Persistent Logs

```bash
# Create volume for logs
docker volume create agentpipe-logs

# Run with persistent logs
docker run --rm -it \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  -v agentpipe-logs:/home/agentpipe/.agentpipe/chats \
  agentpipe:latest run -c /home/agentpipe/config.yaml
```

### With Environment Variables

```bash
# Pass API keys via environment
docker run --rm -it \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -e GOOGLE_API_KEY=$GOOGLE_API_KEY \
  agentpipe:latest run -c /home/agentpipe/config.yaml
```

### Interactive Mode

```bash
# Run with TUI
docker run --rm -it \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  agentpipe:latest run -t -c /home/agentpipe/config.yaml
```

## Docker Compose

### Basic Configuration

```yaml
version: '3.8'

services:
  agentpipe:
    image: agentpipe:latest
    volumes:
      - ./config.yaml:/home/agentpipe/config.yaml:ro
      - agentpipe-logs:/home/agentpipe/.agentpipe/chats
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    command: ["run", "-c", "/home/agentpipe/config.yaml"]

volumes:
  agentpipe-logs:
```

### Starting Services

```bash
# Start in background
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Remove volumes
docker-compose down -v
```

### Development Setup

```bash
# Start development environment
docker-compose --profile dev up agentpipe-dev

# This mounts source code for hot reload
```

## Configuration

### Volume Mounts

```bash
# Configuration file (read-only)
-v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro

# Examples directory (read-only)
-v $(pwd)/examples:/home/agentpipe/examples:ro

# Logs directory (read-write)
-v agentpipe-logs:/home/agentpipe/.agentpipe/chats

# Custom log directory
-v $(pwd)/logs:/home/agentpipe/.agentpipe/chats
```

### Environment Variables

```bash
# Log directory
-e AGENTPIPE_LOG_DIR=/home/agentpipe/.agentpipe/chats

# Config file path
-e AGENTPIPE_CONFIG=/home/agentpipe/config.yaml

# API keys
-e ANTHROPIC_API_KEY=your-key-here
-e GOOGLE_API_KEY=your-key-here
-e OPENAI_API_KEY=your-key-here
```

### Resource Limits

```yaml
services:
  agentpipe:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Development

### Hot Reload Development

```bash
# Build development image
docker build -f Dockerfile.dev -t agentpipe:dev .

# Run with source code mounted
docker run --rm -it \
  -v $(pwd):/app \
  agentpipe:dev
```

### Running Tests in Container

```bash
# Run all tests
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  golang:1.24-alpine \
  go test -v -race ./...

# Run specific tests
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  golang:1.24-alpine \
  go test -v ./pkg/orchestrator/
```

### Building in Container

```bash
# Build binary in container
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  golang:1.24-alpine \
  go build -o agentpipe .

# Extract binary
docker cp agentpipe:/app/agentpipe ./
```

## Best Practices

### Security

1. **Run as Non-root User**
   ```dockerfile
   USER agentpipe
   ```

2. **Read-only Filesystems**
   ```bash
   docker run --read-only \
     --tmpfs /tmp \
     agentpipe:latest
   ```

3. **Minimize Image Size**
   - Use multi-stage builds
   - Use Alpine base image
   - Remove unnecessary files

4. **Scan for Vulnerabilities**
   ```bash
   docker scan agentpipe:latest
   ```

### Performance

1. **Use BuildKit**
   ```bash
   DOCKER_BUILDKIT=1 docker build -t agentpipe:latest .
   ```

2. **Layer Caching**
   - Copy go.mod/go.sum first
   - Download dependencies before copying source

3. **Resource Limits**
   - Set appropriate CPU/memory limits
   - Monitor container resource usage

### Reliability

1. **Health Checks**
   ```dockerfile
   HEALTHCHECK --interval=30s --timeout=3s \
     CMD agentpipe version || exit 1
   ```

2. **Restart Policies**
   ```yaml
   restart: unless-stopped
   ```

3. **Log Rotation**
   ```yaml
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs <container-id>

# Inspect container
docker inspect <container-id>

# Run with interactive shell
docker run --rm -it --entrypoint /bin/sh agentpipe:latest
```

### Permission Issues

```bash
# Check user in container
docker run --rm agentpipe:latest id

# Fix volume permissions
sudo chown -R 1000:1000 ./logs

# Or run as root (not recommended)
docker run --rm --user root agentpipe:latest
```

### Configuration Not Found

```bash
# Verify mount
docker run --rm \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  agentpipe:latest \
  ls -la /home/agentpipe/

# Check file exists on host
ls -la config.yaml
```

### Performance Issues

```bash
# Check resource usage
docker stats <container-id>

# Increase limits
docker run --cpus=2 --memory=2g agentpipe:latest

# Check logs for errors
docker logs --tail 100 <container-id>
```

### Network Issues

```bash
# Test network connectivity
docker run --rm agentpipe:latest ping -c 3 anthropic.com

# Check DNS
docker run --rm agentpipe:latest nslookup anthropic.com

# Use host network
docker run --network host agentpipe:latest
```

## Publishing Images

### Docker Hub

```bash
# Login
docker login

# Tag image
docker tag agentpipe:latest kevinelliott/agentpipe:latest
docker tag agentpipe:latest kevinelliott/agentpipe:v1.0.0

# Push image
docker push kevinelliott/agentpipe:latest
docker push kevinelliott/agentpipe:v1.0.0

# Or use Makefile
make docker-push
```

### GitHub Container Registry

```bash
# Login
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Tag image
docker tag agentpipe:latest ghcr.io/kevinelliott/agentpipe:latest

# Push image
docker push ghcr.io/kevinelliott/agentpipe:latest
```

## Examples

### Simple Run

```bash
docker run --rm -it \
  -v $(pwd)/examples/brainstorm.yaml:/config.yaml:ro \
  agentpipe:latest run -c /config.yaml
```

### With All Options

```bash
docker run --rm -it \
  --name agentpipe-conversation \
  -v $(pwd)/config.yaml:/home/agentpipe/config.yaml:ro \
  -v agentpipe-logs:/home/agentpipe/.agentpipe/chats \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -e GOOGLE_API_KEY=$GOOGLE_API_KEY \
  --cpus=2 \
  --memory=1g \
  --restart=unless-stopped \
  agentpipe:latest run -t -c /home/agentpipe/config.yaml
```

### Using Makefile

```bash
# Build and run
make docker-build
make docker-run ARGS="run -c examples/brainstorm.yaml"

# Development
make dev

# Build and push
make docker-push-latest
```

## Additional Resources

- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Multi-stage Builds](https://docs.docker.com/develop/develop-images/multistage-build/)
- [Docker Security](https://docs.docker.com/engine/security/)
