package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"xtop/internal/collector"
)

var gpuNameStyle = lipgloss.NewStyle().Foreground(colGreen).Bold(true)

// gpuCard renders one block per GPU (power / memory / temperature / load), or a
// graceful message when no GPU data source is available. (GPU.png)
func gpuCard(g collector.GPUStat, hist []float64, innerWidth, innerHeight int, focused bool, scroll *cardScroll) string {
	cw := contentWidth(innerWidth)
	sparkW := clampInt(cw/2, 8, maxInt(cw-8, 8))
	header := sparkline(sparkW, hist, colBlue)

	if !g.Available || len(g.Cards) == 0 {
		msg := g.Message
		if msg == "" {
			msg = "无 GPU 数据"
		}
		return renderCard(innerWidth, innerHeight, "◉", "GPU", header, []string{faintStyle.Render(msg)}, focused, scroll)
	}

	var lines []string
	for i, c := range g.Cards {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, gpuNameStyle.Render(truncPlain(c.Name, cw)))
		lines = append(lines, joinLR(labelStyle.Render("功耗"), valueStyle.Render(gpuPower(c.PowerW)), cw))
		lines = append(lines, joinLR(labelStyle.Render("内存"), valueStyle.Render(gpuMem(c)), cw))
		if c.MemTotal > 0 {
			lines = append(lines, blockBar(cw, float64(c.MemUsed)/float64(c.MemTotal)*100))
		}
		lines = append(lines, gaugeRow("温度", c.TempC, gpuTemp(c.TempC), cw))
		lines = append(lines, gaugeRow("负载", c.LoadPct, gpuLoad(c.LoadPct), cw))
	}

	// Process list (top by GPU memory) appended below the card stats.
	lines = append(lines, "")
	if !g.ProcsSupported {
		lines = append(lines, unsupportedLines()...)
	} else {
		lines = append(lines, miniHeaderLine(cw, "显存"))
		rows := make([]miniRow, 0, len(g.TopProcs))
		for _, p := range g.TopProcs {
			rows = append(rows, miniRow{Value: fmtSize(p.MemBytes), Command: p.Command})
		}
		lines = append(lines, miniRowLines(cw, rows)...)
	}

	return renderCard(innerWidth, innerHeight, "◉", "GPU", header, lines, focused, scroll)
}

// gaugeRow renders: label + gauge + right-aligned value. gaugePct <0 draws an
// empty gauge (value already shows N/A).
func gaugeRow(label string, gaugePct float64, value string, innerWidth int) string {
	labelW := 5
	valW := 7
	gw := maxInt(innerWidth-labelW-1-valW, 2)
	p := gaugePct
	if p < 0 {
		p = 0
	}
	return labelStyle.Render(fitCell(label, labelW, false)) + " " +
		lineGauge(gw, p) + valueStyle.Render(fitCell(value, valW, true))
}

func gpuPower(w float64) string {
	if w < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.0f W", w)
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

func gpuTemp(t float64) string {
	if t < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.0f °C", t)
}

func gpuLoad(l float64) string {
	if l < 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.0f %%", l)
}
