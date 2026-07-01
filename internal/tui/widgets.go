package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---- number formatting -------------------------------------------------

var sizeUnits = []string{"B", "K", "M", "G", "T", "P"}
var rateUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// fmtSize renders a byte count with a single-letter unit, e.g. "6.8 T".
func fmtSize(v uint64) string { return scale(float64(v), sizeUnits, "") }

// fmtSizeF is fmtSize for a float input (used for aggregate sums).
func fmtSizeF(v float64) string { return scale(v, sizeUnits, "") }

// fmtBytesF renders a byte value with a two-letter unit and no rate suffix,
// e.g. "0.0 B", "2.1 MB" (the caller's label carries any "/s").
func fmtBytesF(v float64) string { return scale(v, rateUnits, "") }

// fmtRate renders a throughput, e.g. "2.1 MB/s".
func fmtRate(v float64) string { return scale(v, rateUnits, "/s") }

func scale(v float64, units []string, suffix string) string {
	if v < 0 {
		v = 0
	}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s%s", v, units[i], suffix)
}

// ---- bars & meters -----------------------------------------------------

func clampPct(p float64) float64 {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// meterBar renders a CPU-style meter of vertical ticks: filled ticks coloured by
// utilisation level, the remainder in a dark track. (see CPU.png)
func meterBar(width int, pct float64) string {
	if width < 1 {
		width = 1
	}
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled > width {
		filled = width
	}
	on := lipgloss.NewStyle().Foreground(levelColor(pct)).Render(strings.Repeat("│", filled))
	off := lipgloss.NewStyle().Foreground(colTrack).Render(strings.Repeat("│", width-filled))
	return on + off
}

// blockBar renders a solid horizontal bar ('█') filled to pct, coloured by level,
// with a dark track behind. (disk usage, gpu memory)
func blockBar(width int, pct float64) string {
	if width < 1 {
		width = 1
	}
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled > width {
		filled = width
	}
	on := lipgloss.NewStyle().Foreground(levelColor(pct)).Render(strings.Repeat("█", filled))
	off := lipgloss.NewStyle().Foreground(colTrack).Render(strings.Repeat("─", width-filled))
	return on + off
}

// lineGauge renders a thin rounded gauge used by GPU temperature/load. (GPU.png)
func lineGauge(width int, pct float64) string {
	if width < 2 {
		width = 2
	}
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled > width {
		filled = width
	}
	on := lipgloss.NewStyle().Foreground(levelColor(pct)).Render(strings.Repeat("━", filled))
	off := lipgloss.NewStyle().Foreground(colGray).Render(strings.Repeat("━", width-filled))
	return on + off
}

// segPart is one coloured slice of a segmented bar.
type segPart struct {
	frac  float64 // fraction of the whole (0-1)
	color lipgloss.Color
}

// segBar renders proportional coloured segments across width cells. Used for the
// memory used/cached/free composition. (MEM.png)
func segBar(width int, parts []segPart) string {
	if width < 1 {
		width = 1
	}
	var b strings.Builder
	used := 0
	for i, p := range parts {
		n := int(math.Round(p.frac * float64(width)))
		if i == len(parts)-1 {
			n = width - used // last segment fills remainder to avoid rounding gaps
		}
		if n < 0 {
			n = 0
		}
		if used+n > width {
			n = width - used
		}
		if n > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(p.color).Render(strings.Repeat("█", n)))
			used += n
		}
	}
	return b.String()
}

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

// sparkline renders the last `width` samples as a coloured block sparkline,
// auto-scaled to the maximum sample. (chart strip at the top of cards)
func sparkline(width int, data []float64, color lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	if len(data) == 0 {
		return lipgloss.NewStyle().Foreground(colTrack).Render(strings.Repeat("▁", width))
	}
	if len(data) > width {
		data = data[len(data)-width:]
	}
	max := 0.0
	for _, v := range data {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		max = 1
	}
	var b strings.Builder
	// left-pad with empty track when fewer samples than width
	if pad := width - len(data); pad > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colTrack).Render(strings.Repeat("▁", pad)))
	}
	style := lipgloss.NewStyle().Foreground(color)
	for _, v := range data {
		idx := int(v / max * float64(len(sparkRunes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkRunes) {
			idx = len(sparkRunes) - 1
		}
		b.WriteString(style.Render(string(sparkRunes[idx])))
	}
	return b.String()
}

var tankFrame = lipgloss.NewStyle().Foreground(colGreenDim)

// vTank renders a small vertical "tank" filled from the bottom to pct, returning
// innerRows+2 lines (rounded frame included). (disk cards in DISK.png)
func vTank(innerRows int, pct float64) []string {
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(innerRows)))
	col := lipgloss.NewStyle().Foreground(levelColor(pct))

	lines := make([]string, 0, innerRows+2)
	lines = append(lines, tankFrame.Render("╭─╮"))
	for r := 0; r < innerRows; r++ {
		rowFromBottom := innerRows - 1 - r
		cell := " "
		if rowFromBottom < filled {
			cell = col.Render("█")
		}
		lines = append(lines, tankFrame.Render("│")+cell+tankFrame.Render("│"))
	}
	lines = append(lines, tankFrame.Render("╰─╯"))
	return lines
}

// dot returns a coloured bullet used in legends.
func dot(color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render("●")
}
