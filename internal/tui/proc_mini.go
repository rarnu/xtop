package tui

// miniRow is one entry in a card's process mini-list: a formatted metric value
// and the process command.
type miniRow struct {
	Value   string
	Command string
}

// miniValW returns the width of the value column for a process mini-list, sized
// to fit values like "123.4 MB/s" while never taking more than half the card.
func miniValW(cw int) int {
	w := 10
	if half := cw / 2; w > half {
		w = half
	}
	if w < 4 {
		w = 4
	}
	return w
}

// miniHeaderLine is the single column-header line ("<value>  进程") that labels a
// process mini-list. In split cards it stays fixed above the scrollable rows; in
// whole-card cards it simply leads the appended rows.
func miniHeaderLine(cw int, valueHead string) string {
	valW := miniValW(cw)
	cmdW := maxInt(cw-valW-1, 1)
	return labelStyle.Render(fitCell(valueHead, valW, true)) + " " + labelStyle.Render(fitCell("进程", cmdW, false))
}

// miniRowLines renders the data rows of a process mini-list at content width cw.
func miniRowLines(cw int, rows []miniRow) []string {
	if len(rows) == 0 {
		return []string{faintStyle.Render("无进程数据")}
	}
	valW := miniValW(cw)
	cmdW := maxInt(cw-valW-1, 1)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out,
			valueStyle.Render(fitCell(r.Value, valW, true))+" "+
				textStyle.Render(fitCell(r.Command, cmdW, false)))
	}
	return out
}

// unsupportedLines is the placeholder shown when a per-process metric isn't
// obtainable on the current platform.
func unsupportedLines() []string {
	return []string{faintStyle.Render("进程列表: 本平台不支持")}
}
