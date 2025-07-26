package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/timfa/framework-led-matrix-daemon/internal/config"
	"github.com/timfa/framework-led-matrix-daemon/internal/stats"
)

// CreateTempConfig creates a temporary configuration file for testing
func CreateTempConfig(t *testing.T, configData string) string {
	t.Helper()
	
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	configFile := filepath.Join(tmpDir, "test_config.yaml")
	err = os.WriteFile(configFile, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Clean up function
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	
	return configFile
}

// CreateTestConfig creates a test configuration with reasonable defaults
func CreateTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	
	// Set test-friendly values
	cfg.Stats.CollectInterval = 100 * time.Millisecond
	cfg.Display.UpdateRate = 50 * time.Millisecond
	cfg.Matrix.Port = "mock_port"
	cfg.Matrix.Brightness = 128
	
	return cfg
}

// CreateTestStats creates test system statistics for testing
func CreateTestStats() *stats.SystemStats {
	now := time.Now()
	
	return &stats.SystemStats{
		CPU: stats.CPUStats{
			UsagePercent:   75.5,
			PerCorePercent: []float64{70.0, 80.0, 75.0, 76.0},
			PhysicalCores:  4,
			LogicalCores:   8,
			ModelName:      "Test CPU",
			VendorID:       "TestVendor",
		},
		Memory: stats.MemoryStats{
			Total:       16 * 1024 * 1024 * 1024, // 16GB
			Available:   8 * 1024 * 1024 * 1024,  // 8GB
			Used:        8 * 1024 * 1024 * 1024,  // 8GB
			UsedPercent: 50.0,
			Free:        8 * 1024 * 1024 * 1024, // 8GB
			SwapTotal:   4 * 1024 * 1024 * 1024, // 4GB
			SwapUsed:    1024 * 1024 * 1024,     // 1GB
			SwapPercent: 25.0,
		},
		Disk: stats.DiskStats{
			Partitions: []stats.PartitionStat{
				{
					Device:      "/dev/sda1",
					Mountpoint:  "/",
					Fstype:      "ext4",
					Total:       1024 * 1024 * 1024 * 1024, // 1TB
					Used:        512 * 1024 * 1024 * 1024,  // 512GB
					Free:        512 * 1024 * 1024 * 1024,  // 512GB
					UsedPercent: 50.0,
				},
			},
			IOCounters: map[string]stats.IOCounterStat{
				"sda": {
					ReadCount:  10000,
					WriteCount: 5000,
					ReadBytes:  1024 * 1024 * 100, // 100MB
					WriteBytes: 1024 * 1024 * 50,  // 50MB
					ReadTime:   1000,
					WriteTime:  500,
				},
			},
			TotalReads:   10000,
			TotalWrites:  5000,
			ReadBytes:    1024 * 1024 * 100,
			WriteBytes:   1024 * 1024 * 50,
			ActivityRate: 1024.0 * 1024.0, // 1MB/s
		},
		Network: stats.NetworkStats{
			BytesSent:      1024 * 1024 * 100, // 100MB
			BytesRecv:      1024 * 1024 * 200, // 200MB
			PacketsSent:    10000,
			PacketsRecv:    20000,
			TotalBytesSent: 1024 * 1024 * 500,  // 500MB
			TotalBytesRecv: 1024 * 1024 * 1000, // 1GB
			ActivityRate:   512.0 * 1024.0,    // 512KB/s
		},
		Timestamp: now,
		Uptime:    24 * time.Hour,
		LoadAvg:   []float64{1.5, 2.0, 2.5},
	}
}

// CreateTestSummary creates a test stats summary
func CreateTestSummary(status stats.SystemStatus) *stats.StatsSummary {
	return &stats.StatsSummary{
		CPUUsage:        75.5,
		MemoryUsage:     60.2,
		DiskActivity:    1024.0 * 1024.0, // 1MB/s
		NetworkActivity: 512.0 * 1024.0,  // 512KB/s
		Status:          status,
		Timestamp:       time.Now(),
	}
}

