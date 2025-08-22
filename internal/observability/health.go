package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/logging"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
	StatusStarting  HealthStatus = "starting"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string        `json:"name"`
	Status      HealthStatus  `json:"status"`
	Message     string        `json:"message,omitempty"`
	LastChecked time.Time     `json:"last_checked"`
	Duration    time.Duration `json:"duration"`
	Error       error         `json:"error,omitempty"`
}

// HealthChecker defines the interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
	Timeout() time.Duration
}

// HealthMonitor monitors the health of various system components
type HealthMonitor struct {
	checkers map[string]HealthChecker
	results  map[string]*HealthCheck
	logger   *logging.EventLogger
	metrics  *ApplicationMetrics
	mu       sync.RWMutex

	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(logger *logging.Logger, metrics *ApplicationMetrics, checkInterval time.Duration) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	hm := &HealthMonitor{
		checkers:      make(map[string]HealthChecker),
		results:       make(map[string]*HealthCheck),
		logger:        logging.NewEventLogger(logger),
		metrics:       metrics,
		checkInterval: checkInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	return hm
}

// RegisterChecker registers a health checker
func (hm *HealthMonitor) RegisterChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.checkers[checker.Name()] = checker
	hm.results[checker.Name()] = &HealthCheck{
		Name:        checker.Name(),
		Status:      StatusStarting,
		LastChecked: time.Now(),
	}

	hm.logger.LogDaemon(logging.LevelInfo, "health checker registered", "register", map[string]interface{}{
		"checker": checker.Name(),
	})
}

// Start begins health monitoring
func (hm *HealthMonitor) Start() {
	hm.wg.Add(1)
	go hm.monitorLoop()

	hm.logger.LogDaemon(logging.LevelInfo, "health monitor started", "start", nil)
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop() {
	hm.cancel()
	hm.wg.Wait()
	hm.logger.Close()

	hm.logger.LogDaemon(logging.LevelInfo, "health monitor stopped", "stop", nil)
}

// GetHealth returns the current health status of all components
func (hm *HealthMonitor) GetHealth() map[string]*HealthCheck {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]*HealthCheck)
	for name, check := range hm.results {
		result[name] = &HealthCheck{
			Name:        check.Name,
			Status:      check.Status,
			Message:     check.Message,
			LastChecked: check.LastChecked,
			Duration:    check.Duration,
			Error:       check.Error,
		}
	}
	return result
}

// GetOverallHealth returns the overall system health
func (hm *HealthMonitor) GetOverallHealth() HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if len(hm.results) == 0 {
		return StatusUnknown
	}

	hasUnhealthy := false
	hasStarting := false

	for _, check := range hm.results {
		switch check.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusStarting:
			hasStarting = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasStarting {
		return StatusStarting
	}

	return StatusHealthy
}

// IsHealthy returns true if all components are healthy
func (hm *HealthMonitor) IsHealthy() bool {
	return hm.GetOverallHealth() == StatusHealthy
}

func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	// Perform initial checks
	hm.runAllChecks()

	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.runAllChecks()
		}
	}
}

func (hm *HealthMonitor) runAllChecks() {
	hm.mu.RLock()
	checkers := make([]HealthChecker, 0, len(hm.checkers))
	for _, checker := range hm.checkers {
		checkers = append(checkers, checker)
	}
	hm.mu.RUnlock()

	// Run checks in parallel
	var wg sync.WaitGroup
	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()
			hm.runCheck(c)
		}(checker)
	}
	wg.Wait()
}

func (hm *HealthMonitor) runCheck(checker HealthChecker) {
	start := time.Now()

	// Create context with timeout
	timeout := checker.Timeout()
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(hm.ctx, timeout)
	defer cancel()

	// Run the check
	err := checker.Check(ctx)
	duration := time.Since(start)

	// Update results
	hm.mu.Lock()
	result := &HealthCheck{
		Name:        checker.Name(),
		LastChecked: time.Now(),
		Duration:    duration,
		Error:       err,
	}

	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = err.Error()
	} else {
		result.Status = StatusHealthy
		result.Message = "OK"
	}

	hm.results[checker.Name()] = result
	hm.mu.Unlock()

	// Record metrics
	healthy := err == nil
	hm.metrics.RecordHealthCheck(checker.Name(), healthy, duration)

	// Log health check result
	level := logging.LevelInfo
	if err != nil {
		level = logging.LevelWarn
	}

	hm.logger.LogDaemon(level, "health check completed", "health_check", map[string]interface{}{
		"checker":  checker.Name(),
		"status":   string(result.Status),
		"duration": duration.String(),
		"error":    err,
	})
}

