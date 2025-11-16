# CLAUDE.md Enhancement Suggestions

This document contains recommendations for improving CLAUDE.md based on a comprehensive repository scan. The suggestions are organized by priority and category.

## High Priority Additions

### 1. Complete CLI Command Reference

**Current State:** CLAUDE.md mentions development commands but doesn't document the full daemon CLI.

**Suggestion:** Add a section documenting all daemon commands:

```markdown
## Daemon CLI Commands

The daemon binary (`cmd/daemon/main.go`) supports multiple commands for service management:

### Commands
- `run` - Run daemon in foreground (default)
- `install` - Install as system service
- `remove`/`uninstall` - Remove system service
- `start` - Start the service
- `stop` - Stop the service
- `status` - Check service status
- `config` - Display current configuration
- `test` - Test LED matrix connection

### Command-line Flags (Override Config File)
- `-config <path>` - Configuration file path
- `-port <device>` - Serial port (e.g., `/dev/ttyACM0`)
- `-brightness <0-255>` - LED brightness level
- `-mode <mode>` - Display mode (percentage|gradient|activity|status)
- `-metric <metric>` - Primary metric (cpu|memory|disk|network)
- `-log-level <level>` - Logging level (debug|info|warn|error)
- `-version` - Show version information
- `-help` - Show help message

### Configuration Discovery Order
1. CLI flag (`-config`)
2. `$XDG_CONFIG_HOME/framework-led-daemon/config.yaml`
3. `$HOME/.config/framework-led-daemon/config.yaml`
4. `/etc/framework-led-daemon/config.yaml`
5. `./configs/config.yaml`

### Service Management Examples
```bash
# Install and start service
sudo ./framework-led-daemon install
sudo ./framework-led-daemon start

# Check service status
./framework-led-daemon status

# Test connection before installing
./framework-led-daemon test -port /dev/ttyACM0
```
```

**Rationale:** This information is critical for users installing and managing the daemon but isn't currently in CLAUDE.md.

---

### 2. Complete Configuration Schema

**Current State:** Mentions YAML-based configuration and hot-reload, but doesn't document all available options.

**Suggestion:** Add a comprehensive configuration reference:

```markdown
## Configuration Schema

### Complete Configuration Options

```yaml
# Matrix Hardware Configuration
matrix:
  port: "/dev/ttyACM0"           # Serial port (auto-discovered if empty)
  baud_rate: 115200              # Serial baud rate
  auto_discover: true            # Auto-discover Framework matrices (VID: 32AC)
  timeout: "5s"                  # Connection timeout
  brightness: 128                # LED brightness (0-255)

  # Dual Matrix Configuration
  dual_mode: "split"             # mirror|split|extended|independent
  matrices:
    - port: "/dev/ttyACM0"
      brightness: 128
      metric: "cpu"              # For independent mode
    - port: "/dev/ttyACM1"
      brightness: 128
      metric: "memory"

# Statistics Collection
stats:
  collect_interval: "1s"         # How often to collect metrics
  enable_cpu: true               # Enable CPU monitoring
  enable_memory: true            # Enable memory monitoring
  enable_disk: true              # Enable disk I/O monitoring
  enable_network: true           # Enable network monitoring

  # Alert Thresholds
  thresholds:
    cpu_warning: 70.0            # CPU warning threshold (%)
    cpu_critical: 90.0           # CPU critical threshold (%)
    memory_warning: 80.0         # Memory warning threshold (%)
    memory_critical: 95.0        # Memory critical threshold (%)
    disk_warning: 80.0           # Disk usage warning (%)
    disk_critical: 95.0          # Disk usage critical (%)

# Display Configuration
display:
  update_rate: "100ms"           # How often to update LEDs
  mode: "percentage"             # percentage|gradient|activity|status
  primary_metric: "cpu"          # cpu|memory|disk|network
  show_activity: true            # Show activity indicators
  enable_animation: true         # Enable animations
  custom_patterns: []            # Custom LED patterns (advanced)

