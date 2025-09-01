package stats

import (
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	interval := 5 * time.Second
	collector := NewCollector(interval)

	if collector == nil {
		t.Fatal("NewCollector() returned nil")
	}

	if collector.collectInterval != interval {
		t.Errorf("NewCollector() interval = %v, want %v", collector.collectInterval, interval)
	}

	if collector.lastDiskStats == nil {
		t.Error("NewCollector() should initialize lastDiskStats map")
	}

	// Verify default thresholds are set
	thresholds := collector.GetThresholds()
	defaultThresholds := DefaultThresholds()

	if thresholds.CPUWarning != defaultThresholds.CPUWarning {
		t.Errorf("NewCollector() CPU warning threshold = %.1f, want %.1f",
			thresholds.CPUWarning, defaultThresholds.CPUWarning)
	}
}

func TestCollectorSetAndGetThresholds(t *testing.T) {
	collector := NewCollector(time.Second)

	customThresholds := Thresholds{
		CPUWarning:     60.0,
		CPUCritical:    85.0,
		MemoryWarning:  75.0,
		MemoryCritical: 90.0,
		DiskWarning:    85.0,
		DiskCritical:   95.0,
	}

	collector.SetThresholds(customThresholds)
	retrievedThresholds := collector.GetThresholds()

	if retrievedThresholds.CPUWarning != customThresholds.CPUWarning {
		t.Errorf("SetThresholds() CPU warning = %.1f, want %.1f",
			retrievedThresholds.CPUWarning, customThresholds.CPUWarning)
	}

	if retrievedThresholds.CPUCritical != customThresholds.CPUCritical {
		t.Errorf("SetThresholds() CPU critical = %.1f, want %.1f",
			retrievedThresholds.CPUCritical, customThresholds.CPUCritical)
	}

	if retrievedThresholds.MemoryWarning != customThresholds.MemoryWarning {
		t.Errorf("SetThresholds() Memory warning = %.1f, want %.1f",
			retrievedThresholds.MemoryWarning, customThresholds.MemoryWarning)
	}

	if retrievedThresholds.MemoryCritical != customThresholds.MemoryCritical {
		t.Errorf("SetThresholds() Memory critical = %.1f, want %.1f",
			retrievedThresholds.MemoryCritical, customThresholds.MemoryCritical)
	}
}

