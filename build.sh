#!/bin/bash

echo "🔨 Building Parental Monitor CLI..."

# Build for Windows (main target)
echo "  Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/parental-monitor.exe main.go

# Build for Linux
echo "  Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/parental-monitor-linux main.go

# Build for macOS
echo "  Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/parental-monitor-mac main.go

echo "✅ Build complete! Binaries in ./bin/"
echo ""
echo "Run with:"
echo "  ./bin/parental-monitor.exe -config config.json"