// Package stats provides system metrics collection functionality using gopsutil.
// It collects CPU, memory, disk, and network statistics with configurable thresholds
// for determining system health status.
package stats

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Collector gathers system statistics using gopsutil with rate limiting and threshold management.
type Collector struct {
	lastStats       *SystemStats
	lastDiskStats   map[string]disk.IOCountersStat
	lastNetStats    []net.IOCountersStat
	thresholds      Thresholds
	collectInterval time.Duration
	mu              sync.RWMutex
}

// NewCollector creates a new Collector with the specified collection interval and default thresholds.
func NewCollector(interval time.Duration) *Collector {
	return &Collector{
		collectInterval: interval,
		thresholds:      DefaultThresholds(),
		lastDiskStats:   make(map[string]disk.IOCountersStat),
	}
}

// SetThresholds updates the system resource usage thresholds for status determination.
func (c *Collector) SetThresholds(t Thresholds) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.thresholds = t
}

// GetThresholds returns the current system resource usage thresholds.
func (c *Collector) GetThresholds() Thresholds {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.thresholds
}

// CollectCPUStats gathers detailed CPU statistics including usage percentages, core counts, and processor information.
func (c *Collector) CollectCPUStats() (CPUStats, error) {
	var stats CPUStats

	physicalCount, err := cpu.Counts(false)
	if err != nil {
		return stats, fmt.Errorf("failed to get physical CPU count: %w", err)
	}

	stats.PhysicalCores = physicalCount

	logicalCount, err := cpu.Counts(true)
	if err != nil {
		return stats, fmt.Errorf("failed to get logical CPU count: %w", err)
	}

	stats.LogicalCores = logicalCount

	totalPercent, err := cpu.Percent(0, false)
	if err != nil {
		return stats, fmt.Errorf("failed to get total CPU usage: %w", err)
	}

	if len(totalPercent) > 0 {
		stats.UsagePercent = totalPercent[0]
	}

	perCorePercent, err := cpu.Percent(0, true)
	if err != nil {
		log.Printf("Warning: failed to get per-core CPU usage: %v", err)
	} else {
		stats.PerCorePercent = perCorePercent
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Printf("Warning: failed to get CPU info: %v", err)
	} else if len(cpuInfo) > 0 {
		stats.ModelName = cpuInfo[0].ModelName
		stats.VendorID = cpuInfo[0].VendorID
	}

	return stats, nil
}

