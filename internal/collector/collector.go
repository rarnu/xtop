package collector

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

// Collector produces Snapshots. It keeps each rate-based subsystem's previous
// counters plus the timestamp of that subsystem's last sample, so it can derive
// rates (disk IO, network throughput, per-process CPU) from deltas.
//
// The six subsystems keep disjoint state, so the TUI may collect them
// concurrently (one goroutine per subsystem). CPU/mem/GPU are stateless; disk,
// net and proc each touch only their own prev* fields, so no two goroutines
// share mutable state. Do not, however, run two collections of the *same*
// subsystem in parallel.
type Collector struct {
	prevDiskIO   map[string]ioCounter // io-device name -> read/write bytes
	prevDiskTime time.Time

	prevNet     netCounter
	prevNetTime time.Time

	prevProc     map[int32]float64 // pid -> cumulative cpu seconds (user+system)
	prevProcTime time.Time
}

type ioCounter struct {
	read  uint64
	write uint64
}

type netCounter struct {
	sent uint64
	recv uint64
}

// New returns a ready-to-use Collector.
func New() *Collector {
	return &Collector{
		prevDiskIO: map[string]ioCounter{},
		prevProc:   map[int32]float64{},
	}
}

// Snapshot captures every subsystem once. Rate-based fields are zero on the
// very first call for each subsystem (no previous sample to diff against).
func (c *Collector) Snapshot() Snapshot {
	return Snapshot{
		Time: time.Now(),
		CPU:  c.CollectCPU(),
		Mem:  c.CollectMem(),
		Disk: c.CollectDisk(),
		Net:  c.CollectNet(),
		GPU:  c.CollectGPU(),
		Proc: c.CollectProc(),
	}
}

// dtSince returns the elapsed seconds since *prev (0 the first time, when *prev
// is the zero Time) and advances *prev to now.
func dtSince(prev *time.Time) float64 {
	now := time.Now()
	dt := 0.0
	if !prev.IsZero() {
		if d := now.Sub(*prev).Seconds(); d > 0 {
			dt = d
		}
	}
	*prev = now
	return dt
}

// CollectCPU returns overall and per-core utilisation.
func (c *Collector) CollectCPU() CPUStat { return collectCPU() }

// CollectMem returns physical memory usage.
func (c *Collector) CollectMem() MemStat { return collectMem() }

// CollectDisk returns per-mount usage and IO rates.
func (c *Collector) CollectDisk() DiskStat { return c.collectDisk(dtSince(&c.prevDiskTime)) }

// CollectNet returns aggregate network throughput.
func (c *Collector) CollectNet() NetStat { return c.collectNet(dtSince(&c.prevNetTime)) }

// CollectGPU returns GPU stats.
func (c *Collector) CollectGPU() GPUStat { return collectGPU() }

// CollectProc returns process list and top CPU consumers.
func (c *Collector) CollectProc() ProcStat { return c.collectProc(dtSince(&c.prevProcTime)) }

// collectCPU returns overall and per-core utilisation. gopsutil keeps separate
// internal state for percpu vs total, so we derive overall from the per-core
// values to stay consistent and avoid a second sample.
func collectCPU() CPUStat {
	perCore, err := cpu.Percent(0, true)
	if err != nil || len(perCore) == 0 {
		return CPUStat{}
	}
	var sum float64
	for _, v := range perCore {
		sum += v
	}
	return CPUStat{
		Overall: sum / float64(len(perCore)),
		PerCore: perCore,
	}
}
