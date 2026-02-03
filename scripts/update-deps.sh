#!/bin/bash

# Cloudera Cloud Factory MCP Server - Dependency Update Script
# This script updates all Go dependencies to their latest versions

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in a Go module directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod file not found. Make sure you're in the project root directory."
    exit 1
fi

print_status "Starting dependency update process..."

# Backup current go.mod and go.sum
print_status "Creating backup of current dependency files..."
cp go.mod go.mod.backup
if [ -f "go.sum" ]; then
    cp go.sum go.sum.backup
fi
print_success "Backup created (go.mod.backup, go.sum.backup)"

# Show current versions before update
print_status "Current dependency versions:"
go list -m all | grep -v "$(go list -m)" | head -10

echo ""
print_status "Updating all dependencies to latest versions..."

# Update all dependencies
if go get -u ./...; then
    print_success "Dependencies updated successfully"
else
    print_error "Failed to update dependencies"
    print_warning "Restoring from backup..."
    mv go.mod.backup go.mod
    if [ -f "go.sum.backup" ]; then
        mv go.sum.backup go.sum
    fi
    exit 1
fi

# Tidy up the modules
print_status "Tidying up go.mod..."
if go mod tidy; then
    print_success "go mod tidy completed successfully"
else
    print_error "go mod tidy failed"
    print_warning "Restoring from backup..."
    mv go.mod.backup go.mod
    if [ -f "go.sum.backup" ]; then
        mv go.sum.backup go.sum
    fi
    exit 1
fi

# Show updated versions
echo ""
print_status "Updated dependency versions:"
go list -m all | grep -v "$(go list -m)" | head -10

# Test if the project still builds
echo ""
print_status "Testing if project builds with updated dependencies..."
if go build -o /tmp/taikun-mcp-test ./...; then
    rm -f /tmp/taikun-mcp-test
    print_success "Project builds successfully with updated dependencies"
else
    print_error "Project failed to build with updated dependencies"
    print_warning "Restoring from backup..."
    mv go.mod.backup go.mod
    if [ -f "go.sum.backup" ]; then
        mv go.sum.backup go.sum
    fi
    exit 1
fi

# Run tests if they exist
if go list ./... | grep -q test; then
    print_status "Running tests with updated dependencies..."
    if go test ./...; then
        print_success "All tests passed with updated dependencies"
    else
        print_warning "Some tests failed with updated dependencies"
        print_warning "Consider reviewing the test failures before committing"
    fi
else
    print_status "No tests found, skipping test execution"
fi

# Check for security vulnerabilities
print_status "Checking for security vulnerabilities..."
if command -v govulncheck >/dev/null 2>&1; then
    if govulncheck ./...; then
        print_success "No security vulnerabilities found"
    else
        print_warning "Security vulnerabilities detected. Please review and address them."
    fi
else
    print_warning "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# Show differences
echo ""
print_status "Dependency changes:"
if command -v diff >/dev/null 2>&1; then
    if diff -u go.mod.backup go.mod || true; then
        echo ""
    fi
else
    print_status "diff command not available, showing new go.mod content:"
    cat go.mod
fi

# Clean up backup files
print_status "Cleaning up backup files..."
rm -f go.mod.backup go.sum.backup

echo ""
print_success "Dependency update completed successfully!"
print_status "Summary:"
echo "  - All dependencies updated to latest versions"
echo "  - Project builds successfully"
echo "  - go.mod and go.sum files updated"
echo ""
print_status "Next steps:"
echo "  1. Review the changes: git diff go.mod go.sum"
echo "  2. Test your application thoroughly"
echo "  3. Commit the changes: git add go.mod go.sum && git commit -m 'Update dependencies'"
echo ""
print_warning "Note: Always test thoroughly after updating dependencies!"