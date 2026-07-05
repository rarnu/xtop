//go:build linux

package collector

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// runProcLoop is the Linux process-cache refresh loop. It walks /proc every
// procRefreshInterval, which is fast enough that we can do it directly.
func runProcLoop(ctx context.Context, c *Collector) {
	for {
		c.refreshProcCache()
		select {
		case <-ctx.Done():
			return
		case <-time.After(procRefreshInterval):
		}
	}
}

// collectProc walks every process, deriving per-process CPU% (and, where the
// platform supports it, disk I/O rate) from the delta of cumulative counters
// since the previous tick. Rates are 0 on the first tick. A single walk feeds
// the CPU, memory and disk top-N lists.
func (c *Collector) collectProc(dt float64) ProcStat {
	procs, err := process.Processes()
	if err != nil {
		return ProcStat{DiskSupported: procDiskSupported}
	}

	next := make(map[int32]float64, len(procs))
	nextDisk := make(map[int32]uint64, len(procs))
	nextRead := make(map[int32]uint64, len(procs))
	nextWrite := make(map[int32]uint64, len(procs))
	list := make([]ProcInfo, 0, len(procs))

	// Statuses in one batch: on macOS a per-process p.Status() forks `ps` each
	// time, so we fetch them all in a single `ps` call. Nil on Linux where the
	// per-process call is already cheap.
	statusMap := batchStatuses()

	for _, p := range procs {
		cmd := commandOf(p)
		if cmd == "?" || shouldHideProc(cmd) {
			continue
		}
		pid := p.Pid

		var cpuSecs float64
		if t, err := p.Times(); err == nil && t != nil {
			cpuSecs = t.User + t.System
		}
		next[pid] = cpuSecs

		var cpuPct float64
		if dt > 0 {
			if prev, ok := c.prevProc[pid]; ok && cpuSecs >= prev {
				cpuPct = (cpuSecs - prev) / dt * 100
			}
		}

		info := ProcInfo{PID: pid, CPU: cpuPct, Command: cmd}

		info.User = c.usernameOf(p)
		if statusMap != nil {
			info.Status = shortStatusCode(statusMap[pid])
		} else {
			info.Status = shortStatus(p)
		}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			info.MemRSS = mi.RSS
		}
		if ct, err := p.CreateTime(); err == nil && ct > 0 {
			info.Start = time.UnixMilli(ct)
		}

		if procDiskSupported {
			if cur, ok := procDiskBytes(p); ok {
				nextDisk[pid] = cur
				if dt > 0 {
					if prev, ok := c.prevProcDisk[pid]; ok && cur >= prev {
						info.DiskBytesPerSec = float64(cur-prev) / dt
					}
				}
			}
			if curR, ok := procDiskReadBytes(p); ok {
				if dt > 0 {
					if prevR, ok := c.prevProcRead[pid]; ok && curR >= prevR {
						info.DiskReadPerSec = float64(curR-prevR) / dt
					}
				}
				nextRead[pid] = curR
			}
			if curW, ok := procDiskWriteBytes(p); ok {
				if dt > 0 {
					if prevW, ok := c.prevProcWrite[pid]; ok && curW >= prevW {
						info.DiskWritePerSec = float64(curW-prevW) / dt
					}
				}
				nextWrite[pid] = curW
			}
		}

		list = append(list, info)
	}

	c.prevProc = next
	c.prevProcDisk = nextDisk
	c.prevProcRead = nextRead
	c.prevProcWrite = nextWrite

	// Use Top-K heaps instead of full sorts: only the top 20 entries matter.
	top := topKByCPU(list, topProcCount)
	topMem := topKByMem(list, topProcCount)

	var topDisk []ProcInfo
	if procDiskSupported {
		topDisk = topKByDisk(list, topProcCount)
	}

	return ProcStat{
		All:           list,
		Top:           top,
		TopMem:        topMem,
		TopDisk:       topDisk,
		DiskSupported: procDiskSupported,
	}
}