# Daemon Service Configuration
daemon:
  name: "framework-led-daemon"   # Service name
  description: "Framework LED Matrix Display Daemon"
  user: ""                       # Run as user (empty = current)
  group: ""                      # Run as group (empty = current)
  pid_file: "/var/run/framework-led-daemon.pid"
  log_file: "/var/log/framework-led-daemon.log"

# Logging Configuration
logging:
  level: "info"                  # debug|info|warn|error
  format: "text"                 # text|json
  output: "stdout"               # stdout|stderr|file
  add_source: false              # Include source file/line
  event_buffer_size: 1000        # Event buffer size
  # File rotation (when output=file)
  max_size: 100                  # MB
  max_backups: 3
  max_age: 28                    # Days
  compress: true
```

### Dual Matrix Modes Explained

- **mirror** - Both matrices show the same content (synchronized)
- **split** - Different metrics on each matrix (e.g., CPU left, memory right)
- **extended** - Single wide visualization across both matrices
- **independent** - Completely separate configurations per matrix

### Hot-Reload

Configuration changes are detected automatically via fsnotify and applied without restarting the daemon. Changes to the following are applied live:
- Brightness levels
- Display modes and metrics
- Update rates
- Thresholds
- Logging configuration

**Note:** Changes to serial ports or dual_mode require a restart.
```

**Rationale:** Developers need to understand all configuration options when debugging or extending functionality.

---

### 3. Observability & Debugging

**Current State:** Not documented in CLAUDE.md.

**Suggestion:** Add observability section:

```markdown
## Observability & Debugging

### Logging System

The daemon uses structured logging via `internal/logging/`:

- **Logger**: Core slog-based structured logger
- **EventLogger**: Domain-specific logging (daemon, matrix, stats, config, display)
- **MetricsLogger**: Performance metrics logging

**Log Levels:**
- `debug` - Detailed debugging information (serial communication, state changes)
- `info` - Normal operational messages (startup, config changes, mode switches)
- `warn` - Warning conditions (connection issues, threshold warnings)
- `error` - Error conditions (failed commands, initialization errors)

**Enable Debug Logging:**
```bash
# Via flag
./framework-led-daemon run -log-level debug

# Via config
logging:
  level: "debug"
  format: "json"        # JSON for structured parsing
  add_source: true      # Include file:line information
```

### Health Monitoring

`internal/observability/` provides component health checking:

**Health Checkers:**
- `MatrixHealthChecker` - LED matrix connection status
- `StatsHealthChecker` - System metrics collection status
- `ConfigHealthChecker` - Configuration validity
- `MemoryHealthChecker` - Application memory usage
- `DiskSpaceHealthChecker` - Available disk space (platform-specific)

**Check Health:**
```bash
# Service status includes health checks
./framework-led-daemon status
```

### Metrics Collection

`MetricsCollector` tracks application performance:
- Counter - Incrementing values (errors, commands sent)
- Gauge - Point-in-time values (brightness level, update rate)
- Histogram - Value distributions (command latency, update times)

### Debugging Techniques

**Visual Simulator for Pattern Testing:**
```bash
# Test patterns without hardware
make simulator ARGS="-mode gradient -metric cpu -duration 30s"
```

**Serial Communication Debugging:**
```bash
# Enable debug logging to see protocol-level communication
./framework-led-daemon run -log-level debug 2>&1 | grep "matrix"
```

**Configuration Validation:**
```bash
# Test configuration without running daemon
./framework-led-daemon config -config /path/to/config.yaml
```

**Connection Testing:**
```bash
# Test matrix connection
make test-connection
# Or
./framework-led-daemon test -port /dev/ttyACM0
```
```

**Rationale:** Understanding observability is crucial for debugging issues and monitoring daemon health.

---

## Medium Priority Additions

### 4. Testing Infrastructure Details

