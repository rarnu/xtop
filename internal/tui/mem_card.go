package tui

import "xtop/internal/collector"

// memCard renders a used/cached/free segmented bar with a legend (fixed), plus a
// scrollable list of the top processes by resident memory below it. The summary
// stays pinned; only the process list scrolls (feature 1).
func memCard(m collector.MemStat, procs []collector.ProcInfo, innerWidth, innerHeight int, focused bool, scroll *cardScroll, selected selectedProc) string {
	header := pillStyle.Render(fmtSize(m.Total))
	cw := contentWidth(innerWidth)

	if m.Total == 0 {
		return renderCard(innerWidth, innerHeight, "▤", T("card.mem"), header, []string{faintStyle.Render(T("no_data.mem"))}, focused, scroll)
	}

	total := float64(m.Total)
	bar := segBar(cw, []segPart{
		{frac: float64(m.Used) / total, color: colRed},
		{frac: float64(m.Cached) / total, color: colGrayLite},
		{frac: float64(m.Free) / total, color: colGreen},
	})

	colW := cw / 3
	legend := padRow(
		padRow(dot(colRed)+" "+labelStyle.Render(T("label.used")), colW)+
			padRow(dot(colGrayLite)+" "+labelStyle.Render(T("label.cached")), colW)+
			dot(colGreen)+" "+labelStyle.Render(T("label.free")),
		cw)
	values := padRow(
		padRow(valueStyle.Render(fmtSize(m.Used)), colW)+
			padRow(valueStyle.Render(fmtSize(m.Cached)), colW)+
			valueStyle.Render(fmtSize(m.Free)),
		cw)

	rows := make([]miniRow, 0, len(procs))
	for _, p := range procs {
		rows = append(rows, miniRow{Value: fmtSize(p.MemRSS), Command: p.Command, PID: p.PID})
	}

	fixed := []string{bar, "", legend, values, miniHeaderLine(cw, T("label.memory"), rows)}
	list := miniRowLines(cw, rows, selected)
	return renderCardSplit(innerWidth, innerHeight, "▤", T("card.mem"), header, fixed, list, focused, scroll)
}
