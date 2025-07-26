# Framework LED Matrix Daemon Makefile

# Build configuration
BINARY_NAME=framework-led-daemon
BINARY_DIR=bin
CMD_DIR=cmd/daemon
INSTALL_DIR=/usr/local/bin
CONFIG_DIR=/etc/framework-led-daemon
SYSTEMD_DIR=/etc/systemd/system

# Go build flags
GO_BUILD_FLAGS=-ldflags="-w -s" -trimpath
CGO_ENABLED=0

# Version information
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build targets  
.PHONY: all build clean install uninstall test test-coverage test-race test-short test-bench test-ci test-clean fmt vet deps cross-compile simulator help

# Default target
all: clean deps fmt vet test-coverage build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) \
		-ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BINARY_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"

# Build and run the LED matrix simulator
simulator: deps
	@echo "Building and running LED matrix simulator..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) \
		-o $(BINARY_DIR)/framework-led-simulator ./cmd/simulator
	@echo "Starting simulator (Press Ctrl+C to stop)..."
	@echo "Try: make simulator ARGS='-mode activity -metric cpu -duration 60s'"
	./$(BINARY_DIR)/framework-led-simulator $(ARGS)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)
	@go clean
	@echo "Clean complete"

# Install the daemon system-wide
install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BINARY_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo mkdir -p $(CONFIG_DIR)
	@sudo cp configs/config.yaml $(CONFIG_DIR)/
	@sudo cp systemd/$(BINARY_NAME).service $(SYSTEMD_DIR)/
	@sudo systemctl daemon-reload
	@echo "Installation complete"
	@echo "To enable the service: sudo systemctl enable $(BINARY_NAME)"
	@echo "To start the service: sudo systemctl start $(BINARY_NAME)"

# Uninstall the daemon
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo systemctl stop $(BINARY_NAME) 2>/dev/null || true
	@sudo systemctl disable $(BINARY_NAME) 2>/dev/null || true
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo rm -f $(SYSTEMD_DIR)/$(BINARY_NAME).service
	@sudo rm -rf $(CONFIG_DIR)
	@sudo systemctl daemon-reload
	@echo "Uninstall complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Tests complete"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	@go test -v -race ./...
	@echo "Race detection tests complete"

# Run short tests only (skip integration tests)
test-short:
	@echo "Running short tests..."
	@go test -v -short ./...
	@echo "Short tests complete"

# Run specific package tests
test-config:
	@echo "Testing config package..."
	@go test -v ./internal/config

test-matrix:
	@echo "Testing matrix package..."
	@go test -v ./internal/matrix

test-stats:
	@echo "Testing stats package..."
	@go test -v ./internal/stats

test-visualizer:
	@echo "Testing visualizer package..."
	@go test -v ./internal/visualizer

test-daemon:
	@echo "Testing daemon package..."
	@go test -v ./internal/daemon

# Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	@go test -v -bench=. -benchmem ./...
	@echo "Benchmarks complete"

# Generate coverage report with threshold check
test-coverage-check:
	@echo "Running coverage check..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1 | awk '{print "Total coverage: " $$3}'
	@go tool cover -func=coverage.out | tail -1 | awk '{if($$3+0 < 70.0) {print "Coverage below 70%: " $$3; exit 1}}'
	@echo "Coverage check passed"

# Run tests in CI environment
test-ci:
	@echo "Running CI tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1 | awk '{print "Coverage: " $$3}'
	@echo "CI tests complete"

# Clean test artifacts
test-clean:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@echo "Test artifacts cleaned"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Formatting complete"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# Cross-compile for different platforms
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BINARY_DIR)
	
	@echo "Building for Linux amd64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) \
		-o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	
	@echo "Building for Linux arm64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) \
		-o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	
	@echo "Building for Windows amd64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) \
		-o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	
	@echo "Building for macOS amd64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) \
		-o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	
	@echo "Building for macOS arm64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) \
		-o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	
	@echo "Cross-compilation complete"

# Create release packages
release: cross-compile
	@echo "Creating release packages..."
	@mkdir -p $(BINARY_DIR)/release
	
	@tar -czf $(BINARY_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz \
		-C $(BINARY_DIR) $(BINARY_NAME)-linux-amd64 \
		-C .. configs/config.yaml systemd/$(BINARY_NAME).service
	
	@tar -czf $(BINARY_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz \
		-C $(BINARY_DIR) $(BINARY_NAME)-linux-arm64 \
		-C .. configs/config.yaml systemd/$(BINARY_NAME).service
	
	@zip -j $(BINARY_DIR)/release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip \
		$(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe configs/config.yaml
	
	@tar -czf $(BINARY_DIR)/release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz \
		-C $(BINARY_DIR) $(BINARY_NAME)-darwin-amd64 \
		-C .. configs/config.yaml
	
	@tar -czf $(BINARY_DIR)/release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz \
		-C $(BINARY_DIR) $(BINARY_NAME)-darwin-arm64 \
		-C .. configs/config.yaml
	
	@echo "Release packages created in $(BINARY_DIR)/release/"

# Run the daemon in development mode
run: build
	@echo "Running $(BINARY_NAME) in development mode..."
	@$(BINARY_DIR)/$(BINARY_NAME) -config configs/config.yaml run

# Test connection to LED matrix
test-connection: build
	@echo "Testing connection to LED matrix..."
	@$(BINARY_DIR)/$(BINARY_NAME) -config configs/config.yaml test

# Show configuration
show-config: build
	@echo "Showing current configuration..."
	@$(BINARY_DIR)/$(BINARY_NAME) -config configs/config.yaml config

# Install development dependencies
dev-deps:
	@echo "Installing development dependencies..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Development dependencies installed"

# Run static analysis
staticcheck:
	@echo "Running staticcheck..."
	@staticcheck ./...
	@echo "Static analysis complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  all                - Build, format, vet, test with coverage (default)"
	@echo "  build              - Build the binary"
	@echo "  simulator          - Build and run LED matrix simulator (no hardware needed)"
	@echo "  clean              - Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  test               - Run all tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-short         - Run short tests only (skip integration)"
	@echo "  test-bench         - Run benchmark tests"
	@echo "  test-coverage-check - Run tests with coverage threshold check"
	@echo "  test-ci            - Run tests in CI environment"
	@echo "  test-clean         - Clean test artifacts"
	@echo "  test-config        - Test config package only"
	@echo "  test-matrix        - Test matrix package only"
	@echo "  test-stats         - Test stats package only"
	@echo "  test-visualizer    - Test visualizer package only"
	@echo "  test-daemon        - Test daemon package only"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt                - Format code"
	@echo "  vet                - Run go vet"
	@echo "  staticcheck        - Run static analysis"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps               - Update dependencies"
	@echo "  dev-deps           - Install development dependencies"
	@echo ""
	@echo "Installation:"
	@echo "  install            - Install daemon system-wide"
	@echo "  uninstall          - Uninstall daemon"
	@echo ""
	@echo "Build & Release:"
	@echo "  cross-compile      - Build for multiple platforms"
	@echo "  release            - Create release packages"
	@echo ""
	@echo "Development:"
	@echo "  run                - Run daemon in development mode"
	@echo "  test-connection    - Test LED matrix connection"
	@echo "  show-config        - Show current configuration"
	@echo ""
	@echo "  help               - Show this help message"