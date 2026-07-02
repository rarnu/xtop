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
// To avoid a cold-start delay, the first nethogs invocation uses a 1-second
// refresh so the UI gets data immediately; after the first sample is published
// the process is restarted with a 3-second refresh for steady-state operation.
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

		// Fast first frame: 1s refresh.
		gotData := runOneNethogs(ctx, c, 1)
		if !gotData {
			// nethogs failed or produced no data; wait before retrying.
			select {
			case <-ctx.Done():
				return
			case <-time.After(restartDelay):
			}
			continue
		}

		// Steady state: 3s refresh.
		runOneNethogs(ctx, c, 3)

		select {
		case <-ctx.Done():
			return
		case <-time.After(restartDelay):
		}
	}
}

// runOneNethogs starts nethogs with the given refresh interval, parses its
// output until the process exits or the context is cancelled, and returns true
// if at least one sample was published.
func runOneNethogs(ctx context.Context, c *Collector, intervalSec int) bool {
	binary, ok := findNethogs()
	if !ok {
		return false
	}

	cmd := detach(exec.CommandContext(ctx, binary, "-t", "-d", strconv.Itoa(intervalSec)))
	// nethogs refuses to run in some environments unless TERM is set.
	cmd.Env = append([]string{"TERM=xterm-256color"}, os.Environ()...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return false
	}
	if err := cmd.Start(); err != nil {
		return false
	}

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Capture stderr for diagnostics (nethogs prints permission/interface errors
	// there). The pipe will back-pressure if never read, so consume it in a tiny
	// goroutine. In normal operation it should be empty.
	go func() {
		_, _ = io.Copy(io.Discard, stderr)
	}()

	published := parseNethogsStream(ctx, stdout, c)
	_ = cmd.Wait()
	return published
}

// DebugNetProcsLogPath, when non-empty, receives raw nethogs output lines for
// troubleshooting why per-process network data isn't appearing.
var DebugNetProcsLogPath = ""

func debugLogNetProcs(line string) {
	if DebugNetProcsLogPath == "" {
		return
	}
	f, err := os.OpenFile(DebugNetProcsLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

func findNethogs() (string, bool) {
	candidates := []string{
		"/usr/sbin/nethogs",
		"/usr/local/bin/nethogs",
		"/usr/bin/nethogs",
		"/bin/nethogs",
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	// Fall back to PATH lookup in case it's somewhere else entirely.
	if p, err := exec.LookPath("nethogs"); err == nil {
		return p, true
	}
	return "", false
}

// parseNethogsStream parses nethogs -t output until the reader closes or ctx is
// cancelled. It returns true if at least one sample was published.
func parseNethogsStream(ctx context.Context, r io.Reader, c *Collector) bool {
	scanner := bufio.NewScanner(r)
	// nethogs lines are short; default buffer is fine.

	var sample []NetProc
	published := false

	flush := func() {
		if len(sample) == 0 {
			return
		}
		publishNetProcs(c, sample)
		published = true
		sample = sample[:0]
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return published
		default:
		}

		line := scanner.Text()
		debugLogNetProcs(line)
		// nethogs -t emits a "Refreshing:" banner before each sample. Use it as
		// the sample boundary so we publish once per refresh.
		if strings.TrimSpace(line) == "Refreshing:" {
			flush()
			continue
		}
		if np, ok := parseNethogsLine(line); ok {
			sample = append(sample, np)
		} else {
			debugLogNetProcs("[parse-failed] " + line)
		}
	}
	flush()
	return published
}

func parseNethogsLine(line string) (NetProc, bool) {
	// nethogs -t actual output examples:
	//   python/1480522/1001     3.40313 410.301
	//   172.17.0.103:22-125.80.223.88:59985/0/0 0.203125 0.128906
	// The last two fields are always sent and received KB/s. The first field
	// may be "program/pid/uid" or a connection key like "ip:port-ip:port/pid/uid".
	// For connection keys the pid is the first number after the slash.
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return NetProc{}, false
	}

	key := fields[0]
	slash := strings.Index(key, "/")
	if slash < 0 {
		return NetProc{}, false
	}

	prefix := strings.TrimSpace(key[:slash])
	after := key[slash+1:]
	// after may be "pid/uid".
	if i := strings.Index(after, "/"); i >= 0 {
		after = after[:i]
	}

	pid, err := strconv.ParseInt(after, 10, 32)
	if err != nil {
		return NetProc{}, false
	}

	sentKbps, err1 := strconv.ParseFloat(fields[len(fields)-2], 64)
	recvKbps, err2 := strconv.ParseFloat(fields[len(fields)-1], 64)
	if err1 != nil || err2 != nil {
		return NetProc{}, false
	}

	// Derive a display name from the prefix.
	program := programFromNethogsKey(prefix)

	return NetProc{
		PID:            int32(pid),
		Command:        program,
		UploadPerSec:   sentKbps * 1024,
		DownloadPerSec: recvKbps * 1024,
		BytesPerSec:    (sentKbps + recvKbps) * 1024,
	}, true
}

// programFromNethogsKey returns a readable program name from a nethogs -t key.
// Examples:
//   "python"               -> "python"
//   "172.17.0.103:22-..."  -> "172.17.0.103:22"
func programFromNethogsKey(prefix string) string {
	if prefix == "" || prefix == "unknown" {
		return "?"
	}
	// If it looks like a network connection key, show the local endpoint.
	if dash := strings.Index(prefix, "-"); dash >= 0 {
		return prefix[:dash]
	}
	return prefix
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