**Current State:** Mentions MockClient and MockPort but not the full testing infrastructure.

**Suggestion:** Add testing infrastructure section:

```markdown
## Testing Infrastructure

### Test Utilities (`internal/testutils/`)

The 437-line testutils package provides comprehensive testing helpers:

**Configuration Helpers:**
- `NewTestConfig()` - Create test configuration with defaults
- `NewTestMatrixConfig()` - Matrix configuration for tests
- `NewTestStatsConfig()` - Stats configuration for tests

**Stats Generators:**
- `NewTestStats()` - Generate realistic system stats
- `NewTestStatsSummary()` - Generate stats summary with thresholds
- `ValidateSystemStats()` - Validate stats structure

**Assertion Helpers:**
- `AssertFloatInRange()` - Validate float values
- `AssertDurationInRange()` - Validate durations
- `AssertTimeRecent()` - Validate timestamps
- `AssertPercentageValid()` - Validate 0-100 percentages
- `AssertBytesNonNegative()` - Validate byte counts

**Concurrent Testing:**
- `RunConcurrent()` - Execute tests concurrently
- `WaitForCondition()` - Wait for condition with timeout

### Mock Components

**Complete Hardware Abstraction:**

```go
// MockPort (internal/matrix/mock_port.go)
type MockPort struct {
    WriteFunc    func([]byte) (int, error)  // Inject responses
    ReadFunc     func([]byte) (int, error)  // Inject responses
    CloseFunc    func() error
    SetTimeoutFunc func(time.Duration) error
}

// MockClient (internal/matrix/client.go)
type MockClient struct {
    Commands []Command  // Track all commands sent
    Err      error      // Inject errors
}

// MockDisplayManager (internal/matrix/display_manager.go)
type MockDisplayManager struct {
    LastPercentage float64
    LastBrightness uint8
    UpdateCount    int
}
```

**Error Injection Examples:**

```go
// Test connection failure
mockPort := &matrix.MockPort{
    WriteFunc: func(b []byte) (int, error) {
        return 0, errors.New("connection lost")
    },
}

// Test partial writes
mockClient := &matrix.MockClient{
    Err: matrix.ErrPartialWrite,
}
```

### Testing Without Hardware

**Package-Specific Test Targets:**
```bash
make test-config      # Configuration loading and validation
make test-matrix      # Serial communication and commands
make test-stats       # System metrics collection
make test-visualizer  # Pattern generation
make test-daemon      # Service lifecycle
```

**Test Coverage by Package:**
- `visualizer/` - 97.5% (highest coverage)
- `stats/` - 87.3%
- `config/` - 85.5%
- `matrix/` - 79.9%
- `daemon/` - 23.1% (limited by OS-specific features)

**Minimum Coverage Threshold:** 50% enforced in CI

### Integration Testing

```bash
# Full service lifecycle test
go test -v ./internal/daemon -run TestServiceLifecycle

# Race condition detection
make test-race

# Benchmark performance
make test-bench
```
```

**Rationale:** Developers need to understand the testing infrastructure to write effective tests.

---

### 5. Complete Build & Quality Tooling

**Current State:** Lists some make commands but not the full suite of 50+ targets.

**Suggestion:** Reorganize and expand development commands:

```markdown
## Development Commands

### Build Commands

```bash
# Build binaries
make build                # Build daemon binary
make simulator            # Build LED simulator
make cross-compile        # Build for Linux (amd64/arm64) + Windows (amd64)
make release              # Create release packages (.tar.gz, .zip)
make clean                # Remove build artifacts
```

### Testing Commands

```bash
# Run tests
make test                 # All tests
make test-short           # Skip integration tests (faster)
make test-coverage        # Generate coverage report (50% minimum)
make test-race            # Race condition detection
make test-bench           # Benchmark tests
make test-ci              # CI environment (coverage + race detection)

