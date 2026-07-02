//go:build linux

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

// collectNetProcsLoop runs a persistent `nethogs` process in trace mode and
// parses its continuous stdout. Trace mode (-t) emits one line per process per
// refresh, with tab-separated PID/program and sent/received KB/s.
//
// Nethogs needs root on many systems (CAP_NET_ADMIN / packet capture), so it may
// exit immediately if not privileged. The loop will simply restart on failure
// and leave netProcCache empty until it succeeds.
func collectNetProcsLoop(ctx context.Context, c *Collector) {
	const restartDelay = 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		runOneNethogs(ctx, c)
		select {
		case <-ctx.Done():
			return
		case <-time.After(restartDelay):
		}
	}
}

func runOneNethogs(ctx context.Context, c *Collector) {
	// -t: trace mode (line-based output, no ncurses)
	// -d 1: 1 second refresh interval
	// No interface argument means it monitors the default route interface.
	cmd := detach(exec.CommandContext(ctx, "nethogs", "-t", "-d", "1"))
	// nethogs refuses to run in some environments unless TERM is set.
	cmd.Env = append([]string{"TERM=xterm-256color"}, os.Environ()...)

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

	parseNethogsStream(ctx, stdout, c)
	_ = cmd.Wait()
}

func parseNethogsStream(ctx context.Context, r io.Reader, c *Collector) {
	scanner := bufio.NewScanner(r)
	// nethogs lines are short; default buffer is fine.

	var sample []NetProc

	flush := func() {
		if len(sample) == 0 {
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
		// Trace-mode output groups process lines, then a blank line, then the
		// per-process lines repeat. Use blank lines to finalise a sample.
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if np, ok := parseNethogsLine(line); ok {
			sample = append(sample, np)
		}
	}
	flush()
}

func parseNethogsLine(line string) (NetProc, bool) {
	// nethogs -t output format examples:
	//   pid/program	dev	sent_kb/s	received_kb/s
	//   1234/foo    eth0 0.123  0.456
	// Some fields may be separated by spaces when output is not a TTY.
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return NetProc{}, false
	}

	pidProg := fields[0]
	slash := strings.Index(pidProg, "/")
	if slash < 0 {
		return NetProc{}, false
	}

	pid, err := strconv.ParseInt(pidProg[:slash], 10, 32)
	if err != nil {
		return NetProc{}, false
	}

	sentKbps, err1 := strconv.ParseFloat(fields[len(fields)-2], 64)
	recvKbps, err2 := strconv.ParseFloat(fields[len(fields)-1], 64)
	if err1 != nil || err2 != nil {
		return NetProc{}, false
	}

	program := pidProg[slash+1:]
	if program == "" || program == "unknown" {
		program = "?"
	}

	return NetProc{
		PID:         int32(pid),
		Command:     program,
		BytesPerSec: (sentKbps + recvKbps) * 1024,
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
