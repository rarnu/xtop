//go:build darwin

package collector

/*
#include <libproc.h>
#include <sys/resource.h>
*/
import "C"
import (
	"unsafe"

	"github.com/shirou/gopsutil/v4/process"
)

// procDiskSupported reports whether per-process disk I/O counters are available
// on this platform. macOS exposes cumulative per-process disk bytes via
// proc_pid_rusage(RIUSAGE_INFO_CURRENT) -> rusage_info_v6.ri_diskio_bytes{read,written}.
const procDiskSupported = true

// procDiskUsage returns the cumulative read and write bytes for a process using
// the libproc private API. It works without root for processes owned by the
// current user; other users' processes are silently skipped.
func procDiskUsage(pid int32) (read uint64, write uint64, ok bool) {
	var ru C.struct_rusage_info_v6
	if C.proc_pid_rusage(C.int(pid), C.RUSAGE_INFO_CURRENT, (*C.rusage_info_t)(unsafe.Pointer(&ru))) != 0 {
		return 0, 0, false
	}
	return uint64(ru.ri_diskio_bytesread), uint64(ru.ri_diskio_byteswritten), true
}

// procDiskBytes returns the process's cumulative read+write bytes.
func procDiskBytes(p *process.Process) (uint64, bool) {
	r, w, ok := procDiskUsage(p.Pid)
	if !ok {
		return 0, false
	}
	return r + w, true
}

// procDiskReadBytes returns the process's cumulative read bytes.
func procDiskReadBytes(p *process.Process) (uint64, bool) {
	r, _, ok := procDiskUsage(p.Pid)
	return r, ok
}

// procDiskWriteBytes returns the process's cumulative write bytes.
func procDiskWriteBytes(p *process.Process) (uint64, bool) {
	_, w, ok := procDiskUsage(p.Pid)
	return w, ok
}
