package stats

import "time"

// CPUStats contains detailed CPU statistics including model information, usage percentages, and core counts.
type CPUStats struct {
	ModelName      string
	VendorID       string
	PerCorePercent []float64
	UsagePercent   float64
	PhysicalCores  int
	LogicalCores   int
}

// MemoryStats contains memory and swap usage statistics including totals, usage percentages, and available memory.
type MemoryStats struct {
	Total       uint64
	Available   uint64
	Used        uint64
	UsedPercent float64
	Free        uint64
	SwapTotal   uint64
	SwapUsed    uint64
	SwapPercent float64
}

// DiskStats contains disk I/O statistics including partition information, read/write counters, and activity rates.
type DiskStats struct {
	IOCounters   map[string]IOCounterStat
	Partitions   []PartitionStat
	TotalReads   uint64
	TotalWrites  uint64
	ReadBytes    uint64
	WriteBytes   uint64
	ActivityRate float64
}

// PartitionStat contains disk partition statistics including device information, filesystem type, and space usage.
type PartitionStat struct {
	Device      string
	Mountpoint  string
	Fstype      string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// IOCounterStat contains I/O operation counters including read/write counts, bytes transferred, and timing information.
type IOCounterStat struct {
	ReadCount  uint64
	WriteCount uint64
	ReadBytes  uint64
	WriteBytes uint64
	ReadTime   uint64
	WriteTime  uint64
}

// NetworkStats contains network interface statistics including bytes and packets sent/received, and activity rates.
type NetworkStats struct {
	BytesSent      uint64
	BytesRecv      uint64
	PacketsSent    uint64
	PacketsRecv    uint64
	TotalBytesSent uint64
	TotalBytesRecv uint64
	ActivityRate   float64
}

// SystemStats contains comprehensive system statistics including CPU, memory, disk, network, and uptime information.
type SystemStats struct {
	Timestamp time.Time
	CPU       CPUStats
	LoadAvg   []float64
	Disk      DiskStats
	Memory    MemoryStats
	Network   NetworkStats
	Uptime    time.Duration
}

// StatsSummary contains summarized system metrics with usage percentages and overall system status.
type StatsSummary struct {
	Timestamp       time.Time
	CPUUsage        float64
	MemoryUsage     float64
	DiskActivity    float64
	NetworkActivity float64
	Status          SystemStatus
}

// SystemStatus represents the overall system health status based on resource usage thresholds.
type SystemStatus int

// System status constants representing different health levels.
const (
	StatusNormal SystemStatus = iota
	StatusWarning
	StatusCritical
)

func (s SystemStatus) String() string {
	switch s {
	case StatusNormal:
		return "normal"
	case StatusWarning:
		return "warning"
	case StatusCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Thresholds defines resource usage thresholds for determining system status warnings and critical alerts.
type Thresholds struct {
	CPUWarning     float64
	CPUCritical    float64
	MemoryWarning  float64
	MemoryCritical float64
	DiskWarning    float64
	DiskCritical   float64
}

// DefaultThresholds returns default system resource usage thresholds for warning and critical status levels.
func DefaultThresholds() Thresholds {
	return Thresholds{
		CPUWarning:     70.0,
		CPUCritical:    90.0,
		MemoryWarning:  80.0,
		MemoryCritical: 95.0,
		DiskWarning:    80.0,
		DiskCritical:   95.0,
	}
}
