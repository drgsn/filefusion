.PHONY: all build install clean uninstall

# Go parameters
BINARY_NAME=filefusion
MAIN_FILE=cmd/filefusion/main.go
INSTALL_PATH=/usr/local/bin
GOPATH=$(shell go env GOPATH)
BUILD_DIR=build
COVER_DIR=coverage

all: build install

# Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	@go build -o ${BINARY_NAME} ${MAIN_FILE}
	@echo "Build complete"

# Install to /usr/local/bin
install: build
	@echo "Installing ${BINARY_NAME} to ${INSTALL_PATH}..."
	@if [ -f "${INSTALL_PATH}/${BINARY_NAME}" ]; then \
		sudo rm ${INSTALL_PATH}/${BINARY_NAME}; \
	fi
	@sudo cp ${BINARY_NAME} ${INSTALL_PATH}
	@echo "Installation complete. You can now use '${BINARY_NAME}' from any directory"

# Install using go install
install-go:
	@echo "Installing using go install..."
	@go install ./cmd/filefusion
	@echo "Installation complete. Ensure ${GOPATH}/bin is in your PATH"
	@echo "Current PATH: ${PATH}"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(COVER_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVER_DIR)
	@go test ./... -coverprofile=$(COVER_DIR)/coverage.out
	@go tool cover -html=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/coverage.html
	@echo "Coverage report generated at $(COVER_DIR)/coverage.html"

# Uninstall the binary
uninstall:
	@echo "Uninstalling ${BINARY_NAME}..."
	@if [ -f "${INSTALL_PATH}/${BINARY_NAME}" ]; then \
		sudo rm ${INSTALL_PATH}/${BINARY_NAME}; \
		echo "${BINARY_NAME} has been removed from ${INSTALL_PATH}"; \
	else \
		echo "${BINARY_NAME} is not installed in ${INSTALL_PATH}"; \
	fi

# Show help
help:
	@echo "Available commands:"
	@echo "  make build      - Build the binary"
	@echo "  make install    - Build and install to /usr/local/bin (requires sudo)"
	@echo "  make install-go - Install using 'go install' (recommended)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make uninstall  - Remove installed binary from /usr/local/bin"
	@echo "  make help       - Show this help message"