package collector

import (
	"context"
	"os/user"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/shirou/gopsutil/v4/cpu"
)

// SnapshotTTL is the maximum age of a cached snapshot returned by Snapshot().
// MCP servers and other high-frequency callers can reuse a snapshot within this
// window instead of triggering a fresh synchronous collection.
const SnapshotTTL = 1 * time.Second

// Collector produces Snapshots. It keeps each rate-based subsystem's previous
// counters plus the timestamp of that subsystem's last sample, so it can derive
// rates (disk IO, network throughput, per-process CPU) from deltas.
//
// CPU/mem/GPU are stateless; disk and net are collected on demand. Process data
// is special: it is refreshed continuously by a single dedicated goroutine that
// writes into procCache. The TUI reads from the cache and re-renders when the
// goroutine signals an update. This isolates the expensive process walk from the
// event loop and the renderer.
type Collector struct {
	prevDiskIO   map[string]ioCounter // io-device name -> read/write bytes
	prevDiskTime time.Time

	prevNet     netCounter
	prevNetTime time.Time

	// Process walk state, only touched by the dedicated process loop goroutine.
	prevProc      map[int32]float64 // pid -> cumulative cpu seconds (user+system)
	prevProcDisk  map[int32]uint64  // pid -> cumulative disk read+write bytes
	prevProcRead  map[int32]uint64  // pid -> cumulative disk read bytes
	prevProcWrite map[int32]uint64  // pid -> cumulative disk write bytes
	prevProcTime  time.Time
	prevDarwinDiskTime time.Time // only used on Darwin for top-derived disk rates
	procCache     ProcStat
	procMu        sync.RWMutex
	procUpdate    chan struct{} // signaled (buffered 1) after each cache refresh
	procCancel    context.CancelFunc

	// Per-process network traffic cache (macOS nettop). A single long-running
	// nettop process is started and its stdout is parsed continuously, avoiding
	// the ~5 second startup cost of spawning a fresh nettop for each sample.
	netProcCache      []NetProc
	netProcSupported  bool
	netProcMu         sync.RWMutex
	netProcUpdate     chan struct{}
	netProcCancel     context.CancelFunc

	// userCache memoises uid -> username. user.LookupId (getpwuid) can be very
	// slow on machines bound to a network directory (LDAP/AD); doing it per
	// process for 1000+ procs would take seconds. Usernames are stable, so we
	// cache across walks and only ever look up each distinct uid once.
	userCache map[uint32]string

	// cached snapshot for high-frequency read-only callers (e.g. MCP).
	snapMu    sync.RWMutex
	snapCache Snapshot
	snapTime  time.Time

	// NetProc interval; 0 means "use default". Mutable so callers can tune the
	// refresh rate of the macOS nettop goroutine.
	netProcInterval time.Duration
}

type ioCounter struct {
	read  uint64
	write uint64
}

type netCounter struct {
	sent uint64
	recv uint64
}

// procRefreshInterval is the pause between the end of one process-cache refresh
// and the start of the next. The walk itself can take a variable amount of time,
// so we sleep *after* it finishes rather than using a fixed ticker.
const procRefreshInterval = 3 * time.Second

// New returns a ready-to-use Collector. It does not start the background loops;
// call StartProcLoop / StartNetProcLoop when the UI wants that data.
func New() *Collector {
	return &Collector{
		prevDiskIO:      map[string]ioCounter{},
		prevProc:        map[int32]float64{},
		prevProcDisk:    map[int32]uint64{},
		prevProcRead:    map[int32]uint64{},
		prevProcWrite:   map[int32]uint64{},
		userCache:       map[uint32]string{},
		procUpdate:      make(chan struct{}, 1),
		netProcUpdate:   make(chan struct{}, 1),
		netProcInterval: 0,
	}
}

// SetNetProcInterval sets the interval used by the macOS nettop goroutine.
// It only takes effect the next time the goroutine is started. A value of 0
// means "use the default (5s steady state)". Values smaller than 1s are clamped
// to 1s to prevent excessive CPU usage.
func (c *Collector) SetNetProcInterval(d time.Duration) {
	if d < 0 {
		d = 0
	}
	if d > 0 && d < time.Second {
		d = time.Second
	}
	c.netProcInterval = d
}

// NetProcInterval returns the configured nettop interval, or 0 if using the
// built-in default.
func (c *Collector) NetProcInterval() time.Duration {
	return c.netProcInterval
}

// StartProcLoop starts the dedicated goroutine that refreshes procCache every
// collectInterval. Safe to call multiple times; subsequent calls are no-ops.
func (c *Collector) StartProcLoop() {
	if c.procCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.procCancel = cancel
	go c.procLoop(ctx)
}

// StopProcLoop stops the dedicated process cache goroutine.
func (c *Collector) StopProcLoop() {
	if c.procCancel != nil {
		c.procCancel()
		c.procCancel = nil
	}
}

// StartNetProcLoop starts the dedicated goroutine that parses nettop's
// continuous stdout. Safe to call multiple times.
func (c *Collector) StartNetProcLoop() {
	if c.netProcCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.netProcCancel = cancel
	go collectNetProcsLoop(ctx, c)
}

// StopNetProcLoop stops the nettop parsing goroutine.
func (c *Collector) StopNetProcLoop() {
	if c.netProcCancel != nil {
		c.netProcCancel()
		c.netProcCancel = nil
	}
}

func (c *Collector) procLoop(ctx context.Context) {
	runProcLoop(ctx, c)
}

func (c *Collector) refreshProcCache() {
	ps := c.collectProc(dtSince(&c.prevProcTime))
	c.procMu.Lock()
	prev := c.procCache.All
	c.procCache = ps
	c.procMu.Unlock()
	putProcInfos(prev)
	select {
	case c.procUpdate <- struct{}{}:
	default:
	}
}

