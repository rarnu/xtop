package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"xtop/internal/collector"
)

// Output is the CLI-friendly subset of a full Snapshot. Pointer fields are
// populated only when the corresponding subsystem was requested, and omitted
// from JSON when nil.
type Output struct {
	Time time.Time           `json:"time"`
	CPU  *collector.CPUStat  `json:"cpu,omitempty"`
	Mem  *collector.MemStat  `json:"mem,omitempty"`
	Disk *collector.DiskStat `json:"disk,omitempty"`
	Net  *collector.NetStat  `json:"net,omitempty"`
	GPU  *collector.GPUStat  `json:"gpu,omitempty"`
	Proc *ProcOutput         `json:"proc,omitempty"`
}

// ProcOutput is a trimmed view of process data suitable for the command line.
type ProcOutput struct {
	Total   int                   `json:"total"`
	TopCPU  []collector.ProcInfo  `json:"top_cpu,omitempty"`
	TopMem  []collector.ProcInfo  `json:"top_mem,omitempty"`
	TopDisk []collector.ProcInfo  `json:"top_disk,omitempty"`
}

func printOutput(out Output, cfg Config) error {
	if cfg.JSON {
		return printJSON(out, cfg.Stream > 0)
	}
	printText(out, cfg)
	return nil
}

func printJSON(out Output, stream bool) error {
	var b []byte
	var err error
	if stream {
		b, err = json.Marshal(out)
	} else {
		b, err = json.MarshalIndent(out, "", "  ")
	}
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func printText(out Output, cfg Config) {
	if cfg.Stream > 0 {
		fmt.Printf("--- Snapshot at %s ---\n", out.Time.Format(time.RFC3339))
	}

	if out.CPU != nil {
		printCPU(*out.CPU)
	}
	if out.Mem != nil {
		printMem(*out.Mem)
	}
	if out.Disk != nil {
		printDisk(*out.Disk)
	}
	if out.GPU != nil {
		printGPU(*out.GPU)
	}
	if out.Net != nil {
		printNet(*out.Net)
	}
	if out.Proc != nil {
		printProc(*out.Proc)
	}

	if cfg.Stream > 0 {
		fmt.Println()
	}
}

func printCPU(c collector.CPUStat) {
	fmt.Printf("CPU:\n")
	if len(c.PerCore) == 0 {
		fmt.Println("  No CPU data")
		return
	}
	fmt.Printf("  Overall: %.1f%%\n", c.Overall)
	for i, v := range c.PerCore {
		fmt.Printf("  Core %d: %.1f%%\n", i, v)
	}
}

func printMem(m collector.MemStat) {
	fmt.Printf("Memory:\n")
	if m.Total == 0 {
		fmt.Println("  No memory data")
		return
	}
	usedPct := 0.0
	if m.Total > 0 {
		usedPct = float64(m.Used) / float64(m.Total) * 100
	}
	fmt.Printf("  Total: %s\n", formatBytes(m.Total))
	fmt.Printf("  Used:  %s (%.1f%%)\n", formatBytes(m.Used), usedPct)
	fmt.Printf("  Cached: %s\n", formatBytes(m.Cached))
	fmt.Printf("  Free:  %s\n", formatBytes(m.Free))
}

func printDisk(d collector.DiskStat) {
	fmt.Printf("Disk:\n")
	if len(d.Mounts) == 0 {
		fmt.Println("  No disk data")
		return
	}
	for _, m := range d.Mounts {
		fmt.Printf("  %s (%s):\n", m.Mountpoint, m.Fstype)
		fmt.Printf("    Used: %s / %s (%.1f%%)\n", formatBytes(m.Used), formatBytes(m.Total), m.UsedPercent)
		fmt.Printf("    Read:  %s/s\n", formatBytesFloat(m.ReadPerSec))
		fmt.Printf("    Write: %s/s\n", formatBytesFloat(m.WritePerSec))
	}
}

func printGPU(g collector.GPUStat) {
	fmt.Printf("GPU:\n")
	if !g.Available {
		msg := g.Message
		if msg == "" {
			msg = "No GPU data"
		}
		fmt.Printf("  %s\n", msg)
		return
	}
	if len(g.Cards) == 0 {
		fmt.Println("  No GPU detected")
		return
	}
	for _, c := range g.Cards {
		fmt.Printf("  %s:\n", c.Name)
		fmt.Printf("    Load: %s\n", formatPercent(c.LoadPct))
		if c.MemTotal > 0 {
			fmt.Printf("    Memory: %s / %s\n", formatBytes(c.MemUsed), formatBytes(c.MemTotal))
		} else if c.MemUsed > 0 {
			fmt.Printf("    Memory: %s\n", formatBytes(c.MemUsed))
		}
		fmt.Printf("    Temperature: %s\n", formatCelsius(c.TempC))
		fmt.Printf("    Power: %s\n", formatWatts(c.PowerW))
	}
}

func printNet(n collector.NetStat) {
	fmt.Printf("Network:\n")
	fmt.Printf("  Upload:   %s/s (total %s)\n", formatBytesFloat(n.UploadPerSec), formatBytes(n.TotalUpload))
	fmt.Printf("  Download: %s/s (total %s)\n", formatBytesFloat(n.DownloadPerSec), formatBytes(n.TotalDownload))
	if n.ProcsSupported && len(n.TopProcs) > 0 {
		fmt.Println("  Top processes by traffic:")
		for _, p := range n.TopProcs {
			fmt.Printf("    %-8d %-20s up: %s/s  down: %s/s\n",
				p.PID, truncate(p.Command, 20),
				formatBytesFloat(p.UploadPerSec),
				formatBytesFloat(p.DownloadPerSec))
		}
	}
}

func printProc(p ProcOutput) {
	fmt.Printf("Processes:\n")
	if p.Total == 0 {
		fmt.Println("  No process data")
		return
	}
	fmt.Printf("  Total: %d\n", p.Total)

	if len(p.TopCPU) > 0 {
		fmt.Println("  Top CPU:")
		for i, pr := range p.TopCPU {
			if i >= 10 {
				break
			}
			fmt.Printf("    %-8d %-24s CPU: %5.1f%%  MEM: %s\n",
				pr.PID, truncate(pr.Command, 24), pr.CPU, formatBytes(pr.MemRSS))
		}
	}

	if len(p.TopMem) > 0 {
		fmt.Println("  Top Memory:")
		for i, pr := range p.TopMem {
			if i >= 10 {
				break
			}
			fmt.Printf("    %-8d %-24s MEM: %s  CPU: %.1f%%\n",
				pr.PID, truncate(pr.Command, 24), formatBytes(pr.MemRSS), pr.CPU)
		}
	}
}

func formatBytes(b uint64) string {
	f := float64(b)
	switch {
	case b >= 1<<40:
		return fmt.Sprintf("%.2f TiB", f/(1<<40))
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GiB", f/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MiB", f/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KiB", f/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatBytesFloat(f float64) string {
	if f < 0 {
		return "N/A"
	}
	switch {
	case f >= 1<<40:
		return fmt.Sprintf("%.2f TiB", f/(1<<40))
	case f >= 1<<30:
		return fmt.Sprintf("%.2f GiB", f/(1<<30))
	case f >= 1<<20:
		return fmt.Sprintf("%.2f MiB", f/(1<<20))
	case f >= 1<<10:
		return fmt.Sprintf("%.2f KiB", f/(1<<10))
	default:
		return fmt.Sprintf("%.1f B", f)
	}
}

func formatCelsius(v float64) string {
	if v < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f°C", v)
}

func formatPercent(v float64) string {
	if v < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", v)
}

func formatWatts(v float64) string {
	if v < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f W", v)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
