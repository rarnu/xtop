//go:build darwin

package collector

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// collectNetProcsLoop runs a persistent `nettop` process and parses its
// continuous stdout. Each sample is a per-process delta of bytes_in +
// bytes_out, so the parsed values are already bytes-per-second rates.
//
// Keeping nettop alive avoids the ~5 second startup cost of spawning a fresh
// process for every sample, and gives the UI a cheap 1-second refresh signal.
// nettopSampleInterval is the interval nettop waits between samples. We keep it
// at 5 seconds because nettop is CPU-heavy: even in logging mode it routinely
// uses 80-130% of a core on a 3-second interval and ~87% on a 5-second interval.
// A 1-second interval makes the situation noticeably worse without giving the UI
// a materially faster refresh.
const nettopSampleInterval = 5 * time.Second

func collectNetProcsLoop(ctx context.Context, c *Collector) {
	const restartDelay = 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		runOneNettop(ctx, c)
		select {
		case <-ctx.Done():
			return
		case <-time.After(restartDelay):
		}
	}
}

func runOneNettop(ctx context.Context, c *Collector) {
	// -P: per-process aggregate
	// -x: raw numeric counts (no MiB/KiB suffixes)
	// -d: delta mode (values are per-interval changes)
	// -s 3: 3 second update interval (nettop is very CPU-heavy, so avoid 1s)
	// -l 0: infinite logging-mode samples
	// -J: keep only bytes_in and bytes_out
	cmd := detach(exec.CommandContext(ctx, "nettop",
		"-P", "-x", "-d", "-s", strconv.Itoa(int(nettopSampleInterval.Seconds())), "-l", "0",
		"-J", "bytes_in,bytes_out",
	))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	// Ensure the child is killed when the collector context is cancelled.
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	parseNettopStream(ctx, stdout, c)
	_ = cmd.Wait()
}

func parseNettopStream(ctx context.Context, r io.Reader, c *Collector) {
	scanner := bufio.NewScanner(r)
	// nettop lines are short; the default 64 KiB buffer is plenty.

	var sample []NetProc
	first := true

	flush := func() {
		if len(sample) == 0 {
			return
		}
		if first {
			// The first sample after nettop starts is cumulative (not a delta),
			// so skip it to avoid reporting lifetime totals as an instantaneous
			// rate.
			first = false
			sample = sample[:0]
			return
		}
		publishNetProcs(c, sample)
		sample = sample[:0]
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if isNettopHeader(line) {
			flush()
			continue
		}
		if np, ok := parseNettopLine(line); ok {
			if !shouldHideProc(np.Command) {
				sample = append(sample, np)
			}
		}
	}
	flush()
}

func isNettopHeader(line string) bool {
	return strings.Contains(line, "bytes_in") && strings.Contains(line, "bytes_out")
}

func parseNettopLine(line string) (NetProc, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return NetProc{}, false
	}

	inStr := fields[len(fields)-2]
	outStr := fields[len(fields)-1]

	inBytes, err1 := strconv.ParseUint(inStr, 10, 64)
	outBytes, err2 := strconv.ParseUint(outStr, 10, 64)
	if err1 != nil || err2 != nil {
		return NetProc{}, false
	}

	namePID := strings.Join(fields[:len(fields)-2], " ")
	dot := strings.LastIndex(namePID, ".")
	if dot < 0 {
		return NetProc{}, false
	}

	pid64, err := strconv.ParseInt(namePID[dot+1:], 10, 32)
	if err != nil {
		return NetProc{}, false
	}

	return NetProc{
		PID:            int32(pid64),
		Command:        strings.TrimSpace(namePID[:dot]),
		UploadPerSec:   float64(outBytes),
		DownloadPerSec: float64(inBytes),
		BytesPerSec:    float64(inBytes + outBytes),
	}, true
}

// zeroTrafficThreshold is the bytes/sec below which a per-process network entry
// is considered effectively idle and hidden from the network card list.
const zeroTrafficThreshold = 0.1 // bytes per second

func publishNetProcs(c *Collector, list []NetProc) {
	list = filterNetProcList(list)
	sort.Slice(list, func(i, j int) bool { return list[i].BytesPerSec > list[j].BytesPerSec })
	if len(list) > topProcCount {
		list = list[:topProcCount]
	}

	c.netProcMu.Lock()
	c.netProcCache = append([]NetProc(nil), list...)
	c.netProcSupported = true
	c.netProcMu.Unlock()

	select {
	case c.netProcUpdate <- struct{}{}:
	default:
	}
}

func filterNetProcList(list []NetProc) []NetProc {
	filtered := make([]NetProc, 0, len(list))
	for _, p := range list {
		if p.UploadPerSec >= zeroTrafficThreshold || p.DownloadPerSec >= zeroTrafficThreshold {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func init() {
	// nettop is launched with Setsid so it survives abrupt xtop crashes. Kill any
	// stale nettop processes started by previous xtop instances to prevent them
	// from stacking up and consuming CPU.
	killStaleNettop()
}

func killStaleNettop() {
	out, err := exec.Command("ps", "-eo", "pid=,command=").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid == os.Getpid() {
			continue
		}
		cmd := strings.ToLower(strings.Join(fields[1:], " "))
		if !strings.Contains(cmd, "nettop") {
			continue
		}
		if strings.Contains(cmd, "-p") && strings.Contains(cmd, "-x") &&
			strings.Contains(cmd, "-d") && strings.Contains(cmd, "bytes_in,bytes_out") {
			_ = killProcess(pid)
		}
	}
}

func killProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
