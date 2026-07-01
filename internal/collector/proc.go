package collector

import (
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

const topProcCount = 20

// collectProc walks every process, deriving per-process CPU% from the delta of
// cumulative CPU time since the previous tick (cheap: one Times() call each,
// no second sampling pass). CPU% is 0 on the first tick.
func (c *Collector) collectProc(dt float64) ProcStat {
	procs, err := process.Processes()
	if err != nil {
		return ProcStat{}
	}

	next := make(map[int32]float64, len(procs))
	list := make([]ProcInfo, 0, len(procs))

	for _, p := range procs {
		// Skip processes whose command can't be read: they vanished mid-scan
		// and would otherwise show up as a useless "?" row.
		cmd := commandOf(p)
		if cmd == "?" {
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

		if u, err := p.Username(); err == nil {
			info.User = u
		}
		info.Status = shortStatus(p)
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			info.MemRSS = mi.RSS
		}
		if ct, err := p.CreateTime(); err == nil && ct > 0 {
			info.Start = time.UnixMilli(ct)
		}

		list = append(list, info)
	}

	c.prevProc = next

	sort.Slice(list, func(i, j int) bool { return list[i].CPU > list[j].CPU })

	top := list
	if len(top) > topProcCount {
		top = append([]ProcInfo(nil), list[:topProcCount]...)
	}

	return ProcStat{All: list, Top: top}
}

// shortStatus reduces gopsutil's status words to a single display letter.
func shortStatus(p *process.Process) string {
	ss, err := p.Status()
	if err != nil || len(ss) == 0 {
		return "?"
	}
	s := ss[0]
	if len(s) == 0 {
		return "?"
	}
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	switch s {
	case process.Running:
		return "R"
	case process.Sleep:
		return "S"
	case process.Idle:
		return "I"
	case process.Stop:
		return "T"
	case process.Zombie:
		return "Z"
	case process.Blocked:
		return "D"
	case process.Wait:
		return "W"
	case process.Lock:
		return "L"
	default:
		return strings.ToUpper(s[:1])
	}
}

// commandOf prefers the full command line, falling back to the process name.
func commandOf(p *process.Process) string {
	if cmd, err := p.Cmdline(); err == nil {
		if cmd = strings.TrimSpace(cmd); cmd != "" {
			return cmd
		}
	}
	if name, err := p.Name(); err == nil {
		return name
	}
	return "?"
}
