# GitHub Copilot Instructions for Framework LED Matrix Daemon

This repository contains a Go daemon that monitors system metrics (CPU, memory, disk, network) and displays them on Framework Laptop LED matrix modules. The system supports both single and dual matrix configurations with various display modes.

## Project Overview

- **Language**: Go 1.26+
- **Type**: System daemon / Background service
- **Platform**: Cross-platform (Linux, Windows)
- **Hardware**: Framework Laptop LED Matrix modules (USB serial protocol)
- **Key Features**: Real-time metrics collection, LED visualization, dual matrix support, systemd integration

## Repository Layout

```
framework-led-matrix-daemon/
├── cmd/
│   ├── daemon/           # Main daemon application entry point
│   └── simulator/        # LED matrix simulator (hardware-independent testing)
├── internal/
│   ├── config/          # YAML configuration parsing and validation
│   ├── daemon/          # Service management and lifecycle
│   ├── matrix/          # LED matrix communication protocol (Framework protocol)
│   ├── stats/           # System metrics collection (gopsutil)
│   ├── visualizer/      # Metrics to LED pattern mapping
│   └── observability/   # Logging, metrics, health monitoring
├── configs/
│   └── config.yaml      # Default configuration with examples
├── systemd/
│   └── *.service        # Systemd service files
├── .github/workflows/   # CI/CD pipelines
│   ├── ci.yml          # Main CI pipeline
│   ├── release.yml     # Release builds
│   ├── claude.yml      # Claude AI integration
│   └── claude-code-review.yml # Claude code reviews
├── Makefile            # Build automation
├── CLAUDE.md           # Claude AI assistant instructions
├── TESTING.md          # Testing strategies and mock usage
└── CONTRIBUTING.md     # Development guidelines
```

## Architecture and Key Components

### Service Architecture
- `internal/daemon/service.go` - Main service orchestrator with layered pattern
- Goroutine-based with context cancellation and sync.WaitGroup coordination
- Structured shutdown process for graceful termination

### Configuration System
- YAML-based config in `internal/config/` with hot-reload via fsnotify
- Supports command-line overrides and environment variable substitution
- Dual matrix modes: mirror, split, extended, independent

### Matrix Communication
- `internal/matrix/` - Serial communication with Framework LED protocol
- Magic bytes: 0x32, 0xAC (Framework protocol)
- Components: `Client`, `MultiClient`, `DisplayManager`, `MultiDisplayManager`
- Auto-discovery of Framework matrices (VID: 32AC)

### System Monitoring
- `internal/stats/collector.go` - Uses gopsutil for metrics
- Collects: CPU, memory, disk I/O, network stats
- Configurable collection intervals and metric selection

### Visualization Pipeline
- `internal/visualizer/` - Converts metrics to LED patterns
- `Visualizer` (single matrix) and `MultiVisualizer` (dual matrix)
- `Mapper` - Maps metrics to visual representations (bars, gradients, activity)

## Build and Development Commands

### Essential Build Commands

```bash
# Install dependencies (ALWAYS run first on fresh clone)
make deps

# Build the daemon
make build

# Build output: bin/framework-led-daemon
```

### Testing Commands

```bash
# Run all tests
make test

# Run tests with coverage (requires >50% coverage to pass)
make test-coverage

# Run tests with race detection (IMPORTANT: always run before submitting)
make test-race

# Run short tests only (skip integration tests)
make test-short

# Package-specific tests
make test-matrix      # LED matrix communication
make test-stats       # System statistics
make test-visualizer  # Display patterns
make test-config      # Configuration management
make test-daemon      # Daemon service

# CI test suite (what runs in GitHub Actions)
make test-ci
```

### Code Quality Commands

```bash
# Format code (ALWAYS run before committing)
make fmt

# Run go vet
make vet

# Full linting suite
make lint

# Quality check (comprehensive validation)
make quality-check
```

### Hardware-Independent Development

```bash
# LED matrix simulator (no hardware needed)
make simulator

# Simulator with specific mode
make simulator ARGS="-mode percentage -metric cpu -duration 60s"
make simulator ARGS="-mode activity -metric memory"
make simulator ARGS="-mode status -duration 60s"
```

### Cross-Platform Building

```bash
# Cross-compile for all platforms (Linux amd64/arm64, Windows amd64)
make cross-compile

# Create release packages
make release
```

## Important Development Patterns

### Error Handling
- Use structured error handling with context propagation
- Wrap errors with context: `fmt.Errorf("failed to connect: %w", err)`
- Matrix communication errors handled gracefully with retry logic
- Falls back to mock operations when hardware unavailable

### Goroutine Management
- Service components run in separate goroutines
- Coordinated by main service loop
- Clean shutdown uses context cancellation and sync.WaitGroup
- See `internal/daemon/service.go` for reference pattern