# Package-specific tests
make test-config          # Configuration package
make test-matrix          # Matrix communication
make test-stats           # System statistics
make test-visualizer      # Display patterns
make test-daemon          # Service lifecycle
```

### Code Quality

```bash
# Formatting
make fmt                  # Format code (gofumpt + goimports)

# Linting
make lint                 # Run all linters (40+ rules)
make lint-fix             # Auto-fix linting issues
make vet                  # Go vet analysis
make staticcheck          # Static analysis

# Comprehensive checks
make quality-check        # All quality checks (fmt + lint + vet + staticcheck)
```

### Security & Dependencies

```bash
# Security scanning
make vuln-check           # Vulnerability scanning (govulncheck)
make sbom                 # Generate Software Bill of Materials (syft)

# Dependencies
make deps                 # Download and verify dependencies
make dev-deps             # Install development tools (golangci-lint, etc.)
```

### Installation & Service Management

```bash
# Service operations
make install              # Install as system service
make uninstall            # Remove system service
make run                  # Run in development mode

# Hardware testing
make test-connection      # Test LED matrix connection
make show-config          # Display current configuration
```

### Development Tools

**Required Tools (installed via `make dev-deps`):**
- `golangci-lint` - Comprehensive linting
- `gofumpt` - Stricter gofmt
- `goimports` - Import organization
- `staticcheck` - Static analysis
- `govulncheck` - Vulnerability scanning
- `syft` - SBOM generation

**Linting Configuration (`.golangci.yml`):**
- 40+ enabled linters
- Cyclomatic complexity: ≤60
- Function length: ≤300 lines / 100 statements
- Line length: ≤120 characters
- Security scanning with gosec
```

**Rationale:** Comprehensive tooling documentation helps maintain code quality standards.

---

### 6. Deployment & Production

**Current State:** Not documented.

**Suggestion:** Add deployment section:

```markdown
## Deployment & Production

### Docker Support

**Multi-stage Build:**
```dockerfile
# Build stage: Alpine with Go toolchain
# Final stage: Scratch (minimal attack surface)
```

**Build and Run:**
```bash
# Build image
docker build -t framework-led-daemon .

# Run container (requires device access)
docker run --device=/dev/ttyACM0 \
           -v /path/to/config.yaml:/etc/framework-led-daemon/config.yaml \
           framework-led-daemon
```

**Security Features:**
- Non-root user (UID 65534)
- Minimal base image (scratch)
- CA certificates included
- Health check integrated

### Systemd Service

**Installation:**
```bash
sudo ./framework-led-daemon install
sudo systemctl enable framework-led-daemon
sudo systemctl start framework-led-daemon
```

**Service File Location:** `systemd/framework-led-daemon.service`

**Security Hardening:**
- `NoNewPrivileges=true` - Prevent privilege escalation
- `PrivateTmp=true` - Isolated /tmp
- `ProtectSystem=strict` - Read-only system directories
- `ProtectHome=true` - No home directory access
- `ProtectKernelTunables=true` - Kernel protection
- `RestrictRealtime=true` - No realtime scheduling
- `MemoryDenyWriteExecute=true` - W^X enforcement
- `SystemCallFilter` - Restricted syscalls
- Device access limited to `/dev/ttyACM*` and `/dev/ttyUSB*`

**Resource Limits:**
- File descriptors: 1024
- Processes: 1024
- Auto-restart with 5s delay

**Logs:**
```bash
# View systemd journal
journalctl -u framework-led-daemon -f

