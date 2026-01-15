#!/bin/bash
set -e

# claude-coord installer
# Usage: curl -fsSL https://raw.githubusercontent.com/LoomLabs-Venture-Studio/claude-coord/main/install.sh | bash

REPO="LoomLabs-Venture-Studio/claude-coord"
BINARY="claude-coord"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
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

case $OS in
    darwin|linux)
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Get latest release
echo "Detecting latest version..."
LATEST=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Failed to detect latest version. Using 'latest'."
    LATEST="latest"
fi

echo "Installing claude-coord $LATEST for $OS/$ARCH..."

# Download
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/${BINARY}-${OS}-${ARCH}"
TMP_FILE=$(mktemp)

echo "Downloading from $DOWNLOAD_URL..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
chmod +x "$TMP_FILE"

# Install
echo "Installing to $INSTALL_DIR/$BINARY..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/$BINARY"
else
    echo "Need sudo to install to $INSTALL_DIR"
    sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY"
fi

echo ""
echo "✓ Installed claude-coord to $INSTALL_DIR/$BINARY"
echo ""
echo "Get started:"
echo "  cd your-project"
echo "  claude-coord init"
echo ""
