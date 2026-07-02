//go:build !linux

package collector

import "github.com/shirou/gopsutil/v4/process"

// procDiskSupported reports whether per-process disk I/O counters are available
// on this platform. gopsutil's IOCounters is unimplemented on darwin (and
// others), so it is reported as unsupported and never called.
const procDiskSupported = false

// procDiskBytes is a no-op on platforms without per-process disk I/O.
func procDiskBytes(_ *process.Process) (uint64, bool) {
	return 0, false
}
