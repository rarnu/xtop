//go:build linux

package collector

import (
	"context"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// zeroGPUThreshold is the GPU memory below which a process is considered not to
// be actively using the GPU and is hidden from the GPU card list.
const zeroGPUThreshold float64 = 0.1 // bytes

func filterGPUProcList(list []GPUProc) []GPUProc {
	filtered := make([]GPUProc, 0, len(list))
	for _, p := range list {
		if float64(p.MemBytes) >= zeroGPUThreshold {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// collectGPUProcs lists all processes currently using the GPU via `nvidia-smi`.
// When nvidia-smi is absent or fails (macOS/Apple silicon, non-NVIDIA Linux) it
// reports the metric as unsupported. Memory is returned in bytes.
func collectGPUProcs() (procs []GPUProc, supported bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-compute-apps=pid,process_name,used_gpu_memory",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, false
	}

	var list []GPUProc
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		// pid = first, used memory = last (MiB), name = everything between.
		pid64, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			continue
		}
		memMiB, err := strconv.ParseFloat(strings.TrimSpace(parts[len(parts)-1]), 64)
		if err != nil || memMiB < 0 {
			continue
		}
		name := strings.TrimSpace(strings.Join(parts[1:len(parts)-1], ","))
		if shouldHideProc(name) {
			continue
		}
		list = append(list, GPUProc{
			PID:      int32(pid64),
			Command:  name,
			MemBytes: uint64(memMiB) * 1024 * 1024,
		})
	}

	list = filterGPUProcList(list)
	sort.Slice(list, func(i, j int) bool { return list[i].MemBytes > list[j].MemBytes })
	return list, true
}
