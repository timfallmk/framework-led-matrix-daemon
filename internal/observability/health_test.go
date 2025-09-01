package observability

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/logging"
)

func TestErrString(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "non-nil error",
			err:      errors.New("test error"),
			expected: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errString(tt.err)
			if result != tt.expected {
				t.Errorf("errString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHealthStatus(t *testing.T) {
	tests := []struct {
		name   string
		status HealthStatus
		want   string
	}{
		{"healthy", StatusHealthy, "healthy"},
		{"unhealthy", StatusUnhealthy, "unhealthy"},
		{"unknown", StatusUnknown, "unknown"},
		{"starting", StatusStarting, "starting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("HealthStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestNewHealthMonitor(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)
	monitor := NewHealthMonitor(logger, appMetrics, time.Second)

	if monitor == nil {
		t.Fatal("NewHealthMonitor() returned nil")
	}

	if monitor.checkers == nil {
		t.Error("NewHealthMonitor() did not initialize checkers map")
	}

	if monitor.results == nil {
		t.Error("NewHealthMonitor() did not initialize results map")
	}
}

func TestHealthMonitor_GetHealth(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)
	monitor := NewHealthMonitor(logger, appMetrics, time.Second)

	// Initially should be empty
	health := monitor.GetHealth()
	if len(health) != 0 {
		t.Errorf("GetHealth() initial count = %d, want 0", len(health))
	}
}

func TestHealthMonitor_GetOverallHealth(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)
	monitor := NewHealthMonitor(logger, appMetrics, time.Second)

	// Test with no checks
	overall := monitor.GetOverallHealth()
	if overall != StatusUnknown {
		t.Errorf("GetOverallHealth() with no checks = %v, want %v", overall, StatusUnknown)
	}
}

func TestHealthMonitor_IsHealthy(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)
	monitor := NewHealthMonitor(logger, appMetrics, time.Second)

	// Test with no checks
	if monitor.IsHealthy() {
		t.Error("IsHealthy() with no checks should return false")
	}
}

func TestMatrixHealthChecker(t *testing.T) {
	// Test creating a matrix health checker with a simple test function
	testFunc := func(ctx context.Context) error {
		return nil // healthy
	}

	checker := NewMatrixHealthChecker("test_matrix", testFunc)

	if checker.Name() != "test_matrix" {
		t.Errorf("Name() = %v, want test_matrix", checker.Name())
	}

	if checker.Timeout() != 5*time.Second {
		t.Errorf("Timeout() = %v, want %v", checker.Timeout(), 5*time.Second)
	}

	// Test check
	err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("Check() returned error: %v", err)
	}
}

func TestStatsHealthChecker(t *testing.T) {
	// Test creating a stats health checker with a simple test function
	testFunc := func(ctx context.Context) error {
		return nil // healthy
	}

	checker := NewStatsHealthChecker("test_stats", testFunc)

	if checker.Name() != "test_stats" {
		t.Errorf("Name() = %v, want test_stats", checker.Name())
	}

	if checker.Timeout() != 3*time.Second {
		t.Errorf("Timeout() = %v, want %v", checker.Timeout(), 3*time.Second)
	}

	// Test check
	err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("Check() returned error: %v", err)
	}
}

func TestConfigHealthChecker(t *testing.T) {
	// Test creating a config health checker with a simple test function
	testFunc := func(ctx context.Context) error {
		return nil // healthy
	}

	checker := NewConfigHealthChecker("test_config", testFunc)

	if checker.Name() != "test_config" {
		t.Errorf("Name() = %v, want test_config", checker.Name())
	}

	if checker.Timeout() != 2*time.Second {
		t.Errorf("Timeout() = %v, want %v", checker.Timeout(), 2*time.Second)
	}

	// Test check
	err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("Check() returned error: %v", err)
	}
}

func TestMemoryHealthChecker(t *testing.T) {
	checker := NewMemoryHealthChecker("test_memory", 80*1024*1024*1024) // 80GB threshold

	if checker.Name() != "test_memory" {
		t.Errorf("Name() = %v, want test_memory", checker.Name())
	}

	if checker.Timeout() != 1*time.Second {
		t.Errorf("Timeout() = %v, want %v", checker.Timeout(), 1*time.Second)
	}

	// Test check - this should work in any environment
	err := checker.Check(context.Background())
	if err != nil {
		// Memory check might fail on some systems, but shouldn't panic
		t.Logf("Memory check failed (expected on some systems): %v", err)
	}
}

func TestDiskSpaceHealthChecker(t *testing.T) {
	checker := NewDiskSpaceHealthChecker("test_disk", os.TempDir(), 80*1024*1024*1024) // 80GB threshold

	if checker.Name() != "test_disk" {
		t.Errorf("Name() = %v, want test_disk", checker.Name())
	}

	if checker.Timeout() != 2*time.Second {
		t.Errorf("Timeout() = %v, want %v", checker.Timeout(), 2*time.Second)
	}

	// Test check - this should work in any Unix environment
	err := checker.Check(context.Background())
	if err != nil {
		// Disk check might fail on some systems, but shouldn't panic
		t.Logf("Disk check failed (expected on some systems): %v", err)
	}
}
