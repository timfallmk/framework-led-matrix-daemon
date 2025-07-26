# Framework LED Matrix Daemon

A cross-platform Go daemon that displays real-time system statistics on Framework Laptop LED matrices. Monitor CPU usage, memory consumption, disk activity, and system status directly on your laptop's LED matrix input modules.

## Features

- **Real-time System Monitoring**: CPU, memory, disk I/O, and network statistics
- **Multiple Display Modes**: Percentage bars, gradients, activity indicators, and status displays
- **Cross-platform Support**: Linux, Windows, macOS with automated service management
- **Configurable Thresholds**: Customizable warning and critical levels
- **Automatic Port Discovery**: Finds Framework LED matrices automatically
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

# Test LED matrix connection
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
# Run with custom configuration
framework-led-daemon -config /path/to/config.yaml run

# Run with command-line overrides
framework-led-daemon -port /dev/ttyACM0 -brightness 128 run

# Display memory usage instead of CPU
framework-led-daemon -metric memory -mode percentage run

# Install and manage as system service
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

### Configuration Options

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

## Display Modes

- **Percentage**: Shows metric as a progress bar (0-100%)
- **Gradient**: Static gradient pattern with optional animation
- **Activity**: Shows activity indicators for disk/network I/O
- **Status**: Color-coded status based on system health (normal/warning/critical)

## System Requirements

- Framework Laptop with LED Matrix input module
- Go 1.24.5 or later (for building)
- Serial port access permissions
- Linux: `udev` rules or user in `dialout` group
- Windows: Administrator privileges for service installation
- macOS: No additional requirements

## Building

### Prerequisites

```bash
# Install Go 1.24.5+
# Install make (usually pre-installed on Linux/macOS)

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
