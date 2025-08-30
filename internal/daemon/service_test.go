package daemon

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
)

// MockDaemon implements a mock daemon for testing service management.
type MockDaemon struct {
	installErr error
	removeErr  error
	startErr   error
	stopErr    error
	statusErr  error
	status     string
	installed  bool
	running    bool
}

func NewMockDaemon() *MockDaemon {
	return &MockDaemon{
		status: "stopped",
	}
}

func (m *MockDaemon) Install() (string, error) {
	if m.installErr != nil {
		return "", m.installErr
	}

	m.installed = true

	return "Service installed successfully", nil
}

func (m *MockDaemon) Remove() (string, error) {
	if m.removeErr != nil {
		return "", m.removeErr
	}

	m.installed = false
	m.running = false

	return "Service removed successfully", nil
}

func (m *MockDaemon) Start() (string, error) {
	if m.startErr != nil {
		return "", m.startErr
	}

	if !m.installed {
		return "", errors.New("service not installed")
	}

	m.running = true
	m.status = "running"

	return "Service started successfully", nil
}

func (m *MockDaemon) Stop() (string, error) {
	if m.stopErr != nil {
		return "", m.stopErr
	}

	m.running = false
	m.status = "stopped"

	return "Service stopped successfully", nil
}

func (m *MockDaemon) Status() (string, error) {
	if m.statusErr != nil {
		return "", m.statusErr
	}

	return m.status, nil
}

func (m *MockDaemon) Run(executable interface{}) (string, error) {
	// Mock implementation - just return success
	return "Service running", nil
}

func (m *MockDaemon) SetErrors(install, remove, start, stop, status error) {
	m.installErr = install
	m.removeErr = remove
	m.startErr = start
	m.stopErr = stop
	m.statusErr = status
}

func TestServiceCreation(t *testing.T) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if service == nil {
		t.Fatal("NewService() returned nil service")
	}

	if service.config != cfg {
		t.Error("NewService() config not set correctly")
	}

	if service.ctx == nil {
		t.Error("NewService() context not initialized")
	}

	if service.cancel == nil {
		t.Error("NewService() cancel function not initialized")
	}

	if service.stopCh == nil {
		t.Error("NewService() stop channel not initialized")
	}
}

func TestServiceInitialization(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Matrix.Port = "mock_port" // Use mock port to avoid real hardware dependency

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// This test will likely fail without actual hardware, but we can test the error handling
	err = service.Initialize()
	if err == nil {
		// If no error, verify components are initialized
		if service.matrix == nil {
			t.Error("Initialize() should set matrix client")
		}

		if service.display == nil {
			t.Error("Initialize() should set display manager")
		}

		if service.collector == nil {
			t.Error("Initialize() should set stats collector")
		}

		if service.visualizer == nil {
			t.Error("Initialize() should set visualizer")
		}
	} else {
		// Expected failure due to mock hardware - just verify error is reasonable
		if err.Error() == "" {
			t.Error("Initialize() error should have meaningful message")
		}

		t.Logf("Initialize() failed as expected with mock hardware: %v", err)
	}
}

func TestServiceLifecycle(t *testing.T) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Test service lifecycle without actual initialization
	// (since we don't have real LED matrix hardware)

	// Test context cancellation
	if service.ctx.Err() != nil {
		t.Error("Service context should not be cancelled initially")
	}

	// Test stop channel
	select {
	case <-service.stopCh:
		t.Error("Stop channel should not be closed initially")
	default:
		// Expected - channel should be open
	}

	// Test stopping the service
	err = service.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Verify context is cancelled
	if service.ctx.Err() == nil {
		t.Error("Service context should be cancelled after Stop()")
	}
}

func TestServiceConfigReload(t *testing.T) {
	// Create a temporary config for testing
	cfg := config.DefaultConfig()
	cfg.Stats.Thresholds.CPUWarning = 60.0
	cfg.Stats.Thresholds.CPUCritical = 80.0

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Simulate config reload (this is typically called by SIGHUP handler)
	// Since we can't easily test the actual signal handling in unit tests,
	// we test the reload logic directly

	// Create a new config with different thresholds
	newCfg := config.DefaultConfig()
	newCfg.Stats.Thresholds.CPUWarning = 75.0
	newCfg.Stats.Thresholds.CPUCritical = 90.0

	// Update service config
	service.config = newCfg

	// Initialize components to test threshold updates
	// (This will fail without hardware, but we can test the error handling)
	err = service.Initialize()
	if err != nil {
		t.Logf("Initialize() failed as expected: %v", err)
		// Can't test threshold updates without successful initialization
		return
	}

	// If initialization succeeded, verify thresholds are updated
	if service.collector != nil {
		thresholds := service.collector.GetThresholds()
		if thresholds.CPUWarning != 75.0 {
			t.Errorf("Reload should update CPU warning threshold to 75.0, got %.1f", thresholds.CPUWarning)
		}

		if thresholds.CPUCritical != 90.0 {
			t.Errorf("Reload should update CPU critical threshold to 90.0, got %.1f", thresholds.CPUCritical)
		}
	}
}

