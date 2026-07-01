package tui

import "xtop/internal/collector"

// memCard renders a used/cached/free segmented bar with a legend, mirroring the
// donut composition in MEM.png.
func memCard(m collector.MemStat, innerWidth int, focused bool) string {
	header := pillStyle.Render(fmtSize(m.Total))

	var lines []string
	if m.Total == 0 {
		lines = []string{faintStyle.Render("无内存数据")}
		return renderCard(innerWidth, "▤", "内存", header, lines, focused)
	}

	total := float64(m.Total)
	bar := segBar(innerWidth, []segPart{
		{frac: float64(m.Used) / total, color: colRed},
		{frac: float64(m.Cached) / total, color: colGrayLite},
		{frac: float64(m.Free) / total, color: colGreen},
	})

	colW := innerWidth / 3
	legend := padRow(
		padRow(dot(colRed)+" "+labelStyle.Render("已用"), colW)+
			padRow(dot(colGrayLite)+" "+labelStyle.Render("缓存"), colW)+
			dot(colGreen)+" "+labelStyle.Render("空闲"),
		innerWidth)
	values := padRow(
		padRow(valueStyle.Render(fmtSize(m.Used)), colW)+
			padRow(valueStyle.Render(fmtSize(m.Cached)), colW)+
			valueStyle.Render(fmtSize(m.Free)),
		innerWidth)

	lines = []string{bar, "", legend, values}
	return renderCard(innerWidth, "▤", "内存", header, lines, focused)
}
