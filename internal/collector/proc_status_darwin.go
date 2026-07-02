//go:build darwin

package collector

import (
	"os/exec"
	"strconv"
	"strings"
)

// batchStatuses returns pid -> raw status string for every process in a single
// `ps` call. On macOS gopsutil's per-process Status() forks `ps` each time, so
// calling it for 1000+ processes forks 1000+ times per refresh (seconds of CPU
// and the walk's dominant cost); one batched call avoids that entirely.
func batchStatuses() map[int32]string {
	out, err := detach(exec.Command("ps", "-axo", "pid=,stat=")).Output()
	if err != nil {
		return nil
	}
	m := make(map[int32]string)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.ParseInt(fields[0], 10, 32)
		if err != nil {
			continue
		}
		m[int32(pid)] = fields[1]
	}
	return m
}