func TestServiceDaemonOperations(t *testing.T) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Test daemon operations (these will use the actual takama/daemon library)
	// The operations might fail depending on the test environment, but we can test the API

	// Test Status
	status, err := service.Status()
	if err != nil {
		t.Logf("Status() error (expected in test environment): %v", err)
	} else {
		if status == "" {
			t.Error("Status() should return non-empty status")
		}

		t.Logf("Service status: %s", status)
	}

	// Test Install (will likely fail without root privileges)
	_, err = service.Install()
	if err != nil {
		t.Logf("Install() error (expected without root): %v", err)
	}

	// Test Remove (will likely fail without root privileges)
	_, err = service.Remove()
	if err != nil {
		t.Logf("Remove() error (expected without root): %v", err)
	}

	// Test StartService and StopService (will likely fail without installation)
	_, err = service.StartService()
	if err != nil {
		t.Logf("StartService() error (expected without installation): %v", err)
	}

	_, err = service.StopService()
	if err != nil {
		t.Logf("StopService() error (expected without installation): %v", err)
	}
}

func TestServiceConcurrency(t *testing.T) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Test concurrent access to service operations
	var wg sync.WaitGroup

	numGoroutines := 5

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Test multiple status calls concurrently
			for j := 0; j < 3; j++ {
				_, err := service.Status()
				if err != nil {
					t.Logf("Goroutine %d Status() error: %v", id, err)
				}

				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify service is still functional
	_, err = service.Status()
	if err != nil {
		t.Logf("Status() after concurrent access error: %v", err)
	}
}

func TestServiceContextCancellation(t *testing.T) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Verify initial context state
	if service.ctx.Err() != nil {
		t.Error("Service context should not be cancelled initially")
	}

	// Test context cancellation through cancel function
	service.cancel()

	// Give some time for cancellation to propagate
	time.Sleep(1 * time.Millisecond)

	if service.ctx.Err() == nil {
		t.Error("Service context should be cancelled after calling cancel()")
	}

	// Verify context is cancelled
	select {
	case <-service.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled immediately")
	}
}

func TestServiceStopChannel(t *testing.T) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Verify stop channel is initially open
	select {
	case <-service.stopCh:
		t.Error("Stop channel should not be closed initially")
	default:
		// Expected
	}

	// Close stop channel (simulating shutdown)
	close(service.stopCh)

	// Verify stop channel is closed
	select {
	case <-service.stopCh:
		// Expected
	case <-time.After(1 * time.Millisecond):
		t.Error("Stop channel should be closed immediately")
	}
}

// Integration test that tests the full service workflow
// This test is more comprehensive but may fail without proper hardware/permissions.
func TestServiceIntegration(t *testing.T) {
	// Skip this test in short mode or CI environments
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.DefaultConfig()
	cfg.Stats.CollectInterval = 100 * time.Millisecond
	cfg.Display.UpdateRate = 100 * time.Millisecond

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Try to initialize (will likely fail without hardware)
	err = service.Initialize()
	if err != nil {
		t.Skipf("Skipping integration test due to initialization failure: %v", err)
	}

	// If initialization succeeded, test the service workflow
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start service components in separate goroutine
	done := make(chan error, 1)

	go func() {
		// Simulate running for a short time
		select {
		case <-ctx.Done():
			done <- ctx.Err()
		case <-service.stopCh:
			done <- nil
		}
	}()

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop the service
	err = service.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Wait for completion
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Service run error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Service did not stop within timeout")
	}
}

// Test error conditions.
func TestServiceErrorHandling(t *testing.T) {
	// Test with invalid config
	cfg := config.DefaultConfig()
	cfg.Daemon.Name = "" // Invalid empty name

	// This might still succeed since takama/daemon might handle empty names
	service, err := NewService(cfg)
	if err != nil {
		t.Logf("NewService() with invalid config error (expected): %v", err)

		return
	}

	// If service creation succeeded, test that operations handle errors gracefully
	if service != nil {
		// Test operations that might fail
		_, err = service.Status()
		if err != nil {
			t.Logf("Status() error with invalid config: %v", err)
		}
	}
}

// Benchmark service creation.
func BenchmarkServiceCreation(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service, err := NewService(cfg)
		if err != nil {
			b.Fatalf("NewService() error = %v", err)
		}

		// Clean up
		service.cancel()
	}
}

// Benchmark daemon status calls.
func BenchmarkServiceStatus(b *testing.B) {
	cfg := config.DefaultConfig()

	service, err := NewService(cfg)
	if err != nil {
		b.Fatalf("NewService() error = %v", err)
	}
	defer service.cancel()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Status()
		if err != nil {
			// Don't fail benchmark for expected errors in test environment
			continue
		}
	}
}

// Test service configuration validation.
func TestServiceConfigValidation(t *testing.T) {
	tests := []struct {
		configMod func(*config.Config)
		name      string
		expectErr bool
	}{
		{
			name:      "valid config",
			configMod: func(cfg *config.Config) {},
			expectErr: false,
		},
		{
			name: "empty daemon name",
			configMod: func(cfg *config.Config) {
				cfg.Daemon.Name = ""
			},
			expectErr: false, // takama/daemon might handle this
		},
		{
			name: "invalid collect interval",
			configMod: func(cfg *config.Config) {
				cfg.Stats.CollectInterval = -1 * time.Second
			},
			expectErr: false, // Service creation doesn't validate config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.configMod(cfg)

			service, err := NewService(cfg)

			if (err != nil) != tt.expectErr {
				t.Errorf("NewService() error = %v, expectErr %v", err, tt.expectErr)

				return
			}

			if service != nil {
				service.cancel() // Clean up
			}
		})
	}
}
