# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This is a Go daemon that monitors system metrics (CPU, memory, disk, network) and displays them on Framework Laptop LED matrix modules. The system supports both single and dual matrix configurations with various display modes.

### Core Components

**Service Architecture**: The daemon uses a layered service pattern centered around `internal/daemon/service.go`. The `Service` struct orchestrates all major components and manages their lifecycle through a structured shutdown process with context cancellation and sync.WaitGroup coordination.

**Configuration System**: YAML-based configuration in `internal/config/` with hot-reload capabilities via fsnotify. Configuration supports command-line overrides and environment variable substitution. The config package handles both single matrix and dual matrix setups with different modes (mirror, split, extended, independent).

**Matrix Communication**: The `internal/matrix/` package manages serial communication with LED matrices using the `go.bug.st/serial` library. Key components:
- `Client`: Individual matrix communication with Framework protocol (magic bytes 0x32 0xAC)
- `MultiClient`: Manages multiple matrix connections
- `DisplayManager`/`MultiDisplayManager`: Higher-level display control
- Commands are defined in `commands.go` with proper byte encoding

**System Monitoring**: `internal/stats/collector.go` uses `gopsutil` to collect system metrics. The collector runs in its own goroutine with configurable intervals and supports enabling/disabling specific metrics (CPU, memory, disk I/O, network).

**Visualization Pipeline**: `internal/visualizer/` converts metrics to LED patterns:
- `Visualizer`: Single matrix pattern generation
- `MultiVisualizer`: Dual matrix coordination
- `Mapper`: Maps metrics to visual representations (percentage bars, gradients, activity indicators)

**Observability**: `internal/observability/` provides structured logging, metrics collection, health monitoring, and application metrics. Uses platform-specific implementations for disk space monitoring (Unix vs Windows).

### Data Flow

1. **Metrics Collection**: `stats.Collector` → system metrics every `collect_interval`
2. **Visualization**: `visualizer.Visualizer` → metrics to LED patterns
3. **Display**: `matrix.DisplayManager` → patterns to matrix via serial protocol
4. **Coordination**: `daemon.Service` orchestrates the pipeline with proper lifecycle management

## Development Commands

### Essential Commands
```bash
# Build the daemon
make build

# Run tests with coverage
make test-coverage

# Visual LED simulator (no hardware needed)
make simulator
make simulator ARGS="-mode percentage -metric cpu -duration 60s"

# Code quality checks
make lint          # Full linting suite
make quality-check # Comprehensive quality checks

# Development mode
make run           # Run daemon with default config
```

### Testing Without Hardware
The project includes comprehensive hardware mocking via `TESTING.md`. Key test commands:
```bash
# Package-specific tests
make test-matrix      # LED matrix communication
make test-stats       # System statistics
make test-visualizer  # Display patterns
make test-config      # Configuration management

# Integration testing
go test -v ./internal/daemon -run TestServiceLifecycle

# Performance testing
make test-race        # Race condition detection
make test-bench       # Benchmark tests
```

### Cross-Platform Building
```bash
# Cross-compile (Linux amd64/arm64, Windows amd64)
make cross-compile

# Create release packages
make release
```

### Hardware Testing
```bash
# Test actual hardware connection
make test-connection

# Show current configuration
make show-config
```

## Key Development Patterns

**Error Handling**: The codebase uses structured error handling with context propagation. Matrix communication errors are handled gracefully with retry logic and fallback to mock operations when hardware is unavailable.

**Goroutine Management**: Service components run in separate goroutines coordinated by the main service loop. Proper shutdown uses context cancellation and sync.WaitGroup to ensure clean termination.

**Configuration Hot-Reload**: The config system supports live configuration updates via file watching, allowing runtime changes without daemon restart.

**Dual Matrix Support**: The architecture cleanly separates single and multi-matrix operations. When multiple matrices are detected, the system automatically switches to `MultiDisplayManager` and `MultiVisualizer` coordination.

**Serial Communication**: Matrix communication follows the Framework LED protocol with proper command encoding, error handling, and connection management. The system automatically discovers Framework matrices (VID: 32AC) but falls back to generic USB ports.

## Testing Strategy

The project emphasizes hardware-independent development through comprehensive mocking:
- **MockClient**: Simulates LED matrix hardware
- **MockPort**: Simulates serial communication
- **Visual Simulator**: ASCII representation of LED patterns
- **Hardware Protocol Accuracy**: Exact Framework LED Matrix command format simulation

This allows full development and testing without requiring physical hardware access.

## Build Tags and Cross-Compilation

The project previously supported a `nocgo` build tag for cross-compilation compatibility with the serial library, but current implementation focuses on Linux and Windows platforms with CGO enabled for full hardware support.