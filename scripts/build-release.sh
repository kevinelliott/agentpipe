#!/bin/bash

# Build multi-architecture releases for AgentPipe
# This script creates binaries for all supported platforms

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the version
VERSION=${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v${VERSION}"
fi

echo -e "${GREEN}🚀 Building AgentPipe ${VERSION} for multiple architectures${NC}"
echo ""

# Create dist directory
rm -rf dist
mkdir -p dist

# Binary name
BINARY="agentpipe"

# Build function
build_for_platform() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="${BINARY}_${GOOS}_${GOARCH}"
    
    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH}...${NC}"
    
    # Add .exe extension for Windows
    local BINARY_NAME="${BINARY}"
    if [ "$GOOS" = "windows" ]; then
        BINARY_NAME="${BINARY}.exe"
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    # Build
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.Version=${VERSION} -s -w" \
        -o "dist/${OUTPUT_NAME}" .
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Built ${OUTPUT_NAME}${NC}"
        
        # Create archive
        if [ "$GOOS" = "windows" ]; then
            # Create zip for Windows
            cd dist
            zip -q "${BINARY}_${GOOS}_${GOARCH}.zip" "${OUTPUT_NAME}"
            rm "${OUTPUT_NAME}"
            cd ..
            echo -e "${GREEN}✓ Created ${BINARY}_${GOOS}_${GOARCH}.zip${NC}"
        else
            # Create tar.gz for Unix-like systems
            cd dist
            tar -czf "${BINARY}_${GOOS}_${GOARCH}.tar.gz" "${OUTPUT_NAME}"
            rm "${OUTPUT_NAME}"
            cd ..
            echo -e "${GREEN}✓ Created ${BINARY}_${GOOS}_${GOARCH}.tar.gz${NC}"
        fi
    else
        echo -e "${RED}✗ Failed to build for ${GOOS}/${GOARCH}${NC}"
        return 1
    fi
    echo ""
}

# Build for all platforms
echo -e "${GREEN}📦 Building distributions...${NC}"
echo ""

# macOS
build_for_platform "darwin" "amd64"  # Intel Macs
build_for_platform "darwin" "arm64"  # M1/M2 Macs

# Linux
build_for_platform "linux" "amd64"   # Intel/AMD Linux
build_for_platform "linux" "arm64"   # ARM Linux (Raspberry Pi, etc.)
build_for_platform "linux" "386"     # 32-bit Linux

# Windows
build_for_platform "windows" "amd64" # 64-bit Windows
build_for_platform "windows" "386"   # 32-bit Windows
build_for_platform "windows" "arm64" # ARM Windows

echo -e "${GREEN}📊 Calculating SHA256 checksums...${NC}"
echo ""

# Calculate SHA256 for all archives
cd dist
shasum -a 256 *.tar.gz *.zip > checksums.txt 2>/dev/null || sha256sum *.tar.gz *.zip > checksums.txt

echo -e "${GREEN}📄 Checksums:${NC}"
cat checksums.txt
echo ""

# Create a release notes template
cat > RELEASE_NOTES.md << EOF
# AgentPipe ${VERSION}

## What's New
- Multi-agent conversation orchestration
- Enhanced TUI with panelized interface
- Response metrics tracking (duration, tokens, cost)
- Automatic chat logging
- Support for Claude, Gemini, Qwen Code, Codex, and Ollama

## Installation

### Homebrew
\`\`\`bash
brew tap kevinelliott/tap
brew install agentpipe
\`\`\`

### Direct Download
Download the appropriate archive for your platform from the assets below.

### Checksums
\`\`\`
$(cat checksums.txt)
\`\`\`

## Quick Start
\`\`\`bash
# Check available agents
agentpipe doctor

# Start a conversation
agentpipe run -a claude:Alice -a gemini:Bob -p "Hello!"
\`\`\`

## Full Documentation
See the [README](https://github.com/kevinelliott/agentpipe#readme) for complete documentation.
EOF

cd ..

echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo -e "${GREEN}📦 Distribution files created in ./dist/${NC}"
echo ""
ls -lh dist/
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "1. Review the release notes in dist/RELEASE_NOTES.md"
echo "2. Create a GitHub release:"
echo "   ${YELLOW}gh release create ${VERSION} dist/*.tar.gz dist/*.zip \\
      --title \"AgentPipe ${VERSION}\" \\
      --notes-file dist/RELEASE_NOTES.md \\
      --draft${NC}"
echo ""
echo "3. Update Homebrew formula with these SHA256 values:"
echo ""
grep darwin_arm64 dist/checksums.txt
grep darwin_amd64 dist/checksums.txt
grep linux_arm64 dist/checksums.txt
grep linux_amd64 dist/checksums.txt