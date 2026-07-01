package tui

import (
	"fmt"
	"strconv"

	"xtop/internal/collector"
)

const (
	openProcLabel  = "打开进程管理"
	openProcNeedle = "打开进程管理"
)

// procCard renders the top-N processes by CPU plus the button that opens the
// full process manager (feature 6).
func procCard(p collector.ProcStat, innerWidth, innerHeight int, focused bool, scroll *cardScroll) string {
	cw := contentWidth(innerWidth)
	pidW, userW, cpuW, memW := 7, 9, 8, 8
	cmdW := maxInt(cw-pidW-userW-cpuW-memW, 6)

	head := labelStyle.Render(
		fitCell("PID", pidW, false) +
			fitCell("USER", userW, false) +
			fitCell("CPU", cpuW, false) +
			fitCell("MEM", memW, false) +
			fitCell("CMD", cmdW, false))

	lines := []string{head}
	for _, pr := range p.Top {
		row := valueStyle.Render(fitCell(strconv.Itoa(int(pr.PID)), pidW, false)) +
			textStyle.Render(fitCell(pr.User, userW, false)) +
			lipglossPct(pr.CPU, cpuW) +
			textStyle.Render(fitCell(fmtSize(pr.MemRSS), memW, false)) +
			textStyle.Render(fitCell(pr.Command, cmdW, false))
		lines = append(lines, row)
	}
	if len(p.Top) == 0 {
		lines = append(lines, faintStyle.Render("无进程数据"))
	}

	btn := rowButtonStyle.Render(openProcLabel)
	return renderCard(innerWidth, innerHeight, "☰", "进程", btn, lines, focused, scroll)
}

// lipglossPct renders a CPU percentage coloured by load level.
func lipglossPct(pct float64, width int) string {
	txt := fmt.Sprintf("%.1f%%", pct)
	return lipglossFg(levelColor(pct)).Render(fitCell(txt, width, false))
}
