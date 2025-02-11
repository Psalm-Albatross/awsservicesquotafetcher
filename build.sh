#!/bin/bash

# Binary name
BINARY_NAME="awsservicesquotafetcher"

# Default version if not provided
VERSION=${VERSION:-"1.1.0"}

# Output directory
OUTPUT_DIR="dist"
mkdir -p "$OUTPUT_DIR"

# Supported platforms
PLATFORMS=("linux/amd64" "linux/arm64" "windows/amd64" "darwin/amd64" "darwin/arm64")

# Build binaries
for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo "$PLATFORM" | cut -d'/' -f1)
    ARCH=$(echo "$PLATFORM" | cut -d'/' -f2)
    OUTPUT_NAME="${BINARY_NAME}-${VERSION}-${OS}-${ARCH}"

    # Add .exe for Windows
    if [ "$OS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi

    echo "🔨 Building for $OS/$ARCH (Version: $VERSION)..."
    
    env GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_DIR/$OUTPUT_NAME" .

    if [ $? -ne 0 ]; then
        echo "❌ Build failed for $OS/$ARCH"
    else
        echo "✅ Build successful: $OUTPUT_DIR/$OUTPUT_NAME"
    fi
done

echo "🚀 All binaries are in the '$OUTPUT_DIR' directory."