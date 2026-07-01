package collector

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

// Collector produces Snapshots. It keeps the previous tick's counters so it can
// derive rates (disk IO, network throughput, per-process CPU) from deltas.
//
// Collector is not safe for concurrent use; call Snapshot from a single
// goroutine (the TUI drives it from one background command at a time).
type Collector struct {
	prevTime time.Time
	started  bool

	prevDiskIO map[string]ioCounter // io-device name -> read/write bytes
	prevNet    netCounter
	prevProc   map[int32]float64 // pid -> cumulative cpu seconds (user+system)
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

// Snapshot captures every subsystem once. Rate-based fields are zero on the very
// first call (no previous sample to diff against).
func (c *Collector) Snapshot() Snapshot {
	now := time.Now()
	dt := now.Sub(c.prevTime).Seconds()
	if !c.started || dt <= 0 {
		dt = 0
	}

	s := Snapshot{Time: now}
	s.CPU = collectCPU()
	s.Mem = collectMem()
	s.Disk = c.collectDisk(dt)
	s.Net = c.collectNet(dt)
	s.GPU = collectGPU()
	s.Proc = c.collectProc(dt)

	c.prevTime = now
	c.started = true
	return s
}

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
