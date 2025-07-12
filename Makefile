# Taikun MCP Server Makefile

# Variables
BINARY_NAME=taikun-mcp
BUILD_DIR=build
SCRIPTS_DIR=scripts

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) .

# Build for production (with optimizations)
.PHONY: build-prod
build-prod: create-build-dir
	@echo "Building $(BINARY_NAME) for production..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# Run unit tests only
.PHONY: test
test:
	@echo "Running unit tests..."
	$(GOTEST) -v -short ./...

# Run all tests including integration tests
.PHONY: test-all
test-all:
	@echo "Running all tests..."
	$(GOTEST) -v ./...

# Run integration tests only (requires TAIKUN credentials)
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@echo "Make sure TAIKUN_EMAIL and TAIKUN_PASSWORD are set"
	$(GOTEST) -v -tags=integration -run TestIntegration ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -short -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Download and tidy dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	@chmod +x $(SCRIPTS_DIR)/update-deps.sh
	@$(SCRIPTS_DIR)/update-deps.sh

# Check for outdated dependencies
.PHONY: check-deps
check-deps:
	@echo "Checking for outdated dependencies..."
	$(GOGET) -u -t ./...
	$(GOMOD) tidy

# Run the server locally
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Run with environment variables from .env file (if exists)
.PHONY: run-env
run-env: build
	@echo "Running $(BINARY_NAME) with environment..."
	@if [ -f .env ]; then \
		set -a && source .env && set +a && ./$(BINARY_NAME); \
	else \
		echo "No .env file found, running without environment variables..."; \
		./$(BINARY_NAME); \
	fi

# Create build directory
.PHONY: create-build-dir
create-build-dir:
	mkdir -p $(BUILD_DIR)

# Create scripts directory  
$(SCRIPTS_DIR):
	mkdir -p $(SCRIPTS_DIR)


# Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Security check
.PHONY: security
security:
	@echo "Running security checks..."
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec
	gosec ./...

# Show current dependency versions
.PHONY: deps-status
deps-status:
	@echo "Current dependency versions:"
	go list -m all

# Create release
.PHONY: release
release: clean build-prod
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/release
	@cd $(BUILD_DIR) && for binary in $(BINARY_NAME)-*; do \
		if [[ $$binary == *.exe ]]; then \
			zip "release/$${binary%.exe}.zip" "$$binary"; \
		else \
			tar -czf "release/$$binary.tar.gz" "$$binary"; \
		fi; \
	done
	@echo "Release packages created in $(BUILD_DIR)/release/"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Clean and build the project"
	@echo "  build         - Build the binary"
	@echo "  build-prod    - Build optimized binaries for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  fmt           - Format code"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  update-deps   - Update all dependencies to latest versions"
	@echo "  check-deps    - Check for outdated dependencies"
	@echo "  run           - Build and run the server"
	@echo "  run-env       - Build and run with .env file"
	@echo "  install-tools - Install development tools"
	@echo "  lint          - Lint code with golangci-lint"
	@echo "  security      - Run security checks"
	@echo "  deps-status   - Show current dependency versions"
	@echo "  release       - Create release packages"
	@echo "  help          - Show this help message"