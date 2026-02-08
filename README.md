# Framework LED Matrix Daemon

[![CI](https://github.com/timfallmk/framework-led-matrix-daemon/actions/workflows/ci.yml/badge.svg)](https://github.com/timfallmk/framework-led-matrix-daemon/actions/workflows/ci.yml)
[![CodeRabbit Reviews](https://coderabbit.ai/images/badge.svg)](https://coderabbit.ai)

A cross-platform Go daemon that displays real-time system statistics on Framework Laptop LED matrices. Monitor CPU usage, memory consumption, disk activity, and system status directly on your laptop's LED matrix input modules. Supports both single and dual matrix configurations for enhanced monitoring capabilities.

## Features

- **Real-time System Monitoring**: CPU, memory, disk I/O, and network statistics
- **Dual Matrix Support**: Configure up to two LED matrices with different display modes
- **Multiple Display Modes**: Percentage bars, gradients, activity indicators, and status displays
- **Cross-platform Support**: Linux, Windows with automated service management
- **Configurable Thresholds**: Customizable warning and critical levels
- **Automatic Port Discovery**: Finds Framework LED matrices automatically
- **Flexible Matrix Modes**: Mirror, split, extended, and independent dual matrix configurations
- **Daemonizable**: Full systemd integration with proper service management
- **YAML Configuration**: Flexible configuration with command-line overrides

## Quick Start

### Build and Install

```bash
# Clone the repository
git clone https://github.com/timfallmk/framework-led-matrix-daemon
cd framework-led-matrix-daemon

# Build the daemon
make build

# Test LED matrix connection (single or dual)
./bin/framework-led-daemon test

# Run in foreground for testing
./bin/framework-led-daemon run

# Install as system service (Linux)
sudo make install
sudo systemctl enable framework-led-daemon
sudo systemctl start framework-led-daemon
```

### Usage Examples

```bash
# Single Matrix Examples
framework-led-daemon -config /path/to/config.yaml run
framework-led-daemon -port /dev/ttyACM0 -brightness 128 run
framework-led-daemon -metric memory -mode percentage run

# Dual Matrix Examples (requires dual matrix configuration)
framework-led-daemon -config /path/to/dual-matrix-config.yaml run

# Service Management
framework-led-daemon install
framework-led-daemon start
framework-led-daemon status
framework-led-daemon stop
framework-led-daemon remove
```

## Configuration

The daemon uses YAML configuration files with the following search order:

1. Path specified by `-config` flag
2. `$XDG_CONFIG_HOME/framework-led-daemon/config.yaml`
3. `$HOME/.config/framework-led-daemon/config.yaml`
4. `/etc/framework-led-daemon/config.yaml`
5. `./configs/config.yaml`

### Single Matrix Configuration

For single matrix setups (default):

```yaml
matrix:
  port: ""                    # Auto-discover if empty
  baud_rate: 115200          # Serial communication baud rate
  auto_discover: true        # Automatically find LED matrix port
  brightness: 100            # LED brightness (0-255)

stats:
  collect_interval: 2s       # How often to collect system statistics
  enable_cpu: true           # Enable CPU monitoring
  enable_memory: true        # Enable memory monitoring
  enable_disk: true          # Enable disk monitoring
  thresholds:
    cpu_warning: 70.0        # CPU usage warning threshold (%)
    cpu_critical: 90.0       # CPU usage critical threshold (%)
    memory_warning: 80.0     # Memory usage warning threshold (%)
    memory_critical: 95.0    # Memory usage critical threshold (%)

display:
  update_rate: 1s            # How often to update the display
  mode: "percentage"         # Display mode: percentage, gradient, activity, status
  primary_metric: "cpu"      # Primary metric: cpu, memory, disk, network
  show_activity: true        # Show activity indicators
```

### Dual Matrix Configuration

For dual matrix setups with two LED matrices:

```yaml
matrix:
  # Legacy single matrix settings (used as fallback)
  port: ""                    # Auto-discover if empty
  baud_rate: 115200          # Serial communication baud rate
  auto_discover: true        # Automatically find LED matrix port
  brightness: 100            # Default brightness (0-255)
  
  # Dual matrix configuration
  dual_mode: "split"         # Dual matrix mode: mirror, split, extended, independent
  matrices:
    - name: "primary"        # Primary matrix (left side)
      port: ""               # Auto-discover if empty
      role: "primary"        # Matrix role: primary, secondary
      brightness: 100        # Individual brightness control
      metrics: ["cpu", "memory"]  # Metrics to display on this matrix
    - name: "secondary"      # Secondary matrix (right side)
      port: ""               # Auto-discover if empty  
      role: "secondary"      # Matrix role: primary, secondary
      brightness: 100        # Individual brightness control
      metrics: ["disk", "network"]  # Metrics to display on this matrix

stats:
  collect_interval: 2s       # How often to collect system statistics
  enable_cpu: true           # Enable CPU monitoring
  enable_memory: true        # Enable memory monitoring
  enable_disk: true          # Enable disk monitoring
  enable_network: true       # Enable network monitoring (recommended for dual matrix)
  thresholds:
    cpu_warning: 70.0        # CPU usage warning threshold (%)
    cpu_critical: 90.0       # CPU usage critical threshold (%)
    memory_warning: 80.0     # Memory usage warning threshold (%)
    memory_critical: 95.0    # Memory usage critical threshold (%)

display:
  update_rate: 1s            # How often to update the display
  mode: "percentage"         # Display mode: percentage, gradient, activity, status
  primary_metric: "cpu"      # Primary metric: cpu, memory, disk, network
  show_activity: true        # Show activity indicators
```

### Dual Matrix Modes

- **Mirror Mode** (`dual_mode: "mirror"`): Both matrices display identical content
- **Split Mode** (`dual_mode: "split"`): Each matrix displays different metrics (default)
- **Extended Mode** (`dual_mode: "extended"`): Wide visualization across both matrices
- **Independent Mode** (`dual_mode: "independent"`): Completely separate configurations

## Display Modes

The Framework LED Matrix (9x34 pixels) supports multiple visualization modes for system monitoring. Each mode provides different insights into your system's performance.

### 1. **Percentage Mode** (`display_mode: "percentage"`)

Shows system metrics as dynamic progress bars across the LED matrix:

```
CPU Usage (75%):
┌─────────────────────────────────┐
│████████████████████████░░░░░░░░│ ← 75% filled
└─────────────────────────────────┘

Memory Usage (50%):
┌─────────────────────────────────┐
│█████████████████░░░░░░░░░░░░░░░░│ ← 50% filled
└─────────────────────────────────┘

Network Activity (25%):
┌─────────────────────────────────┐
│████████░░░░░░░░░░░░░░░░░░░░░░░░░│ ← 25% filled
└─────────────────────────────────┘
```

**Best for:** Real-time monitoring of specific metrics, development workstations, compile progress tracking.

**Configuration Example:**
```yaml
display:
  mode: "percentage"
  primary_metric: "cpu"        # cpu, memory, disk, network
  update_rate: 1s
  brightness: 200
```

### 2. **Gradient Mode** (`display_mode: "gradient"`)

Displays smooth gradient patterns representing overall system load:

```
System Load Gradient:
┌─────────────────────────────────┐
│░░░▒▒▒▓▓▓███▓▓▓▒▒▒░░░▒▒▒▓▓▓███│ ← Gradient pattern
│░░▒▒▓▓███████████▓▓▒▒░░▒▒▓▓███│   representing
│▒▒▓▓███████████████▓▓▒▒▓▓██████│   overall system
│▓▓███████████████████▓▓████████│   activity level
└─────────────────────────────────┘
```

**Best for:** Ambient monitoring, aesthetic appeal, low-distraction environments.

**Configuration Example:**
```yaml
display:
  mode: "gradient"
  brightness: 128
  animation: true
  update_rate: 3s
```

### 3. **Activity Mode** (`display_mode: "activity"`)

Shows real-time system activity with dynamic patterns that change based on system load:

**High Activity (Zig-Zag Pattern):**
```
High CPU/Disk/Network Activity:
┌─────────────────────────────────┐
│█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█│ ← Animated zig-zag
│░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░│   pattern shows
│█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█│   system is active
│░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░│
└─────────────────────────────────┘
```

**Low Activity (Smooth Gradient):**
```
Low System Activity:
┌─────────────────────────────────┐
│░░░░▒▒▒▒▓▓▓▓████▓▓▓▓▒▒▒▒░░░░│ ← Calm gradient
│░░▒▒▒▓▓▓██████████▓▓▓▒▒▒░░│   shows system
│▒▒▓▓████████████████▓▓▒▒│   is idle
│▓▓██████████████████▓▓│
└─────────────────────────────────┘
```

**Best for:** Gaming setups, performance monitoring, understanding system behavior patterns.

**Configuration Example:**
```yaml
display:
  mode: "activity"
  brightness: 255
  animation: true
  update_rate: 1s
stats:
  thresholds:
    cpu_warning: 60.0    # Lower threshold for more sensitive activity detection
```

### 4. **Status Mode** (`display_mode: "status"`)

Color-coded system health indicators with distinct patterns for each status level:

**Normal Status (Green):**
```
System Health: NORMAL
┌─────────────────────────────────┐
│    ░░░░░░░░░░░░░░░░░░░░░░░░    │ ← Soft gradient
│  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │   (low intensity)
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│   
│  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
└─────────────────────────────────┘
```

**Warning Status (Orange):**
```
System Health: WARNING
┌─────────────────────────────────┐
│█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█│ ← Zig-zag pattern
│░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░│   (medium intensity)
│█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█│   
│░█░█░█░█░█░█░█░█░█░█░█░█░█░█░█░│
└─────────────────────────────────┘
```

**Critical Status (Red):**
```
System Health: CRITICAL
┌─────────────────────────────────┐
│███████████████████████████████│ ← Solid pattern
│███████████████████████████████│   (high intensity)
│███████████████████████████████│   
│███████████████████████████████│
└─────────────────────────────────┘
```

**Best for:** Server monitoring, system administration, alert-focused environments.

**Configuration Example:**
```yaml
display:
  mode: "status"
  brightness: 180
  animation: false
stats:
  thresholds:
    cpu_warning: 70.0
    cpu_critical: 85.0
    memory_warning: 80.0
    memory_critical: 95.0
```

## Configuration Examples by Use Case

### 🎮 Gaming Setup
Bright, responsive activity monitoring for performance awareness:

```yaml
matrix:
  brightness: 255
  auto_discover: true

display:
  mode: "activity"
  primary_metric: "cpu"
  update_rate: 1s
  animation: true

stats:
  collect_interval: 1s
  thresholds:
    cpu_warning: 70
    cpu_critical: 85
    memory_warning: 75
    memory_critical: 90
```
**Visual Result:** Animated zig-zag patterns during gaming sessions, smooth gradients during idle periods.

### 🖥️ Server Monitoring
Reliable status indicators for system health:

```yaml
matrix:
  brightness: 128
  auto_discover: true

display:
  mode: "status"
  primary_metric: "memory"
  update_rate: 5s
  animation: false

stats:
  collect_interval: 3s
  thresholds:
    memory_warning: 80
    memory_critical: 95
    cpu_warning: 80
    cpu_critical: 95
```
**Visual Result:** Steady status indicators - green for normal operation, orange for resource pressure, red for critical states.

### 💻 Development Workstation
Real-time progress bars for compilation and system load:

```yaml
matrix:
  brightness: 200
  auto_discover: true

display:
  mode: "percentage"
  primary_metric: "cpu"
  update_rate: 2s
  animation: true

stats:
  collect_interval: 2s
  enable_cpu: true
  enable_memory: true
  enable_disk: true
```
**Visual Result:** Dynamic CPU usage bars that update every 2 seconds, perfect for monitoring compile times and system responsiveness.

### 🌙 Ambient Monitoring
Subtle, aesthetic system awareness:

```yaml
matrix:
  brightness: 80
  auto_discover: true

display:
  mode: "gradient"
  update_rate: 10s
  animation: true

stats:
  collect_interval: 5s
```
**Visual Result:** Gentle gradient patterns that subtly shift based on system load, providing awareness without distraction.

### 🖥️💻 Dual Matrix Power User
Enhanced monitoring with two matrices showing different metrics:

```yaml
matrix:
  # Enable dual matrix mode
  dual_mode: "split"
  matrices:
    - name: "primary"
      role: "primary"
      brightness: 200
      metrics: ["cpu", "memory"]    # Left matrix: CPU and Memory
    - name: "secondary"
      role: "secondary"
      brightness: 200
      metrics: ["disk", "network"]  # Right matrix: Disk and Network

display:
  mode: "percentage"
  update_rate: 1s
  animation: true

stats:
  collect_interval: 1s
  enable_cpu: true
  enable_memory: true
  enable_disk: true
  enable_network: true
```
**Visual Result:** Left matrix shows CPU/Memory bars, right matrix shows Disk/Network activity - complete system visibility at a glance.

### 🖥️🖥️ Dual Matrix Mirror Mode
Identical content on both matrices for enhanced visibility:

```yaml
matrix:
  dual_mode: "mirror"
  matrices:
    - name: "primary"
      brightness: 255
    - name: "secondary"
      brightness: 255

display:
  mode: "activity"
  update_rate: 1s
```
**Visual Result:** Both matrices display the same activity patterns, perfect for wide viewing angles or presentations.

## Display Features

- **Brightness Control:** 0-255 intensity levels for any lighting condition
- **Animation Support:** Pulsing, scrolling, and pattern transitions
- **Update Rates:** 1-30 second intervals for smooth or battery-friendly operation
- **Auto-Discovery:** Automatically detects Framework LED Matrix hardware
- **Hot Configuration:** Changes apply immediately without service restart

## System Requirements

- Framework Laptop with LED Matrix input module(s)
  - Single matrix: Works with one LED matrix module
  - Dual matrix: Supports up to two LED matrix modules simultaneously
- Go 1.25 or later (for building)
- Serial port access permissions
- Linux: `udev` rules or user in `dialout` group
- Windows: Administrator privileges for service installation

## Building

### Prerequisites

```bash
# Install Go 1.25+
# Install make (usually pre-installed on Linux)

# Clone and build
git clone https://github.com/timfallmk/framework-led-matrix-daemon
cd framework-led-matrix-daemon
make deps
make build
```

### Cross-compilation

```bash
# Build for all supported platforms
make cross-compile

# Create release packages
make release
```

## Installation

### Linux (systemd)

```bash
# Build and install
make install

# Enable and start service
sudo systemctl enable framework-led-daemon
sudo systemctl start framework-led-daemon

# Check status
sudo systemctl status framework-led-daemon

# View logs
sudo journalctl -u framework-led-daemon -f
```

### Manual Installation

```bash
# Copy binary
sudo cp bin/framework-led-daemon /usr/local/bin/

# Copy configuration
sudo mkdir -p /etc/framework-led-daemon
sudo cp configs/config.yaml /etc/framework-led-daemon/

# Run daemon
framework-led-daemon run
```

## Troubleshooting

### Permission Issues

```bash
# Add user to dialout group (Linux)
sudo usermod -a -G dialout $USER

# Set udev rules for Framework LED Matrix
echo 'SUBSYSTEM=="tty", ATTRS{idVendor}=="32ac", MODE="0666"' | sudo tee /etc/udev/rules.d/99-framework-led.rules
sudo udevadm control --reload-rules
```

### Connection Issues

```bash
# Test LED matrix connection
framework-led-daemon test

# List available serial ports
framework-led-daemon -log-level debug test

# Manually specify port
framework-led-daemon -port /dev/ttyACM0 test
```

### Service Issues

```bash
# Check service status
sudo systemctl status framework-led-daemon

# View detailed logs
sudo journalctl -u framework-led-daemon -f

# Restart service
sudo systemctl restart framework-led-daemon
```

## Development

### Project Structure

```
├── cmd/daemon/           # Main CLI application
├── internal/
│   ├── config/          # Configuration management
│   ├── matrix/          # LED Matrix communication
│   ├── stats/           # System statistics collection
│   ├── daemon/          # Service lifecycle management
│   └── visualizer/      # Display pattern mapping
├── configs/             # Default configuration
├── systemd/             # systemd service files
└── Makefile            # Build automation
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make test` and `make fmt`
6. Submit a pull request

## Architecture

The daemon consists of several key components:

- **Matrix Client**: Handles serial communication with Framework LED matrices using the documented protocol
- **Stats Collector**: Gathers system metrics using `gopsutil` library
- **Visualizer**: Maps system statistics to LED display patterns
- **Service Manager**: Handles daemon lifecycle and cross-platform service integration
- **Configuration**: YAML-based configuration with validation and hot-reloading

## Protocol Support

Based on the Framework Input Module protocol:
- Magic bytes: `0x32 0xAC`
- Supported commands: Brightness, Pattern, Animation, Custom drawing
- Baud rate: 115200
- Auto-discovery via USB VID/PID detection

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Framework Computer for the open-source LED matrix protocol documentation
- The Go community for excellent system monitoring libraries (`gopsutil`, `serial`)
- Contributors to the `takama/daemon` library for cross-platform service management
