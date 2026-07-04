package mcp

import (
	"fmt"
	"sort"
	"strings"

	"xtop/internal/collector"
)

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

func formatPercent(v float64) string {
	if v < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", v)
}

func formatSummary(s collector.Snapshot) string {
	var b strings.Builder
	b.WriteString("## System Summary\n\n")

	if len(s.CPU.PerCore) > 0 {
		fmt.Fprintf(&b, "- **CPU**: %.1f%% overall (%d cores)\n", s.CPU.Overall, len(s.CPU.PerCore))
	} else {
		b.WriteString("- **CPU**: no data\n")
	}

	if s.Mem.Total > 0 {
		usedPct := float64(s.Mem.Used) / float64(s.Mem.Total) * 100
		fmt.Fprintf(&b, "- **Memory**: %s used / %s total (%.1f%%)\n", formatBytes(s.Mem.Used), formatBytes(s.Mem.Total), usedPct)
	} else {
		b.WriteString("- **Memory**: no data\n")
	}

	if len(s.Disk.Mounts) > 0 {
		fmt.Fprintf(&b, "- **Disk**: %d mount(s), %s used / %s total\n", len(s.Disk.Mounts), formatBytes(s.Disk.UsedBytes), formatBytes(s.Disk.TotalBytes))
	} else {
		b.WriteString("- **Disk**: no data\n")
	}

	if s.GPU.Available && len(s.GPU.Cards) > 0 {
		fmt.Fprintf(&b, "- **GPU**: %d card(s)\n", len(s.GPU.Cards))
	} else {
		b.WriteString("- **GPU**: " + strings.TrimSpace(s.GPU.Message) + "\n")
	}

	fmt.Fprintf(&b, "- **Network**: up %s/s, down %s/s\n", formatBytesFloat(s.Net.UploadPerSec), formatBytesFloat(s.Net.DownloadPerSec))

	if len(s.Proc.Top) > 0 {
		fmt.Fprintf(&b, "- **Processes**: %d total; top CPU is %s (%.1f%%)\n", len(s.Proc.All), s.Proc.Top[0].Command, s.Proc.Top[0].CPU)
	} else {
		fmt.Fprintf(&b, "- **Processes**: %d total\n", len(s.Proc.All))
	}

	return b.String()
}

func formatCPU(s collector.Snapshot) string {
	if len(s.CPU.PerCore) == 0 {
		return "No CPU data available."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Overall: %s\n", formatPercent(s.CPU.Overall))
	for i, v := range s.CPU.PerCore {
		fmt.Fprintf(&b, "Core %d: %s\n", i, formatPercent(v))
	}
	return b.String()
}

func formatMem(s collector.Snapshot) string {
	if s.Mem.Total == 0 {
		return "No memory data available."
	}
	usedPct := float64(s.Mem.Used) / float64(s.Mem.Total) * 100
	var b strings.Builder
	fmt.Fprintf(&b, "Total: %s\n", formatBytes(s.Mem.Total))
	fmt.Fprintf(&b, "Used:  %s (%.1f%%)\n", formatBytes(s.Mem.Used), usedPct)
	fmt.Fprintf(&b, "Cached: %s\n", formatBytes(s.Mem.Cached))
	fmt.Fprintf(&b, "Free:  %s\n", formatBytes(s.Mem.Free))
	return b.String()
}

func formatDisk(s collector.Snapshot) string {
	if len(s.Disk.Mounts) == 0 {
		return "No disk data available."
	}
	var b strings.Builder
	for _, m := range s.Disk.Mounts {
		fmt.Fprintf(&b, "%s (%s): %s / %s (%.1f%%), read %s/s, write %s/s\n",
			m.Mountpoint, m.Fstype,
			formatBytes(m.Used), formatBytes(m.Total), m.UsedPercent,
			formatBytesFloat(m.ReadPerSec), formatBytesFloat(m.WritePerSec))
	}
	return b.String()
}

func formatGPU(s collector.Snapshot) string {
	if !s.GPU.Available || len(s.GPU.Cards) == 0 {
		msg := s.GPU.Message
		if msg == "" {
			msg = "No GPU data available."
		}
		return msg
	}
	var b strings.Builder
	for _, c := range s.GPU.Cards {
		fmt.Fprintf(&b, "%s:\n", c.Name)
		load := "N/A"
		if c.LoadPct >= 0 {
			load = fmt.Sprintf("%.1f%%", c.LoadPct)
		}
		fmt.Fprintf(&b, "  Load: %s\n", load)
		if c.MemTotal > 0 {
			fmt.Fprintf(&b, "  VRAM: %s / %s\n", formatBytes(c.MemUsed), formatBytes(c.MemTotal))
		} else if c.MemUsed > 0 {
			fmt.Fprintf(&b, "  VRAM: %s\n", formatBytes(c.MemUsed))
		}
		if c.PowerW >= 0 {
			if c.TempC >= 0 {
				fmt.Fprintf(&b, "  Power: %.0f W (%.0f °C)\n", c.PowerW, c.TempC)
			} else {
				fmt.Fprintf(&b, "  Power: %.0f W\n", c.PowerW)
			}
		} else if c.TempC >= 0 {
			fmt.Fprintf(&b, "  Temperature: %.0f °C\n", c.TempC)
		}
	}
	return b.String()
}

func formatNet(s collector.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Upload:   %s/s (total %s)\n", formatBytesFloat(s.Net.UploadPerSec), formatBytes(s.Net.TotalUpload))
	fmt.Fprintf(&b, "Download: %s/s (total %s)\n", formatBytesFloat(s.Net.DownloadPerSec), formatBytes(s.Net.TotalDownload))
	return b.String()
}

func formatProc(p collector.ProcStat, topBy string, limit int) string {
	if len(p.All) == 0 {
		return "No process data available."
	}
	var list []collector.ProcInfo
	switch topBy {
	case "mem":
		list = p.TopMem
	case "disk":
		list = p.TopDisk
	default:
		list = p.Top
	}
	if limit > len(list) {
		limit = len(list)
	}
	if limit < 0 {
		limit = 0
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Total processes: %d\n", len(p.All))
	if limit == 0 {
		return b.String()
	}
	fmt.Fprintf(&b, "Top %d by %s:\n", limit, topBy)
	for i := 0; i < limit; i++ {
		pr := list[i]
		switch topBy {
		case "mem":
			fmt.Fprintf(&b, "  %-8d %-24s MEM: %s  CPU: %.1f%%\n", pr.PID, truncate(pr.Command, 24), formatBytes(pr.MemRSS), pr.CPU)
		case "disk":
			fmt.Fprintf(&b, "  %-8d %-24s DISK: %s/s  CPU: %.1f%%\n", pr.PID, truncate(pr.Command, 24), formatBytesFloat(pr.DiskReadPerSec+pr.DiskWritePerSec), pr.CPU)
		default:
			fmt.Fprintf(&b, "  %-8d %-24s CPU: %.1f%%  MEM: %s\n", pr.PID, truncate(pr.Command, 24), pr.CPU, formatBytes(pr.MemRSS))
		}
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// topGPUProcs returns a sorted copy of GPU top procs by VRAM, descending.
func topGPUProcs(procs []collector.GPUProc, limit int) []collector.GPUProc {
	out := append([]collector.GPUProc(nil), procs...)
	sort.Slice(out, func(i, j int) bool { return out[i].MemBytes > out[j].MemBytes })
	if limit > len(out) {
		limit = len(out)
	}
	if limit < 0 {
		limit = 0
	}
	return out[:limit]
}
