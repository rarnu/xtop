package tui

import "xtop/internal/collector"

// memCard renders a used/cached/free segmented bar with a legend (fixed), plus a
// scrollable list of the top processes by resident memory below it. The summary
// stays pinned; only the process list scrolls (feature 1).
func memCard(m collector.MemStat, procs []collector.ProcInfo, innerWidth, innerHeight int, focused bool, scroll *cardScroll) string {
	header := pillStyle.Render(fmtSize(m.Total))
	cw := contentWidth(innerWidth)

	if m.Total == 0 {
		return renderCard(innerWidth, innerHeight, "▤", "内存", header, []string{faintStyle.Render("无内存数据")}, focused, scroll)
	}

	total := float64(m.Total)
	bar := segBar(cw, []segPart{
		{frac: float64(m.Used) / total, color: colRed},
		{frac: float64(m.Cached) / total, color: colGrayLite},
		{frac: float64(m.Free) / total, color: colGreen},
	})

	colW := cw / 3
	legend := padRow(
		padRow(dot(colRed)+" "+labelStyle.Render("已用"), colW)+
			padRow(dot(colGrayLite)+" "+labelStyle.Render("缓存"), colW)+
			dot(colGreen)+" "+labelStyle.Render("空闲"),
		cw)
	values := padRow(
		padRow(valueStyle.Render(fmtSize(m.Used)), colW)+
			padRow(valueStyle.Render(fmtSize(m.Cached)), colW)+
			valueStyle.Render(fmtSize(m.Free)),
		cw)

	rows := make([]miniRow, 0, len(procs))
	for _, p := range procs {
		rows = append(rows, miniRow{Value: fmtSize(p.MemRSS), Command: p.Command})
	}

	fixed := []string{bar, "", legend, values, miniHeaderLine(cw, "内存")}
	list := miniRowLines(cw, rows)
	return renderCardSplit(innerWidth, innerHeight, "▤", "内存", header, fixed, list, focused, scroll)
}