### Testing Without Hardware
- Comprehensive mocking system for hardware-independent development
- `MockClient` - Simulates LED matrix hardware
- `MockPort` - Simulates serial communication
- Visual simulator provides ASCII representation of LED patterns
- See `TESTING.md` for complete testing strategies

### Dual Matrix Support
- Architecture cleanly separates single and multi-matrix operations
- Auto-switches to `MultiDisplayManager` when multiple matrices detected
- Modes: mirror, split, extended, independent
- Maintain backward compatibility with single matrix mode

### Serial Communication
- Framework LED protocol with proper command encoding
- Error handling and connection management
- Auto-discovery of Framework matrices
- Fallback to generic USB ports

## Configuration

Default configuration location: `configs/config.yaml`

Key configuration sections:
- `led_matrix` - Matrix connection and display settings
- `system_stats` - Metrics collection configuration
- `thresholds` - Warning and critical levels
- `dual_matrix` - Dual matrix configuration (optional)

Configuration supports:
- Hot-reload via file watching
- Command-line overrides
- Environment variable substitution

## CI/CD Pipeline

### GitHub Actions Workflows

- **ci.yml** - Main CI pipeline (builds, tests, linting)
- **release.yml** - Automated releases with cross-platform builds
- **claude.yml** - Claude AI integration for issue handling
- **claude-code-review.yml** - Automated code reviews

### CI Validation

The CI pipeline validates:
1. `make deps` - Dependency installation
2. `make test-ci` - Tests with race detection and coverage (≥50% required)
3. `make lint` - Linting with golangci-lint
4. `make build` - Build verification
5. Cross-platform builds (Linux amd64/arm64, Windows amd64)

## Common Development Tasks

### Adding a New Metric

1. Add to `internal/stats/collector.go` - Collection logic
2. Update `internal/visualizer/mapper.go` - Visualization mapping
3. Add configuration option in `internal/config/config.go`
4. Add tests in respective `*_test.go` files
5. Update example config in `configs/config.yaml`

### Adding a New Display Mode

1. Define mode in `internal/visualizer/visualizer.go`
2. Implement visualization logic
3. Add to config validation in `internal/config/validate.go`
4. Add tests with simulator
5. Document in README.md

### Modifying Matrix Protocol

1. Update `internal/matrix/commands.go` - Command definitions
2. Modify `internal/matrix/client.go` - Protocol implementation
3. Update mocks in `internal/matrix/mock.go`
4. Test with simulator before hardware
5. Validate with `make test-matrix`

## Testing Requirements

- **Unit Tests**: All new functions must have unit tests
- **Coverage**: Maintain ≥50% code coverage (enforced by CI)
- **Race Detection**: Run `make test-race` before submitting
- **Hardware Mocking**: Use existing mock infrastructure for tests
- **Integration Tests**: Use build tag `//go:build integration`

## Code Style

- Follow standard Go conventions (`go fmt`, `go vet`)
- Use `goimports` for import organization
- Table-driven tests for comprehensive coverage
- Clear error messages with context
- Document exported functions and types

## Known Build Issues and Workarounds

1. **Serial Port Permissions** (Linux):
   - Add user to `dialout` group: `sudo usermod -a -G dialout $USER`
   - Requires logout/login to take effect

2. **CGO Disabled**:
   - Build uses `CGO_ENABLED=0` for static linking
   - Ensures cross-platform compatibility

3. **Test Timing**:
   - Integration tests may timeout in CI
   - Use `-short` flag to skip: `make test-short`

4. **Dependency Updates**:
   - Run `make deps` after pulling changes
   - Run `go mod tidy` if dependency issues occur

## Important Files

- **CLAUDE.md** - Detailed architecture and patterns for Claude AI
- **TESTING.md** - Comprehensive testing guide with mock usage
- **CONTRIBUTING.md** - Full development workflow and guidelines
- **Makefile** - All build, test, and quality commands
- **.golangci.yml** - Linting configuration
- **configs/config.yaml** - Configuration examples and documentation

## Validation Checklist

Before submitting changes:

1. Run `make fmt` - Format code
2. Run `make deps` - Update dependencies
3. Run `make test-race` - Check for race conditions
4. Run `make test-coverage` - Verify coverage ≥50%
5. Run `make lint` - Check code quality
6. Run `make build` - Verify builds successfully
7. Test with simulator if UI/display changes
8. Update relevant documentation

## Trust These Instructions

These instructions are maintained and validated. When implementing changes:
- Follow the build commands exactly as documented
- Use the documented test patterns
- Refer to existing code in the specified files for examples
- Only search for additional information if these instructions are incomplete or incorrect
