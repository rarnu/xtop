package tui

import (
	"fmt"
	"strconv"

	"xtop/internal/collector"
)

// cpuCard renders per-core meters plus an overall-utilisation sparkline.
func cpuCard(c collector.CPUStat, hist []float64, innerWidth, innerHeight int, focused bool, scroll *cardScroll) string {
	cw := contentWidth(innerWidth)
	sparkW := clampInt(cw/2, 8, maxInt(cw-8, 8))
	header := sparkline(sparkW, hist, colBlue)

	n := len(c.PerCore)
	if n == 0 {
		return renderCard(innerWidth, innerHeight, "▣", "CPU", header, []string{faintStyle.Render("无 CPU 数据")}, focused, scroll)
	}

	labelW := maxInt(len(strconv.Itoa(n-1)), 1)
	const pctW = 5 // "100.0"
	meterW := maxInt(cw-labelW-1-1-pctW, 1)

	lines := make([]string, 0, n)
	for i, v := range c.PerCore {
		label := labelStyle.Render(fmt.Sprintf("%*d", labelW, i))
		pct := valueStyle.Render(fmt.Sprintf("%5.1f", v))
		lines = append(lines, label+" "+meterBar(meterW, v)+" "+pct)
	}
	return renderCard(innerWidth, innerHeight, "▣", "CPU", header, lines, focused, scroll)
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
