package collector

import "time"

// Snapshot is a single point-in-time capture of every subsystem metric.
// It is produced by Collector.Snapshot and consumed by the TUI.
type Snapshot struct {
	Time time.Time

	CPU  CPUStat
	Mem  MemStat
	Disk DiskStat
	Net  NetStat
	GPU  GPUStat
	Proc ProcStat
}

// CPUStat holds overall and per-core utilisation percentages (0-100).
type CPUStat struct {
	Overall float64   // aggregate utilisation across all cores
	PerCore []float64 // one entry per logical core
}

// MemStat holds physical memory usage in bytes.
type MemStat struct {
	Total  uint64
	Used   uint64
	Cached uint64
	Free   uint64 // Total - Used - Cached (clamped >= 0)
}

// DiskStat aggregates per-mount usage and IO throughput.
type DiskStat struct {
	Mounts     []DiskMount
	TotalBytes uint64 // sum of all mount capacities
	UsedBytes  uint64 // sum of all mount used bytes
}

// DiskMount describes a single mounted filesystem.
type DiskMount struct {
	Mountpoint  string
	Fstype      string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	ReadPerSec  float64 // bytes/sec since previous snapshot
	WritePerSec float64 // bytes/sec since previous snapshot
}

// NetStat holds aggregate network throughput and cumulative counters.
type NetStat struct {
	UploadPerSec   float64 // bytes/sec since previous snapshot
	DownloadPerSec float64 // bytes/sec since previous snapshot
	TotalUpload    uint64  // cumulative bytes sent since boot
	TotalDownload  uint64  // cumulative bytes received since boot
}

// GPUStat holds the list of detected GPUs plus availability info.
type GPUStat struct {
	Available bool     // false when no GPU data source is usable
	Message   string   // reason shown when Available is false
	Cards     []GPUCard
}

// GPUCard describes a single GPU. Fields set to the "N/A" sentinels below are
// rendered as unavailable by the TUI.
type GPUCard struct {
	Name     string
	PowerW   float64 // watts; <0 means N/A
	MemUsed  uint64  // bytes
	MemTotal uint64  // bytes; 0 means N/A
	TempC    float64 // celsius; <0 means N/A
	LoadPct  float64 // 0-100; <0 means N/A
}

// ProcStat holds the full process list plus the top-N convenience slice.
type ProcStat struct {
	All []ProcInfo // all processes, sorted by CPU desc by default
	Top []ProcInfo // top processes by CPU (feature 6)
}

// ProcInfo describes a single process.
type ProcInfo struct {
	PID     int32
	User    string
	Status  string // short code: R, S, D, Z, T, I ...
	CPU     float64
	MemRSS  uint64
	Start   time.Time
	Command string
}
