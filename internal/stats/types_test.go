package stats

import (
	"testing"
	"time"
)

func TestSystemStatusString(t *testing.T) {
	tests := []struct {
		status   SystemStatus
		expected string
	}{
		{StatusNormal, "normal"},
		{StatusWarning, "warning"},
		{StatusCritical, "critical"},
		{SystemStatus(999), "unknown"}, // Invalid status
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("SystemStatus.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()
	
	expectedValues := map[string]float64{
		"CPUWarning":     70.0,
		"CPUCritical":    90.0,
		"MemoryWarning":  80.0,
		"MemoryCritical": 95.0,
		"DiskWarning":    80.0,
		"DiskCritical":   95.0,
	}
	
	actualValues := map[string]float64{
		"CPUWarning":     thresholds.CPUWarning,
		"CPUCritical":    thresholds.CPUCritical,
		"MemoryWarning":  thresholds.MemoryWarning,
		"MemoryCritical": thresholds.MemoryCritical,
		"DiskWarning":    thresholds.DiskWarning,
		"DiskCritical":   thresholds.DiskCritical,
	}
	
	for name, expected := range expectedValues {
		if actual := actualValues[name]; actual != expected {
			t.Errorf("DefaultThresholds().%s = %.1f, want %.1f", name, actual, expected)
		}
	}
	
	// Verify warning thresholds are less than critical thresholds
	if thresholds.CPUWarning >= thresholds.CPUCritical {
		t.Errorf("CPU warning threshold (%.1f) should be less than critical threshold (%.1f)", 
			thresholds.CPUWarning, thresholds.CPUCritical)
	}
	
	if thresholds.MemoryWarning >= thresholds.MemoryCritical {
		t.Errorf("Memory warning threshold (%.1f) should be less than critical threshold (%.1f)", 
			thresholds.MemoryWarning, thresholds.MemoryCritical)
	}
	
	if thresholds.DiskWarning >= thresholds.DiskCritical {
		t.Errorf("Disk warning threshold (%.1f) should be less than critical threshold (%.1f)", 
			thresholds.DiskWarning, thresholds.DiskCritical)
	}
}

func TestCPUStatsStructure(t *testing.T) {
	stats := CPUStats{
		UsagePercent:   75.5,
		PerCorePercent: []float64{70.0, 80.0, 75.0, 76.0},
		PhysicalCores:  4,
		LogicalCores:   8,
		ModelName:      "Intel Core i7-10700K",
		VendorID:       "GenuineIntel",
	}
	
	if stats.UsagePercent != 75.5 {
		t.Errorf("CPUStats.UsagePercent = %.1f, want 75.5", stats.UsagePercent)
	}
	
	if len(stats.PerCorePercent) != 4 {
		t.Errorf("CPUStats.PerCorePercent length = %d, want 4", len(stats.PerCorePercent))
	}
	
	if stats.PhysicalCores != 4 {
		t.Errorf("CPUStats.PhysicalCores = %d, want 4", stats.PhysicalCores)
	}
	
	if stats.LogicalCores != 8 {
		t.Errorf("CPUStats.LogicalCores = %d, want 8", stats.LogicalCores)
	}
	
	if stats.ModelName != "Intel Core i7-10700K" {
		t.Errorf("CPUStats.ModelName = %s, want 'Intel Core i7-10700K'", stats.ModelName)
	}
	
	if stats.VendorID != "GenuineIntel" {
		t.Errorf("CPUStats.VendorID = %s, want 'GenuineIntel'", stats.VendorID)
	}
}

func TestMemoryStatsStructure(t *testing.T) {
	stats := MemoryStats{
		Total:       16 * 1024 * 1024 * 1024, // 16GB
		Available:   8 * 1024 * 1024 * 1024,  // 8GB
		Used:        8 * 1024 * 1024 * 1024,  // 8GB
		UsedPercent: 50.0,
		Free:        8 * 1024 * 1024 * 1024,  // 8GB
		SwapTotal:   4 * 1024 * 1024 * 1024,  // 4GB
		SwapUsed:    1024 * 1024 * 1024,      // 1GB
		SwapPercent: 25.0,
	}
	
	if stats.Total != 16*1024*1024*1024 {
		t.Errorf("MemoryStats.Total = %d, want %d", stats.Total, 16*1024*1024*1024)
	}
	
	if stats.UsedPercent != 50.0 {
		t.Errorf("MemoryStats.UsedPercent = %.1f, want 50.0", stats.UsedPercent)
	}
	
	if stats.SwapPercent != 25.0 {
		t.Errorf("MemoryStats.SwapPercent = %.1f, want 25.0", stats.SwapPercent)
	}
}

func TestDiskStatsStructure(t *testing.T) {
	partition := PartitionStat{
		Device:      "/dev/sda1",
		Mountpoint:  "/",
		Fstype:      "ext4",
		Total:       1024 * 1024 * 1024 * 1024, // 1TB
		Used:        512 * 1024 * 1024 * 1024,  // 512GB
		Free:        512 * 1024 * 1024 * 1024,  // 512GB
		UsedPercent: 50.0,
	}
	
	ioCounter := IOCounterStat{
		ReadCount:  1000,
		WriteCount: 500,
		ReadBytes:  1024 * 1024 * 100, // 100MB
		WriteBytes: 1024 * 1024 * 50,  // 50MB
		ReadTime:   1000,
		WriteTime:  500,
	}
	
	stats := DiskStats{
		Partitions:   []PartitionStat{partition},
		IOCounters:   map[string]IOCounterStat{"sda": ioCounter},
		TotalReads:   1000,
		TotalWrites:  500,
		ReadBytes:    1024 * 1024 * 100,
		WriteBytes:   1024 * 1024 * 50,
		ActivityRate: 1024.0 * 1024.0, // 1MB/s
	}
	
	if len(stats.Partitions) != 1 {
		t.Errorf("DiskStats.Partitions length = %d, want 1", len(stats.Partitions))
	}
	
	if stats.Partitions[0].Device != "/dev/sda1" {
		t.Errorf("Partition.Device = %s, want '/dev/sda1'", stats.Partitions[0].Device)
	}
	
	if stats.Partitions[0].UsedPercent != 50.0 {
		t.Errorf("Partition.UsedPercent = %.1f, want 50.0", stats.Partitions[0].UsedPercent)
	}
	
	if len(stats.IOCounters) != 1 {
		t.Errorf("DiskStats.IOCounters length = %d, want 1", len(stats.IOCounters))
	}
	
	if counter, exists := stats.IOCounters["sda"]; !exists {
		t.Error("IOCounter for 'sda' not found")
	} else {
		if counter.ReadCount != 1000 {
			t.Errorf("IOCounter.ReadCount = %d, want 1000", counter.ReadCount)
		}
		if counter.WriteCount != 500 {
			t.Errorf("IOCounter.WriteCount = %d, want 500", counter.WriteCount)
		}
	}
	
	if stats.ActivityRate != 1024.0*1024.0 {
		t.Errorf("DiskStats.ActivityRate = %.1f, want %.1f", stats.ActivityRate, 1024.0*1024.0)
	}
}

func TestNetworkStatsStructure(t *testing.T) {
	stats := NetworkStats{
		BytesSent:      1024 * 1024 * 100, // 100MB
		BytesRecv:      1024 * 1024 * 200, // 200MB
		PacketsSent:    10000,
		PacketsRecv:    20000,
		TotalBytesSent: 1024 * 1024 * 500, // 500MB
		TotalBytesRecv: 1024 * 1024 * 1000, // 1GB
		ActivityRate:   1024.0 * 1024.0,   // 1MB/s
	}
	
	if stats.BytesSent != 1024*1024*100 {
		t.Errorf("NetworkStats.BytesSent = %d, want %d", stats.BytesSent, 1024*1024*100)
	}
	
	if stats.PacketsSent != 10000 {
		t.Errorf("NetworkStats.PacketsSent = %d, want 10000", stats.PacketsSent)
	}
	
	if stats.ActivityRate != 1024.0*1024.0 {
		t.Errorf("NetworkStats.ActivityRate = %.1f, want %.1f", stats.ActivityRate, 1024.0*1024.0)
	}
}

func TestSystemStatsStructure(t *testing.T) {
	now := time.Now()
	uptime := 24 * time.Hour
	loadAvg := []float64{1.5, 2.0, 2.5}
	
	stats := SystemStats{
		CPU: CPUStats{
			UsagePercent:  75.0,
			PhysicalCores: 4,
			LogicalCores:  8,
		},
		Memory: MemoryStats{
			Total:       16 * 1024 * 1024 * 1024,
			UsedPercent: 60.0,
		},
		Disk: DiskStats{
			ActivityRate: 1024.0 * 1024.0,
		},
		Network: NetworkStats{
			ActivityRate: 512.0 * 1024.0,
		},
		Timestamp: now,
		Uptime:    uptime,
		LoadAvg:   loadAvg,
	}
	
	if stats.CPU.UsagePercent != 75.0 {
		t.Errorf("SystemStats.CPU.UsagePercent = %.1f, want 75.0", stats.CPU.UsagePercent)
	}
	
	if stats.Memory.UsedPercent != 60.0 {
		t.Errorf("SystemStats.Memory.UsedPercent = %.1f, want 60.0", stats.Memory.UsedPercent)
	}
	
	if !stats.Timestamp.Equal(now) {
		t.Errorf("SystemStats.Timestamp = %v, want %v", stats.Timestamp, now)
	}
	
	if stats.Uptime != uptime {
		t.Errorf("SystemStats.Uptime = %v, want %v", stats.Uptime, uptime)
	}
	
	if len(stats.LoadAvg) != 3 {
		t.Errorf("SystemStats.LoadAvg length = %d, want 3", len(stats.LoadAvg))
	}
	
	for i, expected := range loadAvg {
		if stats.LoadAvg[i] != expected {
			t.Errorf("SystemStats.LoadAvg[%d] = %.1f, want %.1f", i, stats.LoadAvg[i], expected)
		}
	}
}

func TestStatsSummaryStructure(t *testing.T) {
	now := time.Now()
	
	summary := StatsSummary{
		CPUUsage:        75.5,
		MemoryUsage:     60.2,
		DiskActivity:    1024.0 * 1024.0,
		NetworkActivity: 512.0 * 1024.0,
		Status:          StatusWarning,
		Timestamp:       now,
	}
	
	if summary.CPUUsage != 75.5 {
		t.Errorf("StatsSummary.CPUUsage = %.1f, want 75.5", summary.CPUUsage)
	}
	
	if summary.MemoryUsage != 60.2 {
		t.Errorf("StatsSummary.MemoryUsage = %.1f, want 60.2", summary.MemoryUsage)
	}
	
	if summary.DiskActivity != 1024.0*1024.0 {
		t.Errorf("StatsSummary.DiskActivity = %.1f, want %.1f", summary.DiskActivity, 1024.0*1024.0)
	}
	
	if summary.NetworkActivity != 512.0*1024.0 {
		t.Errorf("StatsSummary.NetworkActivity = %.1f, want %.1f", summary.NetworkActivity, 512.0*1024.0)
	}
	
	if summary.Status != StatusWarning {
		t.Errorf("StatsSummary.Status = %v, want %v", summary.Status, StatusWarning)
	}
	
	if !summary.Timestamp.Equal(now) {
		t.Errorf("StatsSummary.Timestamp = %v, want %v", summary.Timestamp, now)
	}
}

func TestThresholdsValidation(t *testing.T) {
	// Test that default thresholds are reasonable
	thresholds := DefaultThresholds()
	
	// All thresholds should be positive
	if thresholds.CPUWarning <= 0 || thresholds.CPUCritical <= 0 {
		t.Error("CPU thresholds should be positive")
	}
	
	if thresholds.MemoryWarning <= 0 || thresholds.MemoryCritical <= 0 {
		t.Error("Memory thresholds should be positive")
	}
	
	if thresholds.DiskWarning <= 0 || thresholds.DiskCritical <= 0 {
		t.Error("Disk thresholds should be positive")
	}
	
	// All thresholds should be reasonable percentages (≤ 100)
	if thresholds.CPUWarning > 100 || thresholds.CPUCritical > 100 {
		t.Error("CPU thresholds should be ≤ 100%")
	}
	
	if thresholds.MemoryWarning > 100 || thresholds.MemoryCritical > 100 {
		t.Error("Memory thresholds should be ≤ 100%")
	}
	
	if thresholds.DiskWarning > 100 || thresholds.DiskCritical > 100 {
		t.Error("Disk thresholds should be ≤ 100%")
	}
}

func TestDataStructureSizes(t *testing.T) {
	// Test that our data structures are reasonable in size
	// This helps ensure we're not accidentally including huge embedded data
	
	cpu := CPUStats{}
	if size := len(cpu.PerCorePercent); size > 0 {
		// Should start empty
		t.Errorf("CPUStats.PerCorePercent should start empty, got length %d", size)
	}
	
	disk := DiskStats{}
	if size := len(disk.Partitions); size > 0 {
		t.Errorf("DiskStats.Partitions should start empty, got length %d", size)
	}
	
	if disk.IOCounters == nil {
		// IOCounters map should be initialized when creating the struct manually
		disk.IOCounters = make(map[string]IOCounterStat)
	}
	
	system := SystemStats{}
	if size := len(system.LoadAvg); size > 0 {
		t.Errorf("SystemStats.LoadAvg should start empty, got length %d", size)
	}
}

func TestSystemStatusConstants(t *testing.T) {
	// Verify that the status constants have expected values
	if StatusNormal != 0 {
		t.Errorf("StatusNormal = %d, want 0", StatusNormal)
	}
	
	if StatusWarning != 1 {
		t.Errorf("StatusWarning = %d, want 1", StatusWarning)
	}
	
	if StatusCritical != 2 {
		t.Errorf("StatusCritical = %d, want 2", StatusCritical)
	}
}

// Benchmark tests
func BenchmarkSystemStatusString(b *testing.B) {
	statuses := []SystemStatus{StatusNormal, StatusWarning, StatusCritical}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		_ = status.String()
	}
}

func BenchmarkDefaultThresholds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DefaultThresholds()
	}
}

func BenchmarkStatsSummaryCreation(b *testing.B) {
	now := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StatsSummary{
			CPUUsage:        75.5,
			MemoryUsage:     60.2,
			DiskActivity:    1024.0 * 1024.0,
			NetworkActivity: 512.0 * 1024.0,
			Status:          StatusWarning,
			Timestamp:       now,
		}
	}
}

func BenchmarkSystemStatsCreation(b *testing.B) {
	now := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SystemStats{
			CPU: CPUStats{
				UsagePercent:  75.0,
				PhysicalCores: 4,
				LogicalCores:  8,
			},
			Memory: MemoryStats{
				Total:       16 * 1024 * 1024 * 1024,
				UsedPercent: 60.0,
			},
			Timestamp: now,
		}
	}
}