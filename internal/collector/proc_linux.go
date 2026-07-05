//go:build linux

package collector

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	list, counts, err := c.walkProcLinux(dt)
	if err != nil {
		return ProcStat{DiskSupported: procDiskSupported}
	}

	next := counts.cpu
	nextDisk := counts.diskTotal
	nextRead := counts.diskRead
	nextWrite := counts.diskWrite

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

// procCounts holds the cumulative counters gathered during a Linux /proc walk.
type procCounts struct {
	cpu       map[int32]float64
	diskTotal map[int32]uint64
	diskRead  map[int32]uint64
	diskWrite map[int32]uint64
}

// walkProcLinux reads /proc directly, avoiding the per-PID *process.Process
// object allocations created by gopsutil's process.Processes(). It reads only
// the fields required by xtop: command line, status, stat (utime/stime), memory
// RSS, creation time and /proc/[pid]/io counters.
func (c *Collector) walkProcLinux(dt float64) ([]ProcInfo, procCounts, error) {
	dents, err := os.ReadDir("/proc")
	if err != nil {
		return nil, procCounts{}, err
	}

	list := make([]ProcInfo, 0, len(dents))
	counts := procCounts{
		cpu:       make(map[int32]float64, len(dents)),
		diskTotal: make(map[int32]uint64, len(dents)),
		diskRead:  make(map[int32]uint64, len(dents)),
		diskWrite: make(map[int32]uint64, len(dents)),
	}

	for _, e := range dents {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		pid64, err := strconv.ParseInt(name, 10, 32)
		if err != nil {
			continue
		}
		pid := int32(pid64)

		base := filepath.Join("/proc", name)
		cmd := readProcCmdline(filepath.Join(base, "cmdline"))
		if cmd == "" || cmd == "?" || shouldHideProc(cmd) {
			continue
		}

		status := readProcStatus(filepath.Join(base, "status"))
		stat := readProcStat(filepath.Join(base, "stat"))

		info := ProcInfo{
			PID:     pid,
			Command: cmd,
			User:    c.usernameOfUID(status.uid),
			Status:  shortStatusCode(status.state),
			MemRSS:  status.rss,
			Start:   procStartTime(stat.startTime),
		}

		// CPU seconds from utime+stime (clock ticks).
		cpuSecs := float64(stat.utime+stat.stime) / userHZ
		counts.cpu[pid] = cpuSecs
		if dt > 0 {
			if prev, ok := c.prevProc[pid]; ok && cpuSecs >= prev {
				info.CPU = (cpuSecs - prev) / dt * 100
			}
		}

		if procDiskSupported {
			io := readProcIO(filepath.Join(base, "io"))
			counts.diskTotal[pid] = io.readBytes + io.writeBytes
			counts.diskRead[pid] = io.readBytes
			counts.diskWrite[pid] = io.writeBytes
			if dt > 0 {
				if prev, ok := c.prevProcDisk[pid]; ok && io.readBytes+io.writeBytes >= prev {
					info.DiskBytesPerSec = float64(io.readBytes+io.writeBytes-prev) / dt
				}
				if pr, ok := c.prevProcRead[pid]; ok && io.readBytes >= pr {
					info.DiskReadPerSec = float64(io.readBytes-pr) / dt
				}
				if pw, ok := c.prevProcWrite[pid]; ok && io.writeBytes >= pw {
					info.DiskWritePerSec = float64(io.writeBytes-pw) / dt
				}
			}
		}

		list = append(list, info)
	}

	return list, counts, nil
}

// procStatus holds the fields we need from /proc/[pid]/status.
type procStatus struct {
	state    string
	uid      uint32
	rss      uint64
}

// readProcStatus parses a minimal subset of /proc/[pid]/status.
func readProcStatus(path string) procStatus {
	b, err := os.ReadFile(path)
	if err != nil {
		return procStatus{}
	}
	var s procStatus
	for _, line := range strings.Split(string(b), "\n") {
		if len(line) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "State:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				s.state = fields[1]
			}
		case strings.HasPrefix(line, "Uid:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if uid, err := strconv.ParseUint(fields[1], 10, 32); err == nil {
					s.uid = uint32(uid)
				}
			}
		case strings.HasPrefix(line, "VmRSS:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if v, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					s.rss = v * 1024
				}
			}
		}
	}
	return s
}

// readProcCmdline reads /proc/[pid]/cmdline and returns the command line with
// NUL bytes replaced by spaces. Falls back to "?" on error.
func readProcCmdline(path string) string {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return "?"
	}
	for i, c := range b {
		if c == 0 {
			b[i] = ' '
		}
	}
	cmd := strings.TrimSpace(string(b))
	if cmd == "" {
		return "?"
	}
	return cmd
}

// procStat holds the fields we need from /proc/[pid]/stat.
type procStat struct {
	utime     uint64
	stime     uint64
	startTime uint64
}

// userHZ is the number of clock ticks per second. It is treated as a constant
// (100) on Linux, which matches the vast majority of distributions and avoids
// a sysconf call per process walk.
const userHZ = 100

// readProcStat parses the numeric fields of /proc/[pid]/stat. The command field
// may contain spaces and parentheses, so we locate the last ')' and parse the
// integers that follow it.
func readProcStat(path string) procStat {
	b, err := os.ReadFile(path)
	if err != nil {
		return procStat{}
	}
	s := string(b)
	idx := strings.LastIndex(s, ")")
	if idx < 0 {
		return procStat{}
	}
	fields := strings.Fields(s[idx+1:])
	// fields[0] = state
	// fields[1] = ppid
	// fields[2..11] = various counters
	// fields[12] = utime
	// fields[13] = stime
	// fields[19..20] depending on kernel: starttime is field 21 after command
	// After the closing parenthesis there are 40+ fields; indexes shift by one.
	var out procStat
	if len(fields) > 12 {
		out.utime = parseUint(fields[12])
	}
	if len(fields) > 13 {
		out.stime = parseUint(fields[13])
	}
	if len(fields) > 19 {
		out.startTime = parseUint(fields[19])
	}
	return out
}

func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

// procStartTime converts startTime ticks since boot to an absolute time. We
// approximate boot time as time.Now() - uptimeSeconds; this is good enough for
// display purposes and avoids reading /proc/uptime every walk.
var bootTime = time.Now().Add(-time.Duration(uptimeSeconds()) * time.Second)

func procStartTime(ticks uint64) time.Time {
	if ticks == 0 {
		return time.Time{}
	}
	return bootTime.Add(time.Duration(ticks) * time.Second / userHZ)
}

// uptimeSeconds reads /proc/uptime once at startup.
func uptimeSeconds() float64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return v
}

// procIO holds the fields we need from /proc/[pid]/io.
type procIO struct {
	readBytes  uint64
	writeBytes uint64
}

// readProcIO parses /proc/[pid]/io for read_bytes and write_bytes.
func readProcIO(path string) procIO {
	b, err := os.ReadFile(path)
	if err != nil {
		return procIO{}
	}
	var io procIO
	for _, line := range strings.Split(string(b), "\n") {
		if len(line) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "read_bytes:"):
			io.readBytes = parseUint(strings.TrimSpace(strings.TrimPrefix(line, "read_bytes:")))
		case strings.HasPrefix(line, "write_bytes:"):
			io.writeBytes = parseUint(strings.TrimSpace(strings.TrimPrefix(line, "write_bytes:")))
		}
	}
	return io
}
