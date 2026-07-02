//go:build !linux && !darwin

package collector

// collectGPUProcs is unsupported on non-Linux, non-Darwin platforms.
func collectGPUProcs() (procs []GPUProc, supported bool) {
	return nil, false
}