# Check service status
systemctl status framework-led-daemon
```

### Cross-Platform Builds

**Build for Multiple Platforms:**
```bash
make cross-compile
```

**Output:**
- `framework-led-daemon-linux-amd64`
- `framework-led-daemon-linux-arm64`
- `framework-led-daemon-windows-amd64.exe`

**Release Packages:**
```bash
make release
```

Creates `.tar.gz` (Linux) and `.zip` (Windows) with:
- Binary
- Sample config
- README
- LICENSE

### CI/CD Pipeline

**GitHub Actions Workflows:**

**ci.yml** (on push/PR):
1. **Test Job** - Linting + tests + coverage check (50% minimum)
2. **Security Job** - CodeQL analysis + vulnerability scanning
3. **Build Job** - Cross-platform builds
4. Artifact retention: 30 days

**release.yml** (on git tags):
1. Pre-release validation
2. Platform-specific package creation
3. Automatic GitHub release creation

**Dependabot:**
- Weekly Go module updates
- Weekly GitHub Actions updates
```

**Rationale:** Production deployment information is essential for real-world usage.

---

## Lower Priority Enhancements

### 7. Protocol Implementation Details

**Suggestion:** Add a detailed protocol reference:

```markdown
## LED Matrix Protocol Details

### Framework LED Matrix Protocol

**Serial Configuration:**
- Baud rate: 115200
- Data bits: 8
- Parity: None
- Stop bits: 1

**Command Structure:**
```
[Magic Byte 1: 0x32] [Magic Byte 2: 0xAC] [Command ID] [Payload...]
```

**Command IDs:**
- `0x00` - Set Brightness
- `0x01` - Set Pattern
- `0x02` - Set Animation
- `0x03` - Draw Bitmap (Black & White)
- `0x04` - Stage Column (Color)
- `0x05` - Flush Columns
- `0x06` - Get Version

**Pattern Types:**
- `0x00` - Percentage (progress bar)
- `0x01` - Gradient
- `0x02` - ZigZag
- `0x03` - Full Bright

**Implementation:** See `internal/matrix/commands.go`

### Port Discovery

**Automatic Discovery:**
1. Search for devices with VID: 32AC (Framework)
2. Filter for LED matrix devices
3. Fallback to first available USB serial port

**Manual Override:**
```bash
./framework-led-daemon run -port /dev/ttyACM0
```
```

---

### 8. Performance Characteristics

**Suggestion:** Add performance notes:

```markdown
## Performance Characteristics

### Update Rates

**Default Configuration:**
- Stats collection: 1s interval
- LED update: 100ms interval (10 Hz)
- Config hot-reload: Event-driven (fsnotify)

**Rate Limiting:**
- Display updates are rate-limited to prevent LED flickering
- Minimum update interval: 50ms (20 Hz maximum)
- Serial write throttling to prevent buffer overflow

### Resource Usage

**Typical Memory:**
- Base daemon: ~10-15 MB RSS
- With stats collection: ~20-25 MB RSS
- Dual matrix: ~25-30 MB RSS

**CPU Usage:**
- Idle: <1% (single core)
- Active display updates: 1-3% (single core)

**Goroutines:**
- Stats collector: 1 goroutine
- Display manager: 1 goroutine per matrix
- Config watcher: 1 goroutine
- Health monitor: 1 goroutine
- Total: ~4-6 goroutines typical

### Benchmarks

```bash
make test-bench
```

Key benchmarks in the codebase:
- Pattern generation performance
- Metric collection overhead
- Display update latency
```

---

### 9. Known Limitations & Troubleshooting

**Suggestion:** Add troubleshooting section:

```markdown
## Known Limitations

### Hardware
- Maximum 2 LED matrices supported simultaneously
- Requires USB serial access (may need udev rules)
- Some USB hubs may introduce latency

### Software
- Hot-reload doesn't apply to port/dual_mode changes (requires restart)
- Windows support is community-tested (primarily developed on Linux)
- SystemCallFilter in systemd may need adjustment for some kernels

### Performance
- Update rates >20 Hz may cause LED flickering
- Very short collection intervals (<100ms) increase CPU usage
- Dual matrix mode has slightly higher latency than single

## Troubleshooting for Developers

### Common Issues

**"Permission denied" on serial port:**
```bash
# Add user to dialout group
sudo usermod -a -G dialout $USER
# Or create udev rule (preferred)
```

**"No matrix found" with auto-discovery:**
```bash
# Check available ports
ls -l /dev/ttyACM* /dev/ttyUSB*

