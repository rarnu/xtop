package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"xtop/internal/collector"
)

// sortCol identifies the active process-table sort column.
type sortCol int

const (
	sortPID sortCol = iota
	sortUser
	sortStat
	sortCPU
	sortMem
	sortStart
	sortCmd
)

var sortTitles = map[sortCol]string{
	sortPID:   "PID",
	sortUser:  "USER",
	sortStat:  "STAT",
	sortCPU:   "CPU",
	sortMem:   "MEM",
	sortStart: "START",
	sortCmd:   "CMD",
}

// column order left-to-right.
var sortOrder = []sortCol{sortPID, sortUser, sortStat, sortCPU, sortMem, sortStart, sortCmd}

const actionW = 22 // width reserved on the right for the two row buttons

var selRowStyle = lipgloss.NewStyle().Background(lipgloss.Color("#0F3D24")).Foreground(colGreenHi)

// procColumn is a laid-out header column with its absolute x and width.
type procColumn struct {
	key   sortCol
	x, w  int
	title string
}

// procColumns computes the on-screen layout for the given total width. It also
// returns the CMD column geometry and the actions column x/width.
func procColumns(width int) (cols []procColumn, actX int) {
	widths := map[sortCol]int{
		sortPID:   8,
		sortUser:  11,
		sortStat:  6,
		sortCPU:   9,
		sortMem:   9,
		sortStart: 18,
	}
	x := 2 // selection gutter
	for _, k := range sortOrder {
		if k == sortCmd {
			continue
		}
		cols = append(cols, procColumn{key: k, x: x, w: widths[k], title: sortTitles[k]})
		x += widths[k]
	}
	actX = width - actionW
	cmdW := actX - x
	if cmdW < 8 {
		cmdW = 8
	}
	cols = append(cols, procColumn{key: sortCmd, x: x, w: cmdW, title: sortTitles[sortCmd]})
	return cols, actX
}

// actionHit returns the x spans of the KILL and FORCE-KILL buttons within a row.
func actionHit(actX int) (termX0, termX1, killX0, killX1 int) {
	wTerm := lipgloss.Width("结束")
	wKill := lipgloss.Width("强制结束")
	termX0 = actX
	termX1 = termX0 + wTerm
	killX0 = termX1 + 1
	killX1 = killX0 + wKill
	return
}

func sortProcs(list []collector.ProcInfo, col sortCol, asc bool) {
	sort.SliceStable(list, func(i, j int) bool {
		var less bool
		a, b := list[i], list[j]
		switch col {
		case sortPID:
			less = a.PID < b.PID
		case sortUser:
			less = a.User < b.User
		case sortStat:
			less = a.Status < b.Status
		case sortCPU:
			less = a.CPU < b.CPU
		case sortMem:
			less = a.MemRSS < b.MemRSS
		case sortStart:
			less = a.Start.Before(b.Start)
		case sortCmd:
			less = a.Command < b.Command
		}
		if asc {
			return less
		}
		return !less
	})
}

func fmtStart(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("01-02 15:04:05")
}

// renderProcModal draws the full-screen process manager (feature 7).
func renderProcModal(m *model) string {
	width, height := m.width, m.height
	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}
	cols, actX := procColumns(width)
	visible := maxInt(height-3, 1)

	// Title bar
	title := titleStyle.Render("XTOP") + faintStyle.Render(" · 进程管理")
	hint := faintStyle.Render(fmt.Sprintf("共 %d 进程", len(m.procRows)))
	lines := []string{joinLR(title, hint, width)}

	// Header row with sort indicators
	var hb strings.Builder
	hb.WriteString("  ")
	for _, c := range cols {
		t := c.title
		if c.key == m.sortCol {
			if m.sortAsc {
				t += " ▲"
			} else {
				t += " ▼"
			}
		}
		st := labelStyle
		if c.key == m.sortCol {
			st = lipglossFg(colGreenHi)
		}
		hb.WriteString(st.Render(fitCell(t, c.w, false)))
	}
	hb.WriteString(labelStyle.Render(fitCell("操作", actionW, false)))
	lines = append(lines, padRow(hb.String(), width))

	// Body rows
	term := rowButtonStyle.Render("结束")
	kill := rowButtonDangerStyle.Render("强制结束")
	actions := padRow(term+" "+kill, actionW)

	for i := 0; i < visible; i++ {
		idx := m.procScroll + i
		if idx >= len(m.procRows) {
			lines = append(lines, padRow("", width))
			continue
		}
		pr := m.procRows[idx]
		selected := idx == m.procSel
		lines = append(lines, renderProcRow(pr, cols, actions, selected, width, actX))
	}

	// Footer help
	help := helpBarStyle.Render("↑/↓ 选择 · 1-7/点击表头 排序 · k 结束 · K/f 强制结束 · ESC 返回")
	lines = append(lines, padRow(help, width))

	frame := strings.Join(lines, "\n")

	if m.confirm.active {
		return overlayConfirm(m, width, height)
	}
	return frame
}

func renderProcRow(pr collector.ProcInfo, cols []procColumn, actions string, selected bool, width, actX int) string {
	cells := make([]string, 0, len(cols))
	for _, c := range cols {
		var v string
		switch c.key {
		case sortPID:
			v = strconv.Itoa(int(pr.PID))
		case sortUser:
			v = pr.User
		case sortStat:
			v = pr.Status
		case sortCPU:
			v = fmt.Sprintf("%.1f%%", pr.CPU)
		case sortMem:
			v = fmtSize(pr.MemRSS)
		case sortStart:
			v = fmtStart(pr.Start)
		case sortCmd:
			v = pr.Command
		}
		cells = append(cells, fitCell(v, c.w, false))
	}
	body := strings.Join(cells, "")

	if selected {
		gutter := lipglossFg(colGreenHi).Render(">") + " "
		return padRow(gutter+selRowStyle.Render(body)+actions, width)
	}
	return padRow("  "+textStyle.Render(body)+actions, width)
}

// overlayConfirmOnBase renders the confirmation dialog over the existing screen.
func overlayConfirmOnBase(m *model, base string) string {
	kind := "结束"
	danger := false
	if m.confirm.force {
		kind = "强制结束"
		danger = true
	}
	titleC := colGreenHi
	if danger {
		titleC = colRed
	}

	q := fmt.Sprintf("确认%s进程?", kind)
	info := fmt.Sprintf("PID %d  %s", m.confirm.pid, truncPlain(m.confirm.name, 40))
	keys := faintStyle.Render("y 确认   n/ESC 取消")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(titleC).
		Padding(1, 3).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			lipglossFg(titleC).Bold(true).Render(q),
			"",
			textStyle.Render(info),
			"",
			keys,
		))

	boxW := lipgloss.Width(box)
	boxH := len(strings.Split(box, "\n"))
	left := (m.width - boxW) / 2
	top := (m.height - boxH) / 2
	return overlayBox(base, box, left, top)
}

// overlayConfirm is kept for the old full-screen confirmation API.
func overlayConfirm(m *model, width, height int) string {
	return overlayConfirmOnBase(m, strings.Repeat("\n", height-1))
}
