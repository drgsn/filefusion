# Binary name
BINARY_NAME=filefusion
BUILD_DIR=build

# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/$(BUILD_DIR)
GOFILES=$(wildcard *.go)

# Go binary to use for builds
GO?=go

# Build version from git tag
VERSION=$(shell git describe --tags --always --dirty)

# Use linker flags to provide version/build info
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default os-arch target
TARGETOS?=linux
TARGETARCH?=amd64

# Install path
INSTALL_PATH?=/usr/local/bin

.PHONY: all build clean install uninstall test test-coverage lint fmt check docs run help

# Default target
all: check build test

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(GOBIN)
	$(GO) build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Cross compile for different platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-arm64 ./cmd/$(BINARY_NAME)

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-darwin-arm64 ./cmd/$(BINARY_NAME)

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-windows-amd64.exe ./cmd/$(BINARY_NAME)

# Install the binary
install: build
	@echo "Installing to $(INSTALL_PATH)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@sudo install -m 755 $(GOBIN)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)

# Uninstall the binary
uninstall:
	@echo "Uninstalling from $(INSTALL_PATH)..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)

# Clean build directory
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean -testcache

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Check if code is properly formatted and linted
check: fmt lint
	@echo "Running checks..."
	$(GO) mod verify
	$(GO) vet ./...

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null; then \
		echo "View documentation at http://localhost:6060/pkg/github.com/drgsn/filefusion/"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Installing..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
		echo "View documentation at http://localhost:6060/pkg/github.com/drgsn/filefusion/"; \
		godoc -http=:6060; \
	fi

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  all            - Run checks, build, and test"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms (linux, darwin, windows)"
	@echo "  clean          - Clean build directory"
	@echo "  install        - Install binary to $(INSTALL_PATH)"
	@echo "  uninstall      - Remove binary from $(INSTALL_PATH)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  check         - Run formatting and linting checks"
	@echo "  docs          - Generate and serve documentation"
	@echo "  run           - Build and run the application"
	@echo "  help          - Show this help message"

# Dependency management
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

deps-download:
	@echo "Downloading dependencies..."
	$(GO) mod download

# Version information
version:
	@echo "Version: $(VERSION)"