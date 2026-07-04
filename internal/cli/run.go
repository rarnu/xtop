package cli

import (
	"context"
	"fmt"
	"time"

	"xtop/internal/collector"
	"xtop/internal/version"
)

// Run executes the command-line mode according to cfg. It collects the requested
// subsystems, prints them, and optionally streams forever until interrupted.
func Run(cfg Config) error {
	c := collector.New()

	// The process cache is maintained by a background goroutine. Start it when
	// process information is needed and wait briefly for the first sample so that
	// platforms relying on a persistent collector (e.g. Darwin/top) still produce
	// useful one-shot output.
	if cfg.All || cfg.Proc {
		c.StartProcLoop()
		waitForFirstProcCache(c, 3*time.Second)
	}

	for {
		out := collect(cfg, c)
		if err := printOutput(out, cfg); err != nil {
			return err
		}

		if cfg.Stream <= 0 {
			break
		}
		time.Sleep(time.Duration(cfg.Stream) * time.Second)
	}

	return nil
}

// waitForFirstProcCache blocks until the collector's process cache has been
// populated at least once or the timeout expires.
func waitForFirstProcCache(c *collector.Collector, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// If the cache already has data, return immediately.
	if ps := c.ProcCache(); len(ps.All) > 0 {
		return
	}

	select {
	case <-c.ProcUpdate():
	case <-ctx.Done():
	}
}

// collect gathers the requested subsystems into an Output value.
func collect(cfg Config, c *collector.Collector) Output {
	out := Output{Time: time.Now()}

	if cfg.All || cfg.CPU {
		cpu := c.CollectCPU()
		out.CPU = &cpu
	}
	if cfg.All || cfg.Mem {
		mem := c.CollectMem()
		out.Mem = &mem
	}
	if cfg.All || cfg.Disk {
		disk := c.CollectDisk()
		out.Disk = &disk
	}
	if cfg.All || cfg.GPU {
		gpu := c.CollectGPU()
		out.GPU = &gpu
	}
	if cfg.All || cfg.Net {
		net := c.CollectNet()
		out.Net = &net
	}
	if cfg.All || cfg.Proc {
		ps := c.ProcCache()
		out.Proc = &ProcOutput{
			Total:   len(ps.All),
			TopCPU:  ps.Top,
			TopMem:  ps.TopMem,
			TopDisk: ps.TopDisk,
		}
	}

	return out
}

// PrintVersion prints the version line.
func PrintVersion() {
	fmt.Printf("xtop version %s\n", version.Version)
}
