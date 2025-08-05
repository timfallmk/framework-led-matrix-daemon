# Contributing to Framework LED Matrix Daemon

Thank you for your interest in contributing to the Framework LED Matrix Daemon! This guide will help you get started with development, testing, and submitting contributions.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Dual Matrix Development](#dual-matrix-development)
- [Testing](#testing)
- [Debugging](#debugging)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)
- [Hardware Testing](#hardware-testing)
- [Documentation](#documentation)

## Getting Started

### Prerequisites

- **Go 1.24.5** or later
- **Git** for version control
- **Make** for build automation
- **Framework Laptop** with LED Matrix module (for hardware testing)
- **Linux, macOS, or Windows** (cross-platform support)

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/timfallmk/framework-led-matrix-daemon.git
cd framework-led-matrix-daemon

# Install dependencies and development tools
make deps
make dev-deps

# Run tests to verify setup
make test

# Build the project
make build

# Test without hardware using the simulator
make simulator
```

## Development Environment

### Required Tools

Install these development dependencies:

```bash
# Install Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# Or use the Makefile target
make dev-deps
```

### IDE Configuration

#### VS Code
Recommended extensions:
- Go (official Go extension)
- Go Test Explorer
- Coverage Gutters

#### GoLand/IntelliJ
- Enable Go modules support
- Configure test runner for table-driven tests

### Environment Variables

For development, you can set these environment variables:

```bash
export FRAMEWORK_LED_CONFIG=/path/to/config.yaml
export FRAMEWORK_LED_DEBUG=true
export FRAMEWORK_LED_LOG_LEVEL=debug
```

## Project Structure

```
framework-led-matrix-daemon/
├── cmd/
│   ├── daemon/           # Main daemon application
│   └── simulator/        # LED matrix simulator (no hardware needed)
├── internal/
│   ├── config/          # Configuration management
│   ├── daemon/          # Daemon service implementation
│   ├── matrix/          # LED matrix communication protocol
│   ├── stats/           # System statistics collection
│   ├── visualizer/      # Data visualization and display mapping
│   └── testutils/       # Shared testing utilities
├── configs/
│   └── config.yaml      # Default configuration
├── systemd/
│   └── *.service        # Systemd service files
├── docs/                # Documentation (if present)
├── Makefile            # Build automation
├── README.md           # Project overview
├── TESTING.md          # Testing guide
└── CONTRIBUTING.md     # This file
```

### Package Responsibilities

- **cmd/daemon**: Main entry point, CLI parsing, daemon lifecycle
- **cmd/simulator**: Hardware-independent testing and development tool
- **internal/config**: YAML configuration parsing and validation
- **internal/daemon**: Service management, signal handling, main loop
- **internal/matrix**: Low-level LED matrix protocol and communication
- **internal/stats**: System metrics collection (CPU, memory, disk, network)
- **internal/visualizer**: Maps system stats to LED display patterns
- **internal/testutils**: Shared mocks and test helpers

## Development Workflow

### 1. Issue and Feature Planning

- Check existing issues before starting work
- Create an issue for bugs or feature requests
- Discuss major changes in issues before implementing
- Reference issues in commit messages: `fixes #123`

### 2. Branch Strategy

```bash
# Create feature branch from main
git checkout main
git pull origin main
git checkout -b feature/my-new-feature

# Or for bug fixes
git checkout -b fix/issue-description
```

### 3. Development Cycle

```bash
# Make your changes
# ...

# Format and lint code
make fmt
make vet
make staticcheck

# Run tests
make test
make test-race

# Test with simulator (no hardware needed)
make simulator

# Build and test
make build
```

### 4. Commit Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Feature
git commit -m "feat: add new display mode for network activity"

# Bug fix
git commit -m "fix: resolve memory leak in stats collector"

# Documentation
git commit -m "docs: update configuration examples"

# Tests
git commit -m "test: add integration tests for matrix package"

# Refactor
git commit -m "refactor: simplify display manager interface"
```

## Code Standards

### Go Code Style

- Follow standard Go conventions (`go fmt`, `go vet`)
- Use `goimports` for import organization
- Write idiomatic Go code
- Use meaningful variable and function names
- Keep functions focused and small

### Code Organization

```go
// Good: Clear interface definition
type DisplayManager interface {
    UpdatePercentage(key string, percent float64) error
    ShowActivity(active bool) error
    SetBrightness(level byte) error
}

// Good: Table-driven test structure
func TestDisplayManager_UpdatePercentage(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        percent float64
        wantErr bool
    }{
        {"valid cpu percentage", "cpu", 75.5, false},
        {"invalid percentage", "cpu", -10, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Error Handling

```go
// Good: Wrap errors with context
if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to LED matrix: %w", err)
}

// Good: Use custom error types for specific conditions
var ErrDeviceNotFound = errors.New("LED matrix device not found")
```

### Documentation

- Use Go doc comments for exported functions and types
- Include examples in doc comments when helpful
- Keep comments up-to-date with code changes

```go
// UpdatePercentage displays a percentage value on the LED matrix.
// The key parameter identifies the metric type (e.g., "cpu", "memory").
// The percent value should be between 0.0 and 100.0.
//
// Example:
//   err := display.UpdatePercentage("cpu", 75.5)
func UpdatePercentage(key string, percent float64) error {
    // implementation
}
```

## Dual Matrix Development

### Architecture Overview

The dual matrix support is built with backward compatibility in mind:

- **Single Matrix**: Uses `matrix.Client` and `matrix.DisplayManager` (legacy path)
- **Dual Matrix**: Uses `matrix.MultiClient` and `matrix.MultiDisplayManager` (new path)
- **Service Layer**: Automatically detects and initializes the appropriate mode

### Key Components

1. **MultiClient** (`internal/matrix/client.go`):
   - Manages multiple LED matrix connections
   - Handles port discovery for multiple devices
   - Provides individual client access by name

2. **MultiDisplayManager** (`internal/matrix/display.go`):
   - Coordinates display updates across multiple matrices
   - Implements different dual modes: mirror, split, extended, independent

3. **Configuration** (`internal/config/config.go`):
   - Supports both legacy single matrix and new dual matrix config
   - Validates dual matrix configurations
   - Converts between config formats to avoid import cycles

### Dual Matrix Modes

- **Mirror Mode**: Both matrices show identical content
- **Split Mode**: Each matrix shows different metrics (e.g., CPU+Memory vs Disk+Network)
- **Extended Mode**: Wide visualization spanning both matrices
- **Independent Mode**: Completely separate configurations per matrix

### Development Guidelines

When working on dual matrix features:

1. **Maintain Compatibility**: Always ensure single matrix mode continues to work
2. **Graceful Fallback**: If dual matrix setup fails, fall back to single matrix
3. **Configuration Validation**: Validate dual matrix configs in `config.Validate()`
4. **Testing**: Use the simulator to test dual matrix visualizations
5. **Error Handling**: Log warnings for matrix-specific failures, don't crash the daemon

### Testing Dual Matrix Features

```bash
# Test with dual matrix configuration
./bin/framework-led-daemon -config configs/dual-matrix-example.yaml run

# Use simulator to visualize dual matrix layouts
make simulator ARGS='-mode percentage -metric cpu'

# Test port discovery for multiple matrices
./bin/framework-led-daemon test
```

### Adding New Dual Matrix Modes

To add a new dual matrix mode:

1. Add the mode to `validDualModes` in `config.Validate()`
2. Implement the mode in `MultiDisplayManager.UpdateMetric()`
3. Add configuration example to README.md
4. Add tests for the new mode

Example:
```go
case "my_new_mode":
    return mdm.updateMyNewMode(metricName, value, stats)
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run specific package tests
make test-matrix
make test-stats
make test-config

# Run integration tests only
go test -run Integration ./...

# Run short tests (skip integration)
make test-short
```

### Writing Tests

#### Unit Tests

```go
func TestCollector_GetSummary(t *testing.T) {
    collector := stats.NewCollector(time.Second)
    
    summary, err := collector.GetSummary()
    require.NoError(t, err)
    assert.NotNil(t, summary)
    assert.GreaterOrEqual(t, summary.CPUUsage, 0.0)
    assert.LessOrEqual(t, summary.CPUUsage, 100.0)
}
```

#### Integration Tests

```go
//go:build integration
// +build integration

func TestDisplayManager_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Test with real hardware or simulation
}
```

#### Mock Usage

```go
func TestVisualizer_UpdateDisplay(t *testing.T) {
    mockDisplay := &matrix.MockDisplayManager{}
    cfg := config.DefaultConfig()
    viz := visualizer.NewVisualizer(mockDisplay, cfg)
    
    summary := &stats.StatsSummary{
        CPUUsage: 75.0,
        Status:   stats.StatusNormal,
    }
    
    err := viz.UpdateDisplay(summary)
    assert.NoError(t, err)
    
    // Verify mock was called correctly
    mockDisplay.AssertCalled(t, "UpdatePercentage", "cpu", 75.0)
}
```

### Test Coverage

Maintain test coverage above 70%:

```bash
# Check coverage with threshold
make test-coverage-check

# Generate detailed coverage report
make test-coverage
open coverage.html
```

### Testing Without Hardware

Use the built-in simulator for development:

```bash
# Run simulator with different modes
make simulator ARGS="-mode percentage -metric cpu -duration 30s"
make simulator ARGS="-mode activity -metric memory"
make simulator ARGS="-mode status -duration 60s"

# Custom configuration
make simulator ARGS="-config custom-config.yaml"
```

See [TESTING.md](TESTING.md) for comprehensive testing strategies.

## Debugging

### Debug Builds

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o bin/framework-led-daemon-debug ./cmd/daemon

# Run with verbose logging
./bin/framework-led-daemon-debug -config configs/config.yaml -debug run
```

### Common Issues

#### Serial Port Problems
```bash
# Check permissions
ls -la /dev/ttyACM*
sudo usermod -a -G dialout $USER  # May require logout

# Test connection
make test-connection
```

#### Performance Issues
```bash
# Run with profiling
go run -cpuprofile=cpu.prof ./cmd/daemon -config configs/config.yaml run

# Analyze profile
go tool pprof cpu.prof
```

#### Memory Leaks
```bash
# Run with race detection
make test-race

# Memory profiling
go run -memprofile=mem.prof ./cmd/daemon -config configs/config.yaml run
```

## Submitting Changes

### Pull Request Process

1. **Ensure CI passes locally**:
   ```bash
   make test-ci
   make staticcheck
   make fmt
   ```

2. **Update documentation** if needed:
   - Update README.md for user-facing changes
   - Update code comments for API changes
   - Add or update tests

3. **Create Pull Request**:
   - Use a clear, descriptive title
   - Reference related issues
   - Include testing notes
   - Add screenshots for UI changes (simulator output)

### PR Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] New tests added for new functionality
- [ ] Tested with hardware (if applicable)
- [ ] Tested with simulator

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes (or breaking changes documented)
```

### Code Review Guidelines

#### For Authors
- Keep PRs focused and reasonably sized
- Write clear commit messages
- Respond to feedback promptly
- Update tests and documentation

#### For Reviewers
- Be constructive and specific
- Focus on code quality, not personal preferences
- Test changes when possible
- Approve when ready, request changes when needed

## Release Process

### Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Creating Releases

1. **Update version**:
   ```bash
   git tag v1.2.3
   ```

2. **Build release packages**:
   ```bash
   make release
   ```

3. **Test release builds**:
   ```bash
   # Test each platform binary
   ./bin/framework-led-daemon-linux-amd64 --version
   ```

4. **Create GitHub release** with:
   - Release notes
   - Binary attachments
   - Breaking change notices

### Release Checklist

- [ ] All tests pass
- [ ] Documentation updated
- [ ] Version bumped appropriately
- [ ] Changelog updated
- [ ] Cross-platform builds successful
- [ ] Release notes written
- [ ] Hardware tested (if possible)

## Hardware Testing

### Framework Laptop Setup

1. **Verify LED Matrix Module**:
   - Check physical connection
   - Verify in BIOS/UEFI settings
   - Check device enumeration: `lsusb` or `dmesg`

2. **Permission Setup**:
   ```bash
   sudo usermod -a -G dialout $USER
   # Log out and back in
   ```

3. **Test Connection**:
   ```bash
   make test-connection
   ```

### Hardware-Specific Tests

```bash
# Test all display modes
make build
./bin/framework-led-daemon -config configs/config.yaml test-patterns

# Test brightness levels
./bin/framework-led-daemon -config configs/config.yaml test-brightness

# Test animation performance
./bin/framework-led-daemon -config configs/config.yaml test-animation
```

## Documentation

### Documentation Standards

- Use clear, concise language
- Include code examples
- Keep documentation up-to-date with code
- Use proper Markdown formatting

### Required Documentation Updates

When making changes, update:

- **README.md**: User-facing features and installation
- **TESTING.md**: New testing procedures
- **Code comments**: API changes
- **Configuration docs**: New config options
- **Examples**: Usage examples and tutorials

### Writing Guidelines

- Use present tense: "This function returns..." not "This function will return..."
- Be specific: Include exact commands and expected outputs
- Include error scenarios and troubleshooting
- Use code blocks for terminal commands and code snippets

## Getting Help

### Resources

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **README.md**: Usage and installation guide
- **TESTING.md**: Testing strategies and tools

### Communication

- Search existing issues before creating new ones
- Use issue templates when available
- Provide minimal reproducible examples
- Include system information for bug reports

---

Thank you for contributing to the Framework LED Matrix Daemon! Your contributions help make this project better for everyone in the Framework community.