# Test specific port
./framework-led-daemon test -port /dev/ttyACM0

# Disable auto-discovery
matrix:
  auto_discover: false
  port: "/dev/ttyACM0"
```

**Configuration not reloading:**
- Check file permissions on config file
- Verify fsnotify is working: `make test-config`
- Port/dual_mode changes require restart

**High CPU usage:**
- Increase `update_rate` (e.g., 200ms instead of 100ms)
- Increase `collect_interval` (e.g., 2s instead of 1s)
- Disable unused metrics in stats config

### Debug Checklist

1. Enable debug logging: `-log-level debug`
2. Test matrix connection: `make test-connection`
3. Validate configuration: `./framework-led-daemon config`
4. Check systemd logs: `journalctl -u framework-led-daemon -f`
5. Run in foreground: `./framework-led-daemon run`
6. Use visual simulator: `make simulator`
```

---

## Structural Recommendations

### 10. Reorganize CLAUDE.md Structure

**Current Structure:**
- Architecture Overview
- Development Commands
- Testing Strategy

**Suggested Structure:**
```markdown
# CLAUDE.md

## Quick Reference
- Entry points and how to run
- Common commands cheat sheet
- Where to find things

## Architecture Overview
- High-level design (current content)
- Component relationships (current content)
- Data flow (current content)

## Core Components (Expanded)
- Service Architecture (current content + lifecycle details)
- Configuration System (current + complete schema)
- Matrix Communication (current + protocol details)
- System Monitoring (current content)
- Visualization Pipeline (current content)
- **Observability** (NEW - logging, health, metrics)

## Development Guide
- Setting up development environment
- Development Commands (reorganized and expanded)
- Code Quality Tools (NEW)
- Testing Infrastructure (expanded from current)
- Debugging Techniques (NEW)

## Deployment & Production (NEW)
- Docker support
- Systemd service
- Cross-platform builds
- CI/CD pipeline

## Performance & Troubleshooting (NEW)
- Performance characteristics
- Known limitations
- Common issues and solutions

## Reference
- Complete CLI reference (NEW)
- Complete configuration schema (NEW)
- Protocol specification (NEW)
```

---

## Summary of Key Gaps

### Critical Missing Information
1. **CLI commands and flags** - Essential for using the daemon
2. **Complete configuration schema** - Needed for customization
3. **Observability system** - Critical for debugging
4. **Deployment options** - Docker and systemd details

### Important Missing Information
5. **Testing infrastructure** - testutils package
6. **Build tooling** - Full make command reference
7. **Protocol details** - Serial communication specifics
8. **Performance characteristics** - Resource usage expectations

### Nice-to-Have Information
9. **Troubleshooting guide** - Common issues
10. **Known limitations** - What doesn't work

---

## Implementation Priority

**Phase 1 (High Impact):**
1. CLI Command Reference
2. Complete Configuration Schema
3. Observability & Debugging

**Phase 2 (Developer Experience):**
4. Testing Infrastructure Details
5. Complete Build & Quality Tooling
6. Deployment & Production

**Phase 3 (Reference):**
7. Protocol Implementation Details
8. Performance Characteristics
9. Known Limitations & Troubleshooting
10. Structural Reorganization

---

## Estimated Impact

**Before Enhancements:**
- CLAUDE.md: ~170 lines
- Coverage: Architecture + basic commands
- Audience: Developers familiar with the codebase

**After Enhancements:**
- CLAUDE.md: ~500-600 lines
- Coverage: Complete development guide
- Audience: New contributors + Claude Code + advanced users

**Benefits:**
- Faster onboarding for new contributors
- Better Claude Code assistance with complete context
- Self-service troubleshooting
- Comprehensive development reference
- Production deployment guidance
