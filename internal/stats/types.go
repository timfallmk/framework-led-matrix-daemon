package stats

import "time"

type CPUStats struct {
	ModelName      string
	VendorID       string
	PerCorePercent []float64
	UsagePercent   float64
	PhysicalCores  int
	LogicalCores   int
}

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

type DiskStats struct {
	IOCounters   map[string]IOCounterStat
	Partitions   []PartitionStat
	TotalReads   uint64
	TotalWrites  uint64
	ReadBytes    uint64
	WriteBytes   uint64
	ActivityRate float64
}

type PartitionStat struct {
	Device      string
	Mountpoint  string
	Fstype      string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

type IOCounterStat struct {
	ReadCount  uint64
	WriteCount uint64
	ReadBytes  uint64
	WriteBytes uint64
	ReadTime   uint64
	WriteTime  uint64
}

type NetworkStats struct {
	BytesSent      uint64
	BytesRecv      uint64
	PacketsSent    uint64
	PacketsRecv    uint64
	TotalBytesSent uint64
	TotalBytesRecv uint64
	ActivityRate   float64
}

type SystemStats struct {
	Timestamp time.Time
	CPU       CPUStats
	LoadAvg   []float64
	Disk      DiskStats
	Memory    MemoryStats
	Network   NetworkStats
	Uptime    time.Duration
}

type StatsSummary struct {
	Timestamp       time.Time
	CPUUsage        float64
	MemoryUsage     float64
	DiskActivity    float64
	NetworkActivity float64
	Status          SystemStatus
}

type SystemStatus int

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

type Thresholds struct {
	CPUWarning     float64
	CPUCritical    float64
	MemoryWarning  float64
	MemoryCritical float64
	DiskWarning    float64
	DiskCritical   float64
}

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
