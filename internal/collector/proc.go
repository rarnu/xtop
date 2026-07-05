package collector

import (
	"container/heap"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

const topProcCount = 20

// hiddenProcs lists process names that should be hidden from dashboard process
// lists because they are the collectors/tools used by xtop itself.
var hiddenProcs = map[string]bool{
	"nettop": true,
	"top":    true,
	"xtop":   true,
}

func shouldHideProc(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	if hiddenProcs[base] {
		return true
	}
	// Also match commands that are absolute paths ending with these names.
	if base == "" {
		return false
	}
	// strip a trailing .exe for Windows/Wine compatibility
	if strings.HasSuffix(base, ".exe") {
		base = strings.TrimSuffix(base, ".exe")
		return hiddenProcs[base]
	}
	return false
}

func filterProcList(list []ProcInfo) []ProcInfo {
	filtered := make([]ProcInfo, 0, len(list))
	for _, p := range list {
		if !shouldHideProc(p.Command) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// topN returns a copy of the first n entries of an already-sorted slice.
func topN(sorted []ProcInfo, n int) []ProcInfo {
	if len(sorted) < n {
		n = len(sorted)
	}
	return append([]ProcInfo(nil), sorted[:n]...)
}

// ---- Top-K helpers: avoid O(n log n) full sorts for large process lists ------

type cpuHeap []ProcInfo

func (h cpuHeap) Len() int            { return len(h) }
func (h cpuHeap) Less(i, j int) bool  { return h[i].CPU < h[j].CPU }
func (h cpuHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *cpuHeap) Push(x interface{}) { *h = append(*h, x.(ProcInfo)) }
func (h *cpuHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type memHeap []ProcInfo

func (h memHeap) Len() int           { return len(h) }
func (h memHeap) Less(i, j int) bool { return h[i].MemRSS < h[j].MemRSS }
func (h memHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *memHeap) Push(x interface{}) {
	*h = append(*h, x.(ProcInfo))
}
func (h *memHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type diskHeap []ProcInfo

func (h diskHeap) diskRate(i int) float64 {
	return h[i].DiskReadPerSec + h[i].DiskWritePerSec
}
func (h diskHeap) Len() int { return len(h) }
func (h diskHeap) Less(i, j int) bool {
	return h.diskRate(i) < h.diskRate(j)
}
func (h diskHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *diskHeap) Push(x interface{}) {
	*h = append(*h, x.(ProcInfo))
}
func (h *diskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// topKByCPU returns the top n processes by CPU, sorted descending.
func topKByCPU(list []ProcInfo, n int) []ProcInfo {
	if n <= 0 {
		return nil
	}
	h := &cpuHeap{}
	heap.Init(h)
	for _, p := range list {
		if h.Len() < n {
			heap.Push(h, p)
		} else if p.CPU > (*h)[0].CPU {
			heap.Pop(h)
			heap.Push(h, p)
		}
	}
	out := make([]ProcInfo, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(ProcInfo)
	}
	return out
}

// topKByMem returns the top n processes by memory RSS, sorted descending.
func topKByMem(list []ProcInfo, n int) []ProcInfo {
	if n <= 0 {
		return nil
	}
	h := &memHeap{}
	heap.Init(h)
	for _, p := range list {
		if h.Len() < n {
			heap.Push(h, p)
		} else if p.MemRSS > (*h)[0].MemRSS {
			heap.Pop(h)
			heap.Push(h, p)
		}
	}
	out := make([]ProcInfo, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(ProcInfo)
	}
	return out
}

// topKByDisk returns the top n processes by (read+write) rate, sorted descending.
func topKByDisk(list []ProcInfo, n int) []ProcInfo {
	if n <= 0 {
		return nil
	}
	h := &diskHeap{}
	heap.Init(h)
	for _, p := range list {
		if h.Len() < n {
			heap.Push(h, p)
		} else if p.DiskReadPerSec+p.DiskWritePerSec > (*h)[0].DiskReadPerSec+(*h)[0].DiskWritePerSec {
			heap.Pop(h)
			heap.Push(h, p)
		}
	}
	out := make([]ProcInfo, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(ProcInfo)
	}
	return out
}

// shortStatus reduces one process's status to a single display letter.
func shortStatus(p *process.Process) string {
	ss, err := p.Status()
	if err != nil || len(ss) == 0 {
		return "?"
	}
	return shortStatusCode(ss[0])
}

// shortStatusCode maps a raw status string to a single display letter. It
// accepts both gopsutil's letters/words and BSD `ps` codes (e.g. "Ss", "R+"),
// for which the leading character is the state.
func shortStatusCode(s string) string {
	if s == "" {
		return "?"
	}
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	switch s {
	case process.Running:
		return "R"
	case process.Sleep:
		return "S"
	case process.Idle:
		return "I"
	case process.Stop:
		return "T"
	case process.Zombie:
		return "Z"
	case process.Blocked:
		return "D"
	case process.Wait:
		return "W"
	case process.Lock:
		return "L"
	default:
		return strings.ToUpper(s[:1])
	}
}

// commandOf prefers the full command line, falling back to the process name.
func commandOf(p *process.Process) string {
	if cmd, err := p.Cmdline(); err == nil {
		if cmd = strings.TrimSpace(cmd); cmd != "" {
			return cmd
		}
	}
	if name, err := p.Name(); err == nil {
		return name
	}
	return "?"
}