// AssertFloatEqual asserts that two float64 values are equal within a tolerance
func AssertFloatEqual(t *testing.T, actual, expected, tolerance float64, msg string) {
	t.Helper()
	
	if abs(actual-expected) > tolerance {
		t.Errorf("%s: actual=%.3f, expected=%.3f (tolerance=%.3f)", msg, actual, expected, tolerance)
	}
}

// AssertDurationEqual asserts that two durations are equal within a tolerance
func AssertDurationEqual(t *testing.T, actual, expected, tolerance time.Duration, msg string) {
	t.Helper()
	
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}
	
	if diff > tolerance {
		t.Errorf("%s: actual=%v, expected=%v (tolerance=%v)", msg, actual, expected, tolerance)
	}
}

// AssertTimeRecent asserts that a timestamp is recent (within the last few seconds)
func AssertTimeRecent(t *testing.T, timestamp time.Time, maxAge time.Duration, msg string) {
	t.Helper()
	
	age := time.Since(timestamp)
	if age > maxAge {
		t.Errorf("%s: timestamp %v is too old (age=%v, maxAge=%v)", msg, timestamp, age, maxAge)
	}
}

// AssertStringNotEmpty asserts that a string is not empty
func AssertStringNotEmpty(t *testing.T, value, name string) {
	t.Helper()
	
	if value == "" {
		t.Errorf("%s should not be empty", name)
	}
}

// AssertPercentageValid asserts that a percentage value is between 0 and 100
func AssertPercentageValid(t *testing.T, value float64, name string) {
	t.Helper()
	
	if value < 0 || value > 100 {
		t.Errorf("%s should be between 0 and 100, got %.2f", name, value)
	}
}

// AssertBytesNonNegative asserts that a byte count is non-negative
func AssertBytesNonNegative(t *testing.T, value uint64, name string) {
	t.Helper()
	
	// uint64 is always non-negative, but we include this for consistency
	// and in case the type changes in the future
	if value < 0 {
		t.Errorf("%s should be non-negative, got %d", name, value)
	}
}

// SkipIfShort skips a test if running in short mode
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()
	
	if testing.Short() {
		t.Skipf("Skipping test in short mode: %s", reason)
	}
}

// SkipIfCI skips a test if running in CI environment
func SkipIfCI(t *testing.T, reason string) {
	t.Helper()
	
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skipf("Skipping test in CI environment: %s", reason)
	}
}

// CreateTestThresholds creates test thresholds for testing
func CreateTestThresholds() stats.Thresholds {
	return stats.Thresholds{
		CPUWarning:     60.0,
		CPUCritical:    85.0,
		MemoryWarning:  70.0,
		MemoryCritical: 90.0,
		DiskWarning:    75.0,
		DiskCritical:   95.0,
	}
}

// WaitForCondition waits for a condition to become true within a timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	
	t.Errorf("Condition not met within %v: %s", timeout, message)
}

// ExpectError asserts that an error is not nil and optionally contains a message
func ExpectError(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	
	if err == nil {
		t.Error("Expected error but got nil")
		return
	}
	
	if expectedMessage != "" && err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, err.Error())
	}
}

// ExpectNoError asserts that an error is nil
func ExpectNoError(t *testing.T, err error) {
	t.Helper()
	
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// RunConcurrently runs multiple functions concurrently and waits for completion
func RunConcurrently(t *testing.T, functions ...func()) {
	t.Helper()
	
	done := make(chan bool, len(functions))
	
	for _, fn := range functions {
		go func(f func()) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in concurrent function: %v", r)
				}
				done <- true
			}()
			f()
		}(fn)
	}
	
	// Wait for all functions to complete
	for i := 0; i < len(functions); i++ {
		select {
		case <-done:
			// Function completed
		case <-time.After(5 * time.Second):
			t.Error("Concurrent function did not complete within timeout")
			return
		}
	}
}