// CollectMemoryStats gathers memory and swap usage statistics.
func (c *Collector) CollectMemoryStats() (MemoryStats, error) {
	var stats MemoryStats

	vmem, err := mem.VirtualMemory()
	if err != nil {
		return stats, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	stats.Total = vmem.Total
	stats.Available = vmem.Available
	stats.Used = vmem.Used
	stats.UsedPercent = vmem.UsedPercent
	stats.Free = vmem.Free

	swap, err := mem.SwapMemory()
	if err != nil {
		log.Printf("Warning: failed to get swap memory stats: %v", err)
	} else {
		stats.SwapTotal = swap.Total
		stats.SwapUsed = swap.Used
		stats.SwapPercent = swap.UsedPercent
	}

	return stats, nil
}

// CollectDiskStats gathers disk I/O statistics including partition information and activity rates.
func (c *Collector) CollectDiskStats() (DiskStats, error) {
	var stats DiskStats

	partitions, err := disk.Partitions(false)
	if err != nil {
		return stats, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	for _, partition := range partitions {
		usage, usageErr := disk.Usage(partition.Mountpoint)
		if usageErr != nil {
			log.Printf("Warning: failed to get usage for partition %s: %v", partition.Device, usageErr)

			continue
		}

		partStat := PartitionStat{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Fstype:      partition.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		}
		stats.Partitions = append(stats.Partitions, partStat)
	}

	ioCounters, err := disk.IOCounters()
	if err != nil {
		log.Printf("Warning: failed to get disk I/O counters: %v", err)
	} else {
		stats.IOCounters = make(map[string]IOCounterStat)
		for device, counter := range ioCounters {
			stats.IOCounters[device] = IOCounterStat{
				ReadCount:  counter.ReadCount,
				WriteCount: counter.WriteCount,
				ReadBytes:  counter.ReadBytes,
				WriteBytes: counter.WriteBytes,
				ReadTime:   counter.ReadTime,
				WriteTime:  counter.WriteTime,
			}
			stats.TotalReads += counter.ReadCount
			stats.TotalWrites += counter.WriteCount
			stats.ReadBytes += counter.ReadBytes
			stats.WriteBytes += counter.WriteBytes
		}

		c.mu.Lock()

		if c.lastDiskStats != nil {
			var totalActivity uint64

			for device, current := range ioCounters {
				if last, exists := c.lastDiskStats[device]; exists {
					readDiff := current.ReadBytes - last.ReadBytes
					writeDiff := current.WriteBytes - last.WriteBytes
					totalActivity += readDiff + writeDiff
				}
			}

			stats.ActivityRate = float64(totalActivity) / c.collectInterval.Seconds()
		}

		c.lastDiskStats = ioCounters
		c.mu.Unlock()
	}

	return stats, nil
}

// CollectNetworkStats gathers network interface statistics including bytes transferred and activity rates.
func (c *Collector) CollectNetworkStats() (NetworkStats, error) {
	var stats NetworkStats

	netIO, err := net.IOCounters(false)
	if err != nil {
		return stats, fmt.Errorf("failed to get network I/O counters: %w", err)
	}

	if len(netIO) > 0 {
		stats.BytesSent = netIO[0].BytesSent
		stats.BytesRecv = netIO[0].BytesRecv
		stats.PacketsSent = netIO[0].PacketsSent
		stats.PacketsRecv = netIO[0].PacketsRecv
		stats.TotalBytesSent = netIO[0].BytesSent
		stats.TotalBytesRecv = netIO[0].BytesRecv

		c.mu.Lock()

		if len(c.lastNetStats) > 0 {
			sentDiff := netIO[0].BytesSent - c.lastNetStats[0].BytesSent
			recvDiff := netIO[0].BytesRecv - c.lastNetStats[0].BytesRecv
			stats.ActivityRate = float64(sentDiff+recvDiff) / c.collectInterval.Seconds()
		}

		c.lastNetStats = netIO
		c.mu.Unlock()
	}

	return stats, nil
}

// CollectSystemStats gathers comprehensive system statistics including CPU, memory, disk,
// network, and uptime information.
func (c *Collector) CollectSystemStats() (*SystemStats, error) {
	stats := &SystemStats{
		Timestamp: time.Now(),
	}

	cpuStats, err := c.CollectCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to collect CPU stats: %w", err)
	}

	stats.CPU = cpuStats

	memStats, err := c.CollectMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory stats: %w", err)
	}

	stats.Memory = memStats

	diskStats, err := c.CollectDiskStats()
	if err != nil {
		log.Printf("Warning: failed to collect disk stats: %v", err)
	}

	stats.Disk = diskStats

	netStats, err := c.CollectNetworkStats()
	if err != nil {
		log.Printf("Warning: failed to collect network stats: %v", err)
	}

	stats.Network = netStats

	uptime, err := host.Uptime()
	if err != nil {
		log.Printf("Warning: failed to get uptime: %v", err)
	} else {
		// #nosec G115 - uptime conversion is safe within reasonable system limits
		stats.Uptime = time.Duration(uptime) * time.Second
	}

	loadAvg, err := load.Avg()
	if err != nil {
		log.Printf("Warning: failed to get load average: %v", err)
	} else {
		stats.LoadAvg = []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
	}

	c.mu.Lock()
	c.lastStats = stats
	c.mu.Unlock()

	return stats, nil
}

// GetLastStats returns the most recently collected system statistics.
func (c *Collector) GetLastStats() *SystemStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastStats
}

// GetSummary collects current system statistics and returns a summarized view with overall system status.
func (c *Collector) GetSummary() (*StatsSummary, error) {
	stats, err := c.CollectSystemStats()
	if err != nil {
		return nil, err
	}

	summary := &StatsSummary{
		CPUUsage:        stats.CPU.UsagePercent,
		MemoryUsage:     stats.Memory.UsedPercent,
		DiskActivity:    stats.Disk.ActivityRate,
		NetworkActivity: stats.Network.ActivityRate,
		Timestamp:       stats.Timestamp,
	}

	summary.Status = c.determineStatus(summary)

	return summary, nil
}

func (c *Collector) determineStatus(summary *StatsSummary) SystemStatus {
	thresholds := c.GetThresholds()

	if summary.CPUUsage >= thresholds.CPUCritical ||
		summary.MemoryUsage >= thresholds.MemoryCritical {
		return StatusCritical
	}

	if summary.CPUUsage >= thresholds.CPUWarning ||
		summary.MemoryUsage >= thresholds.MemoryWarning {
		return StatusWarning
	}

	return StatusNormal
}
