#!/bin/bash

# AgentPipe Installation Script
# This script downloads and installs the latest release of AgentPipe

set -e

REPO="kevinelliott/agentpipe"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="agentpipe"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "🚀 Installing AgentPipe..."
echo "   OS: $OS"
echo "   Architecture: $ARCH"
echo ""

# Get latest release URL
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo "❌ Could not determine latest release. Installing from source..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo "❌ Go is not installed. Please install Go first."
        echo "   Visit: https://golang.org/doc/install"
        exit 1
    fi
    
    # Install from source
    echo "📦 Installing from source..."
    go install github.com/${REPO}@latest
    
    echo "✅ AgentPipe installed successfully!"
    echo "   Run 'agentpipe doctor' to check available agents"
    exit 0
fi

# Download binary
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}_${OS}_${ARCH}.tar.gz"

echo "📦 Downloading AgentPipe $LATEST_RELEASE..."
curl -L -o /tmp/agentpipe.tar.gz "$DOWNLOAD_URL" 2>/dev/null || {
    echo "❌ Download failed. Installing from source instead..."
    go install github.com/${REPO}@latest
    echo "✅ AgentPipe installed successfully!"
    exit 0
}

# Extract and install
echo "📦 Installing to $INSTALL_DIR..."
tar -xzf /tmp/agentpipe.tar.gz -C /tmp
sudo mv /tmp/$BINARY_NAME $INSTALL_DIR/
sudo chmod +x $INSTALL_DIR/$BINARY_NAME

# Clean up
rm /tmp/agentpipe.tar.gz

echo "✅ AgentPipe installed successfully!"
echo "   Run 'agentpipe doctor' to check available agents"