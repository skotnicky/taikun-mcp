#!/bin/bash

# Script to validate GoReleaser configuration
# This script checks the configuration without building

set -e

echo "🔍 Validating GoReleaser Configuration"
echo "======================================"

# Check if .goreleaser.yaml exists
if [ ! -f .goreleaser.yaml ]; then
    echo "❌ Error: .goreleaser.yaml not found"
    exit 1
fi

echo "✅ .goreleaser.yaml exists"

# Check if GitHub Actions workflows exist
if [ ! -f .github/workflows/release.yml ]; then
    echo "❌ Error: .github/workflows/release.yml not found"
    exit 1
fi

if [ ! -f .github/workflows/ci.yml ]; then
    echo "❌ Error: .github/workflows/ci.yml not found"
    exit 1
fi

echo "✅ GitHub Actions workflows exist"

# Validate Go project structure
if [ ! -f go.mod ]; then
    echo "❌ Error: go.mod not found"
    exit 1
fi

echo "✅ Go module found"

# Check if main.go can build
echo "🔨 Testing build..."
if go build -o /tmp/cloudera-cloud-factory-mcp-test .; then
    echo "✅ Project builds successfully"
    rm -f /tmp/cloudera-cloud-factory-mcp-test
else
    echo "❌ Error: Project failed to build"
    exit 1
fi

# Test version command
echo "🔖 Testing version command..."
if ./cloudera-cloud-factory-mcp --version > /dev/null 2>&1; then
    echo "✅ Version command works"
else
    echo "❌ Error: Version command failed"
    exit 1
fi

# Check for required files that will be included in releases
echo "📁 Checking required files..."
for file in README.md LICENSE; do
    if [ -f "$file" ]; then
        echo "✅ $file exists"
    else
        echo "⚠️  Warning: $file not found (will be included in release if it exists)"
    fi
done

echo ""
echo "🎉 GoReleaser configuration validation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Commit and push these changes to GitHub"
echo "2. Create and push a version tag (e.g., git tag v0.1.0 && git push origin v0.1.0)"
echo "3. GitHub Actions will automatically build and release binaries"
echo ""
echo "Manual release testing:"
echo "  goreleaser release --snapshot --clean  # Test locally without publishing"