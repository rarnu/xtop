//go:build darwin

package collector

import (
	"context"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

var reIOUserClientCreator = regexp.MustCompile(`"IOUserClientCreator"\s*=\s*"pid\s+(\d+),`)

// collectGPUProcs lists all processes that currently have an IOAccelerator user
// client open on macOS. This mirrors the approach used by apple-smi: the data is
// available without root by querying the IORegistry. Memory is the process's RSS
// (Apple Silicon uses unified memory, so this is the closest available proxy for
// "GPU memory").
func collectGPUProcs() (procs []GPUProc, supported bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ioreg", "-c", "IOAccelerator", "-r", "-l").Output()
	if err != nil {
		return nil, false
	}

	pidSet := make(map[int32]struct{})
	for _, m := range reIOUserClientCreator.FindAllSubmatch(out, -1) {
		pid64, err := strconv.ParseInt(string(m[1]), 10, 32)
		if err != nil {
			continue
		}
		pidSet[int32(pid64)] = struct{}{}
	}

	if len(pidSet) == 0 {
		return nil, false
	}

	var list []GPUProc
	for pid := range pidSet {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		name := commandOf(p)
		if name == "?" {
			continue
		}
		var mem uint64
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			mem = mi.RSS
		}
		list = append(list, GPUProc{
			PID:      pid,
			Command:  name,
			MemBytes: mem,
		})
	}

	if len(list) == 0 {
		return nil, false
	}

	sort.Slice(list, func(i, j int) bool { return list[i].MemBytes > list[j].MemBytes })
	return list, true
}
