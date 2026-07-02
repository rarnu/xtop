//go:build !darwin

package collector

import "context"

// collectNetProcsLoop is a no-op on platforms without a per-process network
// traffic data source. The TUI will simply report network traffic as
// unsupported at the process level.
func collectNetProcsLoop(_ context.Context, _ *Collector) {}
