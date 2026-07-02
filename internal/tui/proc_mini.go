package tui

import "github.com/charmbracelet/lipgloss"

// miniRow is one entry in a card's process mini-list: a formatted metric value
// and the process command. The optional Up/Down fields are used by the network
// card to show per-process traffic in both directions.
type miniRow struct {
	Value     string
	UpValue   string
	DownValue string
	Command   string
	TwoCol    bool // if true, render UpValue and DownValue side by side
}

// miniValW returns the width of the value column for a process mini-list, sized
// to fit the header text and the widest value in rows, capped at half the card.
func miniValW(cw int, valueHead string, rows []miniRow) int {
	w := lipgloss.Width(valueHead)
	for _, r := range rows {
		if v := lipgloss.Width(r.Value); v > w {
			w = v
		}
	}
	if w < 4 {
		w = 4
	}
	if half := cw / 2; w > half {
		w = half
	}
	return w
}

// miniHeaderLine is the single column-header line ("<value>  进程") that labels a
// process mini-list. In split cards it stays fixed above the scrollable rows; in
// whole-card cards it simply leads the appended rows.
func miniHeaderLine(cw int, valueHead string, rows []miniRow) string {
	valW := miniValW(cw, valueHead, rows)
	cmdW := maxInt(cw-valW-1, 1)
	return labelStyle.Render(fitCell(valueHead, valW, false)) + " " + labelStyle.Render(fitCell("进程", cmdW, false))
}

// miniRowLines renders the data rows of a process mini-list at content width cw.
func miniRowLines(cw int, rows []miniRow) []string {
	if len(rows) == 0 {
		return []string{faintStyle.Render("无进程数据")}
	}

	// Two-column network rows: size the value columns to the widest actual value
	// so rates like "1.23 MB/s" are not truncated, while leaving the rest for the
	// process command.
	if rows[0].TwoCol {
		minValW := 6
		maxValW := maxInt(cw/3, minValW)
		upW, downW := minValW, minValW
		for _, r := range rows {
			upW = maxInt(upW, lipgloss.Width(r.UpValue))
			downW = maxInt(downW, lipgloss.Width(r.DownValue))
		}
		upW = minInt(upW, maxValW)
		downW = minInt(downW, maxValW)
		cmdW := maxInt(cw-upW-downW-2, 1)
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			out = append(out,
				valueStyle.Render(fitCell(r.UpValue, upW, false))+" "+
					valueStyle.Render(fitCell(r.DownValue, downW, false))+" "+
					textStyle.Render(fitCell(r.Command, cmdW, false)))
		}
		return out
	}

	// Single-column rows (mem, disk, gpu): size the value column to fit the
	// header text and the widest value, leaving the rest for the command.
	valW := miniValW(cw, "", rows)
	cmdW := maxInt(cw-valW-1, 1)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out,
			valueStyle.Render(fitCell(r.Value, valW, true))+" "+
				textStyle.Render(fitCell(r.Command, cmdW, false)))
	}
	return out
}

// miniTwoColHeaderLine renders a header like "上传  下载  进程" for the network
// card's split value column, matching the row widths computed by miniRowLines.
func miniTwoColHeaderLine(cw int, rows []miniRow) string {
	if len(rows) == 0 {
		return labelStyle.Render("上传") + " " + labelStyle.Render("下载") + " " + labelStyle.Render("进程")
	}
	minValW := 6
	maxValW := maxInt(cw/3, minValW)
	upW, downW := minValW, minValW
	for _, r := range rows {
		upW = maxInt(upW, lipgloss.Width(r.UpValue))
		downW = maxInt(downW, lipgloss.Width(r.DownValue))
	}
	upW = minInt(upW, maxValW)
	downW = minInt(downW, maxValW)
	cmdW := maxInt(cw-upW-downW-2, 1)
	return labelStyle.Render(fitCell("上传", upW, false)) + " " +
		labelStyle.Render(fitCell("下载", downW, false)) + " " +
		labelStyle.Render(fitCell("进程", cmdW, false))
}

// unsupportedLines is no longer used: every card simply shows "无进程数据" when
// per-process metrics are unavailable. Kept as a thin alias to avoid churn in
// any code that might still reference it.
func unsupportedLines() []string { return []string{faintStyle.Render("无进程数据")} }