func TestCollectorThreadSafety(t *testing.T) {
	collector := NewCollector(time.Second)

	// Test concurrent access to thresholds
	done := make(chan bool, 2)

	// Goroutine 1: continuously set thresholds
	go func() {
		for i := 0; i < 100; i++ {
			thresholds := Thresholds{
				CPUWarning:     float64(50 + i%20),
				CPUCritical:    float64(70 + i%20),
				MemoryWarning:  float64(60 + i%20),
				MemoryCritical: float64(80 + i%20),
				DiskWarning:    float64(70 + i%20),
				DiskCritical:   float64(90 + i%20),
			}
			collector.SetThresholds(thresholds)
		}

		done <- true
	}()

	// Goroutine 2: continuously get thresholds
	go func() {
		for i := 0; i < 100; i++ {
			thresholds := collector.GetThresholds()
			// Basic sanity check
			if thresholds.CPUWarning >= thresholds.CPUCritical {
				t.Errorf("CPU warning threshold should be less than critical: %.1f >= %.1f",
					thresholds.CPUWarning, thresholds.CPUCritical)
			}
		}

		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

func TestCollectorGetLastStats(t *testing.T) {
	collector := NewCollector(time.Second)

	// Initially should return nil
	stats := collector.GetLastStats()
	if stats != nil {
		t.Error("GetLastStats() should return nil when no stats have been collected")
	}

	// After collecting stats, should return the stats
	_, err := collector.CollectSystemStats()
	if err != nil {
		t.Skipf("Skipping test due to system stats collection error: %v", err)
	}

	stats = collector.GetLastStats()
	if stats == nil {
		t.Error("GetLastStats() should return stats after collection")

		return
	}

	// Timestamp should be recent
	if time.Since(stats.Timestamp) > 5*time.Second {
		t.Errorf("GetLastStats() timestamp is too old: %v", stats.Timestamp)
	}
}

func TestCollectorDetermineStatus(t *testing.T) {
	collector := NewCollector(time.Second)

	// Set custom thresholds for testing
	thresholds := Thresholds{
		CPUWarning:     50.0,
		CPUCritical:    80.0,
		MemoryWarning:  60.0,
		MemoryCritical: 85.0,
		DiskWarning:    70.0,
		DiskCritical:   90.0,
	}
	collector.SetThresholds(thresholds)

	tests := []struct {
		summary        *StatsSummary
		name           string
		expectedStatus SystemStatus
	}{
		{
			name: "normal status",
			summary: &StatsSummary{
				CPUUsage:    30.0,
				MemoryUsage: 40.0,
			},
			expectedStatus: StatusNormal,
		},
		{
			name: "warning status - CPU",
			summary: &StatsSummary{
				CPUUsage:    60.0,
				MemoryUsage: 40.0,
			},
			expectedStatus: StatusWarning,
		},
		{
			name: "warning status - Memory",
			summary: &StatsSummary{
				CPUUsage:    30.0,
				MemoryUsage: 70.0,
			},
			expectedStatus: StatusWarning,
		},
		{
			name: "critical status - CPU",
			summary: &StatsSummary{
				CPUUsage:    85.0,
				MemoryUsage: 40.0,
			},
			expectedStatus: StatusCritical,
		},
		{
			name: "critical status - Memory",
			summary: &StatsSummary{
				CPUUsage:    30.0,
				MemoryUsage: 90.0,
			},
			expectedStatus: StatusCritical,
		},
		{
			name: "critical status - both high",
			summary: &StatsSummary{
				CPUUsage:    85.0,
				MemoryUsage: 90.0,
			},
			expectedStatus: StatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := collector.determineStatus(tt.summary)
			if status != tt.expectedStatus {
				t.Errorf("determineStatus() = %v, want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestCollectorGetSummaryBasic(t *testing.T) {
	collector := NewCollector(time.Second)

	summary, err := collector.GetSummary()
	if err != nil {
		t.Skipf("Skipping test due to system stats collection error: %v", err)
	}

	if summary == nil {
		t.Fatal("GetSummary() returned nil summary")
	}

	// Basic sanity checks
	if summary.CPUUsage < 0 || summary.CPUUsage > 100 {
		t.Errorf("GetSummary() CPU usage = %.1f, should be 0-100", summary.CPUUsage)
	}

	if summary.MemoryUsage < 0 || summary.MemoryUsage > 100 {
		t.Errorf("GetSummary() Memory usage = %.1f, should be 0-100", summary.MemoryUsage)
	}

	if summary.DiskActivity < 0 {
		t.Errorf("GetSummary() Disk activity = %.1f, should be >= 0", summary.DiskActivity)
	}

	if summary.NetworkActivity < 0 {
		t.Errorf("GetSummary() Network activity = %.1f, should be >= 0", summary.NetworkActivity)
	}

	// Status should be valid
	validStatuses := []SystemStatus{StatusNormal, StatusWarning, StatusCritical}
	statusValid := false

	for _, validStatus := range validStatuses {
		if summary.Status == validStatus {
			statusValid = true

			break
		}
	}

	if !statusValid {
		t.Errorf("GetSummary() Status = %v, should be one of %v", summary.Status, validStatuses)
	}

	// Timestamp should be recent
	if time.Since(summary.Timestamp) > 5*time.Second {
		t.Errorf("GetSummary() timestamp is too old: %v", summary.Timestamp)
	}
}

// Integration test that attempts to collect real system stats
// This test may be skipped on systems where gopsutil doesn't work properly.
func TestCollectorCollectCPUStats(t *testing.T) {
	collector := NewCollector(time.Second)

	cpuStats, err := collector.CollectCPUStats()
	if err != nil {
		t.Skipf("Skipping CPU stats test due to collection error: %v", err)
	}

	// Basic validation
	if cpuStats.PhysicalCores <= 0 {
		t.Errorf("CollectCPUStats() PhysicalCores = %d, should be > 0", cpuStats.PhysicalCores)
	}

	if cpuStats.LogicalCores <= 0 {
		t.Errorf("CollectCPUStats() LogicalCores = %d, should be > 0", cpuStats.LogicalCores)
	}

	if cpuStats.LogicalCores < cpuStats.PhysicalCores {
		t.Errorf("CollectCPUStats() LogicalCores (%d) should be >= PhysicalCores (%d)",
			cpuStats.LogicalCores, cpuStats.PhysicalCores)
	}

	if cpuStats.UsagePercent < 0 || cpuStats.UsagePercent > 100 {
		t.Errorf("CollectCPUStats() UsagePercent = %.1f, should be 0-100", cpuStats.UsagePercent)
	}

	// Per-core percentages should be reasonable if present
	for i, percent := range cpuStats.PerCorePercent {
		if percent < 0 || percent > 100 {
			t.Errorf("CollectCPUStats() PerCorePercent[%d] = %.1f, should be 0-100", i, percent)
		}
	}
}

func TestCollectorCollectMemoryStats(t *testing.T) {
	collector := NewCollector(time.Second)

	memStats, err := collector.CollectMemoryStats()
	if err != nil {
		t.Skipf("Skipping memory stats test due to collection error: %v", err)
	}

	// Basic validation
	if memStats.Total == 0 {
		t.Errorf("CollectMemoryStats() Total = %d, should be > 0", memStats.Total)
	}

	if memStats.Used > memStats.Total {
		t.Errorf("CollectMemoryStats() Used (%d) should be <= Total (%d)", memStats.Used, memStats.Total)
	}

	if memStats.Available > memStats.Total {
		t.Errorf("CollectMemoryStats() Available (%d) should be <= Total (%d)", memStats.Available, memStats.Total)
	}

	if memStats.UsedPercent < 0 || memStats.UsedPercent > 100 {
		t.Errorf("CollectMemoryStats() UsedPercent = %.1f, should be 0-100", memStats.UsedPercent)
	}

	// Swap stats validation (if swap is available)
	if memStats.SwapTotal > 0 {
		if memStats.SwapUsed > memStats.SwapTotal {
			t.Errorf("CollectMemoryStats() SwapUsed (%d) should be <= SwapTotal (%d)",
				memStats.SwapUsed, memStats.SwapTotal)
		}

		if memStats.SwapPercent < 0 || memStats.SwapPercent > 100 {
			t.Errorf("CollectMemoryStats() SwapPercent = %.1f, should be 0-100", memStats.SwapPercent)
		}
	}
}

func TestCollectorCollectDiskStats(t *testing.T) {
	collector := NewCollector(time.Second)

	diskStats, err := collector.CollectDiskStats()
	if err != nil {
		t.Logf("Disk stats collection warning: %v", err)
		// Don't skip the test entirely, as this might just be a warning
	}

	// Partitions validation
	for i, partition := range diskStats.Partitions {
		if partition.Device == "" {
			t.Errorf("CollectDiskStats() Partitions[%d].Device is empty", i)
		}

		if partition.Mountpoint == "" {
			t.Errorf("CollectDiskStats() Partitions[%d].Mountpoint is empty", i)
		}

		if partition.Total > 0 {
			if partition.Used > partition.Total {
				t.Errorf("CollectDiskStats() Partition[%d] Used (%d) should be <= Total (%d)",
					i, partition.Used, partition.Total)
			}

			if partition.UsedPercent < 0 || partition.UsedPercent > 100 {
				t.Errorf("CollectDiskStats() Partition[%d] UsedPercent = %.1f, should be 0-100",
					i, partition.UsedPercent)
			}
		}
	}

	// IO counters validation
	for device := range diskStats.IOCounters {
		if device == "" {
			t.Error("CollectDiskStats() IOCounters has empty device name")
		}
	}

	// Activity rate should be non-negative
	if diskStats.ActivityRate < 0 {
		t.Errorf("CollectDiskStats() ActivityRate = %.1f, should be >= 0", diskStats.ActivityRate)
	}
}

func TestCollectorCollectNetworkStats(t *testing.T) {
	collector := NewCollector(time.Second)

	netStats, err := collector.CollectNetworkStats()
	if err != nil {
		t.Skipf("Skipping network stats test due to collection error: %v", err)
	}

	// Basic validation - all counters should be non-negative
	// uint64 fields are always non-negative, no need to check

	if netStats.ActivityRate < 0 {
		t.Errorf("CollectNetworkStats() ActivityRate = %.1f, should be >= 0", netStats.ActivityRate)
	}
}

func TestCollectorCollectSystemStatsIntegration(t *testing.T) {
	collector := NewCollector(time.Second)

	stats, err := collector.CollectSystemStats()
	if err != nil {
		t.Skipf("Skipping system stats integration test due to collection error: %v", err)
	}

	if stats == nil {
		t.Fatal("CollectSystemStats() returned nil stats")
	}

	// Verify all major components are populated
	if stats.CPU.PhysicalCores <= 0 {
		t.Error("CollectSystemStats() CPU stats not properly populated")
	}

	if stats.Memory.Total == 0 {
		t.Error("CollectSystemStats() Memory stats not properly populated")
	}

	// Timestamp should be recent
	if time.Since(stats.Timestamp) > 5*time.Second {
		t.Errorf("CollectSystemStats() timestamp is too old: %v", stats.Timestamp)
	}

	// Verify stats are stored as last stats
	lastStats := collector.GetLastStats()
	if lastStats == nil {
		t.Error("CollectSystemStats() should update last stats")

		return
	}

	if !lastStats.Timestamp.Equal(stats.Timestamp) {
		t.Error("CollectSystemStats() last stats timestamp mismatch")
	}
}

// Test activity rate calculation over multiple collections.
func TestCollectorActivityRateCalculation(t *testing.T) {
	collector := NewCollector(100 * time.Millisecond) // Short interval for testing

	// First collection to establish baseline
	_, err := collector.CollectDiskStats()
	if err != nil {
		t.Skipf("Skipping activity rate test due to disk stats error: %v", err)
	}

	// Wait a bit and collect again
	time.Sleep(150 * time.Millisecond)

	diskStats, err := collector.CollectDiskStats()
	if err != nil {
		t.Skipf("Skipping activity rate test due to disk stats error: %v", err)
	}

	// Activity rate should be calculated (may be 0 if no disk activity)
	if diskStats.ActivityRate < 0 {
		t.Errorf("Activity rate should be non-negative, got %.1f", diskStats.ActivityRate)
	}
}

// Benchmark tests.
func BenchmarkCollectorGetThresholds(b *testing.B) {
	collector := NewCollector(time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		collector.GetThresholds()
	}
}

func BenchmarkCollectorSetThresholds(b *testing.B) {
	collector := NewCollector(time.Second)
	thresholds := DefaultThresholds()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		collector.SetThresholds(thresholds)
	}
}

func BenchmarkCollectorDetermineStatus(b *testing.B) {
	collector := NewCollector(time.Second)
	summary := &StatsSummary{
		CPUUsage:    75.0,
		MemoryUsage: 60.0,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		collector.determineStatus(summary)
	}
}

// Test that would collect real system stats (may be slow).
func BenchmarkCollectorCollectSystemStats(b *testing.B) {
	collector := NewCollector(time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := collector.CollectSystemStats()
		if err != nil {
			b.Skip("System stats collection failed:", err)
		}
	}
}

// Test concurrent access patterns.
func TestCollectorConcurrentAccess(t *testing.T) {
	collector := NewCollector(time.Second)

	// Run multiple goroutines that access collector concurrently
	done := make(chan bool, 3)

	// Goroutine 1: collect stats
	go func() {
		for i := 0; i < 10; i++ {
			collector.CollectSystemStats()
			time.Sleep(10 * time.Millisecond)
		}

		done <- true
	}()

	// Goroutine 2: get summaries
	go func() {
		for i := 0; i < 10; i++ {
			collector.GetSummary()
			time.Sleep(15 * time.Millisecond)
		}

		done <- true
	}()

	// Goroutine 3: access thresholds
	go func() {
		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				thresholds := collector.GetThresholds()
				thresholds.CPUWarning += 1.0
				collector.SetThresholds(thresholds)
			} else {
				collector.GetThresholds()
			}

			time.Sleep(5 * time.Millisecond)
		}

		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify collector is still functional
	summary, err := collector.GetSummary()
	if err != nil {
		t.Errorf("Collector should be functional after concurrent access: %v", err)
	}

	if summary == nil {
		t.Error("GetSummary() should return valid summary after concurrent access")
	}
}
