#!/bin/bash

echo "🔨 Building Parental Monitor CLI..."

# Build for Windows (main target)
echo "  Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/parental-monitor.exe .

# Build for Linux
echo "  Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/parental-monitor-linux .

# Build for macOS
echo "  Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/parental-monitor-mac .

echo "✅ Build complete! Binaries in ./bin/"
echo ""
echo "Run with:"
echo "  ./bin/parental-monitor.exe -env .env"