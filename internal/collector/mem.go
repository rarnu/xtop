package collector

import "github.com/shirou/gopsutil/v4/mem"

// collectMem maps gopsutil's virtual memory stats onto used / cached / free so
// the three segments always sum to Total (matching the donut in MEM.png).
func collectMem() MemStat {
	vm, err := mem.VirtualMemory()
	if err != nil || vm == nil {
		return MemStat{}
	}

	used := vm.Used
	cached := vm.Cached
	// On some platforms (notably darwin) Cached is 0 while Inactive holds the
	// reclaimable page cache; fall back to it so the segment is meaningful.
	if cached == 0 {
		cached = vm.Inactive
	}

	if used > vm.Total {
		used = vm.Total
	}
	if used+cached > vm.Total {
		cached = vm.Total - used
	}
	free := vm.Total - used - cached

	return MemStat{
		Total:  vm.Total,
		Used:   used,
		Cached: cached,
		Free:   free,
	}
}