// CreateTestConfigYAML returns a test configuration in YAML format
func CreateTestConfigYAML() string {
	return `
matrix:
  port: "mock_port"
  baud_rate: 115200
  auto_discover: false
  brightness: 128

stats:
  collect_interval: 1s
  enable_cpu: true
  enable_memory: true
  enable_disk: true
  enable_network: false
  thresholds:
    cpu_warning: 60.0
    cpu_critical: 85.0
    memory_warning: 70.0
    memory_critical: 90.0
    disk_warning: 75.0
    disk_critical: 95.0

display:
  update_rate: 500ms
  mode: "percentage"
  primary_metric: "cpu"
  show_activity: true
  enable_animation: false

daemon:
  name: "test-daemon"
  description: "Test Daemon"
  user: ""
  group: ""
  pid_file: "/tmp/test-daemon.pid"
  log_file: "/tmp/test-daemon.log"

logging:
  level: "debug"
  file: ""
  max_size: 10
  max_backups: 3
  max_age: 28
  compress: true
`
}

// ValidateSystemStats validates that system stats are reasonable
func ValidateSystemStats(t *testing.T, stats *stats.SystemStats) {
	t.Helper()
	
	if stats == nil {
		t.Fatal("SystemStats is nil")
	}
	
	// Validate CPU stats
	AssertPercentageValid(t, stats.CPU.UsagePercent, "CPU.UsagePercent")
	
	if stats.CPU.PhysicalCores <= 0 {
		t.Error("CPU.PhysicalCores should be positive")
	}
	
	if stats.CPU.LogicalCores <= 0 {
		t.Error("CPU.LogicalCores should be positive")
	}
	
	if stats.CPU.LogicalCores < stats.CPU.PhysicalCores {
		t.Error("CPU.LogicalCores should be >= PhysicalCores")
	}
	
	// Validate per-core percentages
	for i, percent := range stats.CPU.PerCorePercent {
		AssertPercentageValid(t, percent, fmt.Sprintf("CPU.PerCorePercent[%d]", i))
	}
	
	// Validate Memory stats
	AssertPercentageValid(t, stats.Memory.UsedPercent, "Memory.UsedPercent")
	
	if stats.Memory.Total == 0 {
		t.Error("Memory.Total should be positive")
	}
	
	if stats.Memory.Used > stats.Memory.Total {
		t.Error("Memory.Used should not exceed Total")
	}
	
	if stats.Memory.Available > stats.Memory.Total {
		t.Error("Memory.Available should not exceed Total")
	}
	
	// Validate timestamp
	AssertTimeRecent(t, stats.Timestamp, 10*time.Second, "SystemStats.Timestamp")
	
	// Validate uptime is non-negative
	if stats.Uptime < 0 {
		t.Error("SystemStats.Uptime should be non-negative")
	}
}

// ValidateStatsSummary validates that a stats summary is reasonable
func ValidateStatsSummary(t *testing.T, summary *stats.StatsSummary) {
	t.Helper()
	
	if summary == nil {
		t.Fatal("StatsSummary is nil")
	}
	
	AssertPercentageValid(t, summary.CPUUsage, "StatsSummary.CPUUsage")
	AssertPercentageValid(t, summary.MemoryUsage, "StatsSummary.MemoryUsage")
	
	if summary.DiskActivity < 0 {
		t.Error("StatsSummary.DiskActivity should be non-negative")
	}
	
	if summary.NetworkActivity < 0 {
		t.Error("StatsSummary.NetworkActivity should be non-negative")
	}
	
	// Validate status
	validStatuses := []stats.SystemStatus{stats.StatusNormal, stats.StatusWarning, stats.StatusCritical}
	statusValid := false
	for _, validStatus := range validStatuses {
		if summary.Status == validStatus {
			statusValid = true
			break
		}
	}
	if !statusValid {
		t.Errorf("StatsSummary.Status is invalid: %v", summary.Status)
	}
	
	AssertTimeRecent(t, summary.Timestamp, 10*time.Second, "StatsSummary.Timestamp")
}