// ProcCache returns the latest cached process snapshot. It is safe for
// concurrent read with the background loop.
func (c *Collector) ProcCache() ProcStat {
	c.procMu.RLock()
	defer c.procMu.RUnlock()
	return c.procCache
}

// ProcUpdate returns a channel that is signaled whenever procCache is refreshed.
func (c *Collector) ProcUpdate() <-chan struct{} { return c.procUpdate }

// NetProcCache returns the latest cached per-process network snapshot.
func (c *Collector) NetProcCache() []NetProc {
	c.netProcMu.RLock()
	defer c.netProcMu.RUnlock()
	return c.netProcCache
}

// NetProcSupported reports whether the nettop loop has produced at least one
// sample.
func (c *Collector) NetProcSupported() bool {
	c.netProcMu.RLock()
	defer c.netProcMu.RUnlock()
	return c.netProcSupported
}

// NetProcUpdate returns a channel that is signaled whenever netProcCache is refreshed.
func (c *Collector) NetProcUpdate() <-chan struct{} { return c.netProcUpdate }

// Snapshot captures every subsystem once. Rate-based fields are zero on the
// very first call for each subsystem (no previous sample to diff against).
//
// A short-lived cache is used so that high-frequency read-only callers (such as
// the MCP server) do not trigger a fresh synchronous collection on every call.
func (c *Collector) Snapshot() Snapshot {
	c.snapMu.RLock()
	if !c.snapTime.IsZero() && time.Since(c.snapTime) < SnapshotTTL {
		snap := c.snapCache
		c.snapMu.RUnlock()
		snap.Time = time.Now()
		return snap
	}
	c.snapMu.RUnlock()

	snap := Snapshot{
		Time: time.Now(),
		CPU:  c.CollectCPU(),
		Mem:  c.CollectMem(),
		Disk: c.CollectDisk(),
		Net:  c.CollectNet(),
		GPU:  c.CollectGPU(),
		Proc: c.CollectProc(),
	}
	c.snapMu.Lock()
	c.snapCache = snap
	c.snapTime = time.Now()
	c.snapMu.Unlock()
	return snap
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

// CollectDisk returns per-mount usage and IO rates. On the very first call it
// also fills in per-process disk IO by cumulative bytes so the disk card has
// something to show immediately; real per-second rates replace this on the next
// refresh.
func (c *Collector) CollectDisk() DiskStat {
	d := c.collectDisk(dtSince(&c.prevDiskTime))
	if procDiskSupported && len(c.ProcCache().TopDisk) == 0 {
		d.TopDiskProcs = c.collectDiskProcsOnce()
	}
	return d
}

// CollectNet returns aggregate network throughput plus, where supported, the
// top processes by per-process traffic. Aggregate counters are collected on
// demand; per-process traffic is read from the nettop cache maintained by the
// dedicated goroutine started with StartNetProcLoop.
func (c *Collector) CollectNet() NetStat {
	n := c.collectNet(dtSince(&c.prevNetTime))
	n.TopProcs = c.NetProcCache()
	n.ProcsSupported = c.NetProcSupported()
	return n
}

// CollectGPU returns GPU stats plus, where supported, every process currently
// using the GPU (sorted by VRAM descending).
func (c *Collector) CollectGPU() GPUStat {
	g := collectGPU()
	if top, supported := collectGPUProcs(); supported {
		g.TopProcs = top
		g.ProcsSupported = true
	}
	return g
}

// CollectProc returns the latest cached process snapshot. If the cache loop has
// not been started yet, it falls back to a synchronous walk so that Snapshot()
// and standalone callers still work.
func (c *Collector) CollectProc() ProcStat {
	c.procMu.RLock()
	cache := c.procCache
	c.procMu.RUnlock()
	if len(cache.All) > 0 {
		return cache
	}
	return c.collectProc(dtSince(&c.prevProcTime))
}

// usernameOf returns the username for a process, using the Collector's uid cache
// to avoid repeated slow getpwuid / directory-service lookups.
func (c *Collector) usernameOf(p *process.Process) string {
	uids, err := p.Uids()
	if err != nil || len(uids) == 0 {
		return ""
	}
	uid := uids[0]
	if u, ok := c.userCache[uid]; ok {
		return u
	}
	name := ""
	if u, err := user.LookupId(strconv.Itoa(int(uid))); err == nil {
		name = u.Username
	}
	c.userCache[uid] = name
	return name
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

// collectDiskProcsOnce walks all processes once and returns the top-N processes
// by cumulative disk read+write bytes. It is used only on the very first disk
// collection so the disk card has process data to show immediately, before the
// background proc loop has had time to compute per-second rates.
func (c *Collector) collectDiskProcsOnce() []ProcInfo {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	list := make([]ProcInfo, 0, len(procs))
	for _, p := range procs {
		cmd := commandOf(p)
		if cmd == "?" || shouldHideProc(cmd) {
			continue
		}
		var r, w uint64
		if v, ok := procDiskReadBytes(p); ok {
			r = v
		}
		if v, ok := procDiskWriteBytes(p); ok {
			w = v
		}
		if r == 0 && w == 0 {
			continue
		}
		list = append(list, ProcInfo{
			PID:             p.Pid,
			Command:         cmd,
			DiskReadPerSec:  float64(r),
			DiskWritePerSec: float64(w),
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].DiskReadPerSec+list[i].DiskWritePerSec >
			list[j].DiskReadPerSec+list[j].DiskWritePerSec
	})
	return topN(list, topProcCount)
}
