package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtop/internal/collector"
)

var gpuNameStyle = lipgloss.NewStyle().Foreground(colGreen).Bold(true)

// gpuCard renders one block per GPU (power / VRAM / load), or a graceful message
// when no GPU data source is available. (GPU.png)
func gpuCard(g collector.GPUStat, hist []float64, innerWidth, innerHeight int, focused bool, scroll *cardScroll, selected selectedProc) string {
	cw := contentWidth(innerWidth)
	sparkW := clampInt(cw/2, 8, maxInt(cw-8, 8))
	header := sparkline(sparkW, hist, colBlue)

	if !g.Available || len(g.Cards) == 0 {
		msg := g.Message
		if msg == "" {
			msg = T("no_data.gpu")
		}
		return renderCard(innerWidth, innerHeight, "◉", T("card.gpu"), header, []string{faintStyle.Render(msg)}, focused, scroll)
	}

	var lines []string
	for i, c := range g.Cards {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, gpuNameStyle.Render(truncPlain(c.Name, cw)))
		lines = append(lines, joinLR(labelStyle.Render(T("label.power")), valueStyle.Render(gpuPowerTemp(c)), cw))
		memPct := -1.0
		if c.MemTotal > 0 {
			memPct = float64(c.MemUsed) / float64(c.MemTotal) * 100
		}
		lines = append(lines, gaugeRow(T("label.gpu_mem"), memPct, gpuMem(c), cw, 15))
		lines = append(lines, gaugeRow(T("label.load"), c.LoadPct, gpuLoad(c.LoadPct), cw, 15))
	}

	// Process list (top by GPU memory) appended below the card stats.
	lines = append(lines, "")
	if !g.ProcsSupported {
		lines = append(lines, miniHeaderLine(cw, T("label.gpu_mem"), nil))
		lines = append(lines, faintStyle.Render(T("no_data.net_procs")))
	} else {
		rows := make([]miniRow, 0, len(g.TopProcs))
		for _, p := range g.TopProcs {
			rows = append(rows, miniRow{Value: fmtSize(p.MemBytes), Command: p.Command, PID: p.PID})
		}
		lines = append(lines, miniHeaderLine(cw, T("label.gpu_mem"), rows))
		lines = append(lines, miniRowLines(cw, rows, selected)...)
	}

	return renderCard(innerWidth, innerHeight, "◉", T("card.gpu"), header, lines, focused, scroll)
}

// gaugeRow renders: label + gauge + right-aligned value. gaugePct <0 draws an
// empty gauge (value already shows N/A).
func gaugeRow(label string, gaugePct float64, value string, innerWidth, valW int) string {
	labelW := 5
	gap := 2 // two-cell spacing between gauge and value
	gw := maxInt(innerWidth-labelW-1-gap-valW, 2)
	p := gaugePct
	if p < 0 {
		p = 0
	}
	return labelStyle.Render(fitCell(label, labelW, false)) + " " +
		lineGauge(gw, p) + spaces(gap) + valueStyle.Render(fitCell(value, valW, true))
}

func gpuPowerTemp(c collector.GPUCard) string {
	var parts []string
	if c.PowerW >= 0 {
		parts = append(parts, fmt.Sprintf("%.0f W", c.PowerW))
	}
	if c.TempC >= 0 {
		parts = append(parts, fmt.Sprintf("(%.0f ℃)", c.TempC))
	}
	if len(parts) == 0 {
		return T("na")
	}
	return strings.Join(parts, "")
}

func gpuMem(c collector.GPUCard) string {
	if c.MemTotal == 0 {
		if c.MemUsed > 0 {
			return fmtSize(c.MemUsed)
		}
		return "N/A"
	}
	return fmtSize(c.MemUsed) + " / " + fmtSize(c.MemTotal)
}

func gpuLoad(l float64) string {
	if l < 0 {
		return T("na")
	}
	return fmt.Sprintf("%.0f %%", l)
}
