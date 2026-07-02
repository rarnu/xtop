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

	TopProcs      []NetProc // top processes by network throughput
	ProcsSupported bool     // false when per-process traffic isn't obtainable here
}

// NetProc describes one process's network throughput.
type NetProc struct {
	PID           int32
	Command       string
	UploadPerSec   float64 // bytes/sec sent
	DownloadPerSec float64 // bytes/sec received
	BytesPerSec   float64 // upload+download bytes/sec (kept for convenience)
}

// GPUStat holds the list of detected GPUs plus availability info.
type GPUStat struct {
	Available bool     // false when no GPU data source is usable
	Message   string   // reason shown when Available is false
	Cards     []GPUCard

	TopProcs       []GPUProc // top processes by GPU memory
	ProcsSupported bool      // false when per-process VRAM isn't obtainable here
}

// GPUProc describes one process's GPU memory usage.
type GPUProc struct {
	PID      int32
	Command  string
	MemBytes uint64
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

// ProcStat holds the full process list plus the top-N convenience slices.
type ProcStat struct {
	All     []ProcInfo // all processes, sorted by CPU desc by default
	Top     []ProcInfo // top processes by CPU (feature 6)
	TopMem  []ProcInfo // top processes by resident memory
	TopDisk []ProcInfo // top processes by disk read+write rate (Linux only)

	DiskSupported bool // false when per-process disk I/O isn't obtainable here
}

// ProcInfo describes a single process.
type ProcInfo struct {
	PID             int32
	User            string
	Status          string // short code: R, S, D, Z, T, I ...
	CPU             float64
	MemRSS          uint64
	DiskBytesPerSec float64 // read+write bytes/sec since previous snapshot (Linux)
	DiskReadPerSec  float64 // bytes/sec since previous snapshot (Linux)
	DiskWritePerSec float64 // bytes/sec since previous snapshot (Linux)
	Start           time.Time
	Command         string
}
