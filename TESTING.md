# Testing Without Hardware

This document explains how to develop, test, and debug the Framework LED Matrix daemon without access to the physical hardware.

## 🧪 Testing Methods

### 1. **Comprehensive Test Suite**

Run the full test suite with mocked hardware components:

```bash
# Run all tests (no hardware required)
make test

# Run with detailed coverage report
make test-coverage

# Run individual package tests
go test -v ./internal/matrix      # LED matrix communication
go test -v ./internal/stats       # System statistics collection
go test -v ./internal/visualizer  # Display pattern generation
go test -v ./internal/daemon      # Service management
go test -v ./internal/config      # Configuration management
```

**Coverage Results:**
- Config package: 85.5% coverage
- Stats package: 87.3% coverage  
- Visualizer package: 97.5% coverage
- Matrix package: 79.9% coverage
- Daemon package: 23.1% coverage (limited by system dependencies)

### 2. **Visual LED Simulator**

Run a real-time ASCII simulation of the LED matrix display:

```bash
# Basic simulation with default settings
make simulator

# Custom configuration examples
make simulator ARGS="-mode percentage -metric cpu -duration 60s"
make simulator ARGS="-mode activity -metric memory -duration 30s" 
make simulator ARGS="-mode status -metric cpu -interval 1s"
make simulator ARGS="-config configs/config.yaml -duration 120s"
```

**Example Output:**
```
🔥 Framework LED Matrix Simulator
=================================
Mode: percentage | Metric: cpu | Duration: 30s

⏰ 14:32:15 | Mode: percentage | Metric: cpu
📊 CPU: 45.2% | Memory: 67.8% | Disk: 2.1 MB/s | Network: 0.3 MB/s

🔲 LED Matrix Simulation (34x9):
┌──────────────────────────────────┐
│███████████████░░░░░░░░░░░░░░░░░░│
│███████████████░░░░░░░░░░░░░░░░░░│
│███████████████░░░░░░░░░░░░░░░░░░│
│███████████████░░░░░░░░░░░░░░░░░░│
└──────────────────────────────────┘

💡 Brightness: 200/255 | Updates: 2s
```

### 3. **Mock Hardware Components**

The codebase includes comprehensive mocks for all hardware interfaces:

#### **MockClient** (LED Matrix)
```go
// From internal/matrix/display_test.go
type MockClient struct {
    commands          []Command
    brightness        byte
    lastPercentage    byte
    lastPattern       string
    animationEnabled  bool
    connectionError   error
}
```

#### **MockPort** (Serial Communication)
```go
// From internal/matrix/client_test.go  
type MockPort struct {
    writeData   []byte
    readData    []byte
    isOpen      bool
    writeError  error
    readError   error
}
```

#### **MockDisplayManager** (Display Control)
```go
// From cmd/simulator/main.go
type MockDisplayManager struct {
    currentPattern []byte
    brightness     byte
    lastUpdate     time.Time
}
```

### 4. **Unit Test Examples**

Test individual components in isolation:

```bash
# Test LED matrix command generation
go test -v ./internal/matrix -run TestBrightnessCommand

# Test system statistics collection
go test -v ./internal/stats -run TestCollectorCollectSystemStats

# Test visualization pattern generation  
go test -v ./internal/visualizer -run TestVisualizerCreateProgressBar

# Test configuration loading and validation
go test -v ./internal/config -run TestConfigValidation
```

### 5. **Integration Testing**

Test component interactions without hardware:

```bash
# Test daemon service lifecycle
go test -v ./internal/daemon -run TestServiceLifecycle

# Test end-to-end visualization pipeline
go test -v ./internal/visualizer -run TestVisualizerUpdateDisplay

# Test error handling and recovery
go test -v ./internal/matrix -run TestDisplayManagerErrorHandling
```

## 🛠️ Development Workflow

### **1. TDD Development Process**

```bash
# 1. Write failing test
go test -v ./internal/matrix -run TestNewFeature

# 2. Implement feature with mocks
# (edit source files)

# 3. Run tests until passing
go test -v ./internal/matrix -run TestNewFeature

# 4. Run full test suite
make test
```

### **2. Visual Testing Workflow**

```bash
# 1. Test display patterns visually
make simulator ARGS="-mode percentage -duration 10s"

# 2. Test different metrics
for metric in cpu memory disk network; do
    echo "Testing $metric metric..."
    make simulator ARGS="-metric $metric -duration 5s"
done

# 3. Test configuration changes
make simulator ARGS="-config test-config.yaml"
```

### **3. Performance Testing**

```bash
# Test with race detection
make test-race

# Benchmark tests
make test-bench  

# Concurrent access testing
go test -v ./internal/matrix -run TestDisplayManagerConcurrency
```

## 🐛 Debugging Without Hardware

### **1. Mock Error Injection**

```go
// Simulate hardware failures in tests
mockClient.SetConnectionError(errors.New("serial port disconnected"))
err := displayManager.UpdatePercentage("cpu", 75.0)
// Test error handling without real hardware failure
```

### **2. Logging and Instrumentation**

```bash
# Run with debug logging
go run ./cmd/daemon -log-level debug run

# Monitor mock interactions
go test -v ./internal/matrix -run TestClientSendCommand
```

### **3. Configuration Testing**

```bash
# Test invalid configurations
echo "invalid: yaml: content" > test-config.yaml
go run ./cmd/daemon -config test-config.yaml validate

# Test edge cases
go test -v ./internal/config -run TestConfigValidation
```

## 📋 Test Coverage Analysis

View detailed test coverage:

```bash
# Generate HTML coverage report
make test-coverage
open coverage.html

# Check coverage by package
go tool cover -func=coverage.out

# Analyze uncovered code
go tool cover -html=coverage.out
```

## 🎯 Validation Checklist

Before deploying to hardware, ensure:

- [ ] All tests pass: `make test`
- [ ] Visual simulation works: `make simulator`
- [ ] Configuration validates: `./bin/framework-led-daemon validate`
- [ ] Service management works: `go test ./internal/daemon`
- [ ] Error handling robust: Test with mock failures
- [ ] Performance acceptable: `make test-bench`
- [ ] Race conditions resolved: `make test-race`

## 🔧 Hardware Simulation Accuracy

The mocks provide high-fidelity simulation:

### **Protocol Accuracy**
- Exact Framework LED Matrix command format (magic bytes: 0x32 0xAC)
- Proper serial communication simulation
- Realistic timing and throttling behavior

### **Visual Accuracy**  
- Correct 34x9 pixel matrix dimensions
- Accurate brightness levels (0-255)
- Proper pattern generation algorithms
- Real system statistics integration

### **Behavioral Accuracy**
- Proper update rate throttling
- State management and persistence
- Error conditions and recovery
- Threading and concurrency handling

This comprehensive testing approach ensures the daemon works reliably when deployed to actual Framework LED Matrix hardware, while enabling full development and debugging capabilities without requiring physical access to the hardware.