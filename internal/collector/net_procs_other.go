//go:build !darwin && !linux

package collector

import "context"

// collectNetProcsLoop is a no-op on platforms without a per-process network
// traffic data source. The UI will simply show an empty process list until
// data becomes available.
func collectNetProcsLoop(_ context.Context, _ *Collector) {}
