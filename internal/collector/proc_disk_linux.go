//go:build linux

package collector

import "github.com/shirou/gopsutil/v4/process"

// procDiskSupported reports whether per-process disk I/O counters are available
// on this platform. Linux exposes them via /proc/<pid>/io.
const procDiskSupported = true

// procDiskBytes returns the process's cumulative read+write bytes.
func procDiskBytes(p *process.Process) (uint64, bool) {
	io, err := p.IOCounters()
	if err != nil || io == nil {
		return 0, false
	}
	return io.ReadBytes + io.WriteBytes, true
}

// procDiskReadBytes returns the process's cumulative read bytes.
func procDiskReadBytes(p *process.Process) (uint64, bool) {
	io, err := p.IOCounters()
	if err != nil || io == nil {
		return 0, false
	}
	return io.ReadBytes, true
}

// procDiskWriteBytes returns the process's cumulative write bytes.
func procDiskWriteBytes(p *process.Process) (uint64, bool) {
	io, err := p.IOCounters()
	if err != nil || io == nil {
		return 0, false
	}
	return io.WriteBytes, true
}
