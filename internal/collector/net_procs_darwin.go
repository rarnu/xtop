//go:build darwin

package collector

import (
	"bufio"
	"context"
	"io"
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
	// -s 1: 1 second update interval
	// -l 0: infinite logging-mode samples
	// -J: keep only bytes_in and bytes_out
	cmd := detach(exec.CommandContext(ctx, "nettop",
		"-P", "-x", "-d", "-s", "1", "-l", "0",
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
			sample = append(sample, np)
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
		PID:         int32(pid64),
		Command:     strings.TrimSpace(namePID[:dot]),
		BytesPerSec: float64(inBytes + outBytes),
	}, true
}

func publishNetProcs(c *Collector, list []NetProc) {
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
