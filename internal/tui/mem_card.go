package tui

import "xtop/internal/collector"

// memCard renders a used/cached/free segmented bar with a legend, mirroring the
// donut composition in MEM.png.
func memCard(m collector.MemStat, innerWidth, innerHeight int, focused bool, scroll *cardScroll) string {
	header := pillStyle.Render(fmtSize(m.Total))
	cw := contentWidth(innerWidth)

	var lines []string
	if m.Total == 0 {
		lines = []string{faintStyle.Render("无内存数据")}
		return renderCard(innerWidth, innerHeight, "▤", "内存", header, lines, focused, scroll)
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

	lines = []string{bar, "", legend, values}
	return renderCard(innerWidth, innerHeight, "▤", "内存", header, lines, focused, scroll)
}
