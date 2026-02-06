#!/bin/bash

# Build script for creating release binaries
# Usage: ./build-release.sh [version]
# Example: ./build-release.sh v0.1.0

set -e

VERSION=${1:-"v0.1.0"}
BUILD_DIR="dist"
BINARY_NAME="ptp"

echo "🚀 Building PTP release binaries version $VERSION"
echo ""

# Clean and create dist directory
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Build for Linux 64-bit
echo "📦 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/${BINARY_NAME}-linux-amd64 .
chmod +x $BUILD_DIR/${BINARY_NAME}-linux-amd64
tar -czf $BUILD_DIR/${BINARY_NAME}-linux-amd64.tar.gz -C $BUILD_DIR ${BINARY_NAME}-linux-amd64
echo "✅ Created: $BUILD_DIR/${BINARY_NAME}-linux-amd64.tar.gz"

# Build for macOS Intel (amd64)
echo "📦 Building for macOS Intel (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/${BINARY_NAME}-darwin-amd64 .
chmod +x $BUILD_DIR/${BINARY_NAME}-darwin-amd64
tar -czf $BUILD_DIR/${BINARY_NAME}-darwin-amd64.tar.gz -C $BUILD_DIR ${BINARY_NAME}-darwin-amd64
echo "✅ Created: $BUILD_DIR/${BINARY_NAME}-darwin-amd64.tar.gz"

# Build for macOS Apple Silicon (arm64)
echo "📦 Building for macOS Apple Silicon (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $BUILD_DIR/${BINARY_NAME}-darwin-arm64 .
chmod +x $BUILD_DIR/${BINARY_NAME}-darwin-arm64
tar -czf $BUILD_DIR/${BINARY_NAME}-darwin-arm64.tar.gz -C $BUILD_DIR ${BINARY_NAME}-darwin-arm64
echo "✅ Created: $BUILD_DIR/${BINARY_NAME}-darwin-arm64.tar.gz"

# Create checksums
echo ""
echo "🔐 Generating checksums..."
cd $BUILD_DIR
sha256sum *.tar.gz > checksums.txt
cd ..

echo ""
echo "✨ Build complete! Files are in $BUILD_DIR/:"
ls -lh $BUILD_DIR/*.tar.gz
echo ""
echo "📋 Checksums saved to $BUILD_DIR/checksums.txt"

