# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/kevinelliott/agentpipe/internal/version.Version=${VERSION:-dev} \
              -X github.com/kevinelliott/agentpipe/internal/version.Commit=${COMMIT:-unknown} \
              -X github.com/kevinelliott/agentpipe/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o agentpipe .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 agentpipe && \
    adduser -D -u 1000 -G agentpipe agentpipe

# Set working directory
WORKDIR /home/agentpipe

# Copy binary from builder
COPY --from=builder /app/agentpipe /usr/local/bin/agentpipe

# Copy example configs
COPY --from=builder /app/examples /home/agentpipe/examples

# Create directories for configs and logs
RUN mkdir -p /home/agentpipe/.agentpipe/chats && \
    chown -R agentpipe:agentpipe /home/agentpipe

# Switch to non-root user
USER agentpipe

# Set default environment variables
ENV AGENTPIPE_LOG_DIR=/home/agentpipe/.agentpipe/chats
ENV AGENTPIPE_CONFIG=/home/agentpipe/config.yaml

# Volume for configuration and logs
VOLUME ["/home/agentpipe/.agentpipe"]

# Expose port (if we add HTTP API in the future)
# EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD agentpipe version || exit 1

# Default command
ENTRYPOINT ["agentpipe"]
CMD ["--help"]
