package collector

import "github.com/shirou/gopsutil/v4/net"

// collectNet aggregates all interfaces into a single up/down throughput plus
// cumulative counters. Throughput is derived from the delta since the last tick.
func (c *Collector) collectNet(dt float64) NetStat {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return NetStat{}
	}
	agg := counters[0]

	cur := netCounter{sent: agg.BytesSent, recv: agg.BytesRecv}

	var up, down float64
	if dt > 0 {
		if cur.sent >= c.prevNet.sent {
			up = float64(cur.sent-c.prevNet.sent) / dt
		}
		if cur.recv >= c.prevNet.recv {
			down = float64(cur.recv-c.prevNet.recv) / dt
		}
	}
	c.prevNet = cur

	return NetStat{
		UploadPerSec:   up,
		DownloadPerSec: down,
		TotalUpload:    agg.BytesSent,
		TotalDownload:  agg.BytesRecv,
	}
}
