//go:build !darwin

package collector

// batchStatuses returns nil on platforms where a per-process Status() is already
// cheap (Linux reads /proc), signalling collectProc to use the per-process call.
func batchStatuses() map[int32]string { return nil }
