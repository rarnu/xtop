//go:build darwin

package collector

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// runProcLoop is the macOS process-cache refresh loop. It runs a persistent
// `top` process and parses its streaming logging output, which is dramatically
// cheaper than walking every process with gopsutil (which forks `ps` for each
// process's status on macOS).
func runProcLoop(ctx context.Context, c *Collector) {
	const restartDelay = 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		runOneTop(ctx, c)
		select {
		case <-ctx.Done():
			return
		case <-time.After(restartDelay):
		}
	}
}

func runOneTop(ctx context.Context, c *Collector) {
	// -l 0: infinite logging-mode samples
	// -s 1: 1 second update interval
	// -o cpu: sort by CPU descending
	// -stats: keep only the columns we need
	cmd := detach(exec.CommandContext(ctx, "top",
		"-l", "0", "-s", "1", "-o", "cpu",
		"-stats", "pid,command,cpu,mem,time,uid,state,user",
	))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	parseTopStream(ctx, stdout, c)
	_ = cmd.Wait()
}

func parseTopStream(ctx context.Context, r io.Reader, c *Collector) {
	scanner := bufio.NewScanner(r)
	var sample []ProcInfo
	samplesSeen := 0

	flush := func() {
		if len(sample) == 0 {
			return
		}
		// Publish the first sample immediately so the UI has data right away,
		// then throttle to once every ~3 seconds afterwards.
		if samplesSeen == 0 || samplesSeen%3 == 0 {
			publishProcs(c, sample)
		}
		samplesSeen++
		sample = sample[:0]
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if isTopHeader(line) {
			flush()
			continue
		}
		if pi, ok := parseTopLine(line); ok {
			if !shouldHideProc(pi.Command) {
				sample = append(sample, pi)
			}
		}
	}
	flush()
}

func isTopHeader(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "PID") && strings.Contains(line, "COMMAND")
}

func parseTopLine(line string) (ProcInfo, bool) {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return ProcInfo{}, false
	}

	pid, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return ProcInfo{}, false
	}

	user := fields[len(fields)-1]
	state := fields[len(fields)-2]
	// uidStr := fields[len(fields)-3] // numeric UID; we use the username column instead.
	// time := fields[len(fields)-4] // CPU time, currently unused
	memStr := fields[len(fields)-5]
	cpuStr := fields[len(fields)-6]
	command := strings.Join(fields[1:len(fields)-6], " ")

	cpu, err1 := strconv.ParseFloat(cpuStr, 64)
	mem, err2 := parseTopMem(memStr)
	if err1 != nil || err2 != nil {
		return ProcInfo{}, false
	}

	return ProcInfo{
		PID:     int32(pid),
		Command: command,
		CPU:     cpu,
		MemRSS:  mem,
		User:    user,
		Status:  topStateCode(state),
		// top does not provide process start time, so leave it zero.
	}, true
}

func parseTopMem(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	// top sometimes appends '+' or '-' to indicate a memory size change.
	last := s[len(s)-1]
	if last == '+' || last == '-' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return 0, nil
	}

	multiplier := uint64(1)
	switch s[len(s)-1] {
	case 'K', 'k':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'T', 't':
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v * multiplier, nil
}

func topStateCode(s string) string {
	switch strings.ToLower(s) {
	case "running":
		return "R"
	case "sleeping":
		return "S"
	case "stuck":
		return "D"
	case "idle":
		return "I"
	case "stopped", "halted":
		return "T"
	case "zombie":
		return "Z"
	default:
		return strings.ToUpper(s[:1])
	}
}

func publishProcs(c *Collector, list []ProcInfo) {
	list = filterProcList(list)

	// Avoid full sorts; use Top-K heaps to extract the top 20 by each metric.
	topCPU := topKByCPU(list, topProcCount)
	topMem := topKByMem(list, topProcCount)

	applyDarwinDiskIO(c, list)

	var topDisk []ProcInfo
	if procDiskSupported {
		topDisk = topKByDisk(list, topProcCount)
	}

	prev := c.procCache.All

	ps := ProcStat{
		All:           cloneProcInfos(list),
		Top:           topCPU,
		TopMem:        topMem,
		TopDisk:       topDisk,
		DiskSupported: procDiskSupported,
	}

	c.procMu.Lock()
	c.procCache = ps
	c.procMu.Unlock()
	putProcInfos(prev)

	select {
	case c.procUpdate <- struct{}{}:
	default:
	}
}

// applyDarwinDiskIO fills in per-process disk read/write rates using
// proc_pid_rusage. Because the darwin process list comes from a streaming `top`
// command, this is done as a second pass over the PIDs rather than during the
// initial parse. It only succeeds for processes owned by the current user;
// system/root processes are silently skipped.
func applyDarwinDiskIO(c *Collector, list []ProcInfo) {
	if !procDiskSupported {
		return
	}
	dt := dtSince(&c.prevDarwinDiskTime)
	nextRead := make(map[int32]uint64, len(list))
	nextWrite := make(map[int32]uint64, len(list))

	for i := range list {
		pid := list[i].PID
		r, w, ok := procDiskUsage(pid)
		if !ok {
			continue
		}
		nextRead[pid] = r
		nextWrite[pid] = w
		if dt > 0 {
			if pr, ok := c.prevProcRead[pid]; ok && r >= pr {
				list[i].DiskReadPerSec = float64(r-pr) / dt
			}
			if pw, ok := c.prevProcWrite[pid]; ok && w >= pw {
				list[i].DiskWritePerSec = float64(w-pw) / dt
			}
			list[i].DiskBytesPerSec = list[i].DiskReadPerSec + list[i].DiskWritePerSec
		}
	}

	c.prevProcRead = nextRead
	c.prevProcWrite = nextWrite
}

// collectProc on Darwin simply returns the top-derived cache. The expensive
// work is done by the persistent top goroutine started with StartProcLoop.
func (c *Collector) collectProc(_ float64) ProcStat {
	c.procMu.RLock()
	defer c.procMu.RUnlock()
	return c.procCache
}