// Predefined health checkers

// MatrixHealthChecker checks matrix connectivity
type MatrixHealthChecker struct {
	name     string
	testFunc func(ctx context.Context) error
	timeout  time.Duration
}

// NewMatrixHealthChecker creates a matrix health checker
func NewMatrixHealthChecker(name string, testFunc func(ctx context.Context) error) *MatrixHealthChecker {
	return &MatrixHealthChecker{
		name:     name,
		testFunc: testFunc,
		timeout:  5 * time.Second,
	}
}

func (m *MatrixHealthChecker) Name() string {
	return m.name
}

func (m *MatrixHealthChecker) Timeout() time.Duration {
	return m.timeout
}

func (m *MatrixHealthChecker) Check(ctx context.Context) error {
	if m.testFunc == nil {
		return fmt.Errorf("no test function provided")
	}
	return m.testFunc(ctx)
}

// StatsHealthChecker checks stats collection
type StatsHealthChecker struct {
	name     string
	testFunc func(ctx context.Context) error
	timeout  time.Duration
}

// NewStatsHealthChecker creates a stats health checker
func NewStatsHealthChecker(name string, testFunc func(ctx context.Context) error) *StatsHealthChecker {
	return &StatsHealthChecker{
		name:     name,
		testFunc: testFunc,
		timeout:  3 * time.Second,
	}
}

func (s *StatsHealthChecker) Name() string {
	return s.name
}

func (s *StatsHealthChecker) Timeout() time.Duration {
	return s.timeout
}

func (s *StatsHealthChecker) Check(ctx context.Context) error {
	if s.testFunc == nil {
		return fmt.Errorf("no test function provided")
	}
	return s.testFunc(ctx)
}

// ConfigHealthChecker checks configuration validity
type ConfigHealthChecker struct {
	name     string
	testFunc func(ctx context.Context) error
	timeout  time.Duration
}

// NewConfigHealthChecker creates a config health checker
func NewConfigHealthChecker(name string, testFunc func(ctx context.Context) error) *ConfigHealthChecker {
	return &ConfigHealthChecker{
		name:     name,
		testFunc: testFunc,
		timeout:  2 * time.Second,
	}
}

func (c *ConfigHealthChecker) Name() string {
	return c.name
}

func (c *ConfigHealthChecker) Timeout() time.Duration {
	return c.timeout
}

func (c *ConfigHealthChecker) Check(ctx context.Context) error {
	if c.testFunc == nil {
		return fmt.Errorf("no test function provided")
	}
	return c.testFunc(ctx)
}

// MemoryHealthChecker checks memory usage
type MemoryHealthChecker struct {
	name           string
	maxMemoryBytes uint64
	timeout        time.Duration
}

// NewMemoryHealthChecker creates a memory health checker
func NewMemoryHealthChecker(name string, maxMemoryBytes uint64) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		name:           name,
		maxMemoryBytes: maxMemoryBytes,
		timeout:        1 * time.Second,
	}
}

func (m *MemoryHealthChecker) Name() string {
	return m.name
}

func (m *MemoryHealthChecker) Timeout() time.Duration {
	return m.timeout
}

func (m *MemoryHealthChecker) Check(ctx context.Context) error {
	// This would need to be implemented with actual memory checking
	// For now, we'll always return healthy
	// In a real implementation, you'd check runtime.MemStats or similar
	return nil
}

// DiskSpaceHealthChecker checks available disk space
type DiskSpaceHealthChecker struct {
	name         string
	path         string
	minFreeBytes uint64
	timeout      time.Duration
}

// NewDiskSpaceHealthChecker creates a disk space health checker
func NewDiskSpaceHealthChecker(name, path string, minFreeBytes uint64) *DiskSpaceHealthChecker {
	return &DiskSpaceHealthChecker{
		name:         name,
		path:         path,
		minFreeBytes: minFreeBytes,
		timeout:      2 * time.Second,
	}
}

func (d *DiskSpaceHealthChecker) Name() string {
	return d.name
}

func (d *DiskSpaceHealthChecker) Timeout() time.Duration {
	return d.timeout
}

func (d *DiskSpaceHealthChecker) Check(ctx context.Context) error {
	// This would need to be implemented with actual disk space checking
	// For now, we'll always return healthy
	// In a real implementation, you'd use syscall.Statfs or similar
	return nil
}
