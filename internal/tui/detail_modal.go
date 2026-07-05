package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// detailBoxSize returns the inner dimensions of the detail popup.
func detailBoxSize(termW, termH int) (innerW, innerH int) {
	innerW = termW / 3
	if innerW > 48 {
		innerW = 48
	}
	if innerW < 30 {
		innerW = 30
	}
	innerH = 11 // title, divider, 6 info rows, 1 blank gap, 1 button row
	if innerH > termH-4 {
		innerH = termH - 4
	}
	if innerH < 8 {
		innerH = 8
	}
	return innerW, innerH
}

// overlayDetailModal composes the detail popup on top of the existing dashboard.
func overlayDetailModal(m *model, base string) string {
	width, height := m.width, m.height
	innerW, innerH := detailBoxSize(width, height)
	d := m.detail.proc

	rows := []string{
		center(titleStyle.Render(T("proc.detail.title")), innerW),
		divider(innerW),
	}

	info := []struct {
		label, value string
	}{
		{T("proc.detail.pid"), strconv.FormatInt(int64(d.PID), 10)},
		{T("proc.detail.command"), d.Command},
		{T("proc.detail.user"), d.User},
		{T("proc.detail.status"), d.Status},
		{T("proc.detail.cpu"), formatFloat1Pct(d.CPU)},
		{T("proc.detail.mem"), fmtSize(d.MemRSS)},
	}

	extraLabel, extraValue := detailExtra(d, m.detail.source)
	if extraLabel != "" {
		info = append(info, struct{ label, value string }{extraLabel, extraValue})
	}

	maxLabel := 0
	for _, it := range info {
		if w := lipgloss.Width(it.label); w > maxLabel {
			maxLabel = w
		}
	}

	for _, it := range info {
		line := labelStyle.Render(fitCell(it.label, maxLabel, false)) + " " +
			truncateStyled(textStyle.Render(it.value), innerW-maxLabel-2)
		rows = append(rows, padRow(line, innerW))
	}

	for len(rows) < innerH-1 {
		rows = append(rows, strings.Repeat(" ", innerW))
	}

	termLabel := T("proc.detail.terminate")
	killLabel := T("proc.detail.force_kill")
	termW := lipgloss.Width(termLabel)
	killW := lipgloss.Width(killLabel)
	gap := 4
	btnRowW := termW + 1 + killW + gap
	killX0 := (innerW - btnRowW) / 2
	forceX0 := killX0 + termW + 1 + gap

	termBtn := rowButtonStyle.Render(termLabel)
	killBtn := rowButtonDangerStyle.Render(killLabel)
	btnLine := strings.Repeat(" ", killX0) + termBtn + strings.Repeat(" ", gap) + killBtn
	btnLine = padRow(btnLine, innerW)
	rows = append(rows, btnLine)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colGreenDim).
		Width(innerW).
		Height(innerH).
		Render(strings.Join(rows, "\n"))

	boxW := lipgloss.Width(box)
	boxH := len(strings.Split(box, "\n"))
	boxLeft := (width - boxW) / 2
	boxTop := (height - boxH) / 2
	m.detail.killX0 = boxLeft + 1 + killX0
	m.detail.killX1 = m.detail.killX0 + lipgloss.Width(termBtn)
	m.detail.forceX0 = boxLeft + 1 + forceX0
	m.detail.forceX1 = m.detail.forceX0 + lipgloss.Width(killBtn)
	m.detail.btnY = boxTop + innerH

	if m.confirm.active {
		return overlayConfirmOnBase(m, base)
	}
	return overlayBox(base, box, boxLeft, boxTop)
}

// overlayBox pastes a rendered box onto a background at (left, top).
func overlayBox(base, box string, left, top int) string {
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")
	boxW := lipgloss.Width(boxLines[0])
	for i, bl := range boxLines {
		y := top + i
		if y < 0 || y >= len(baseLines) {
			continue
		}
		// Keep the original styled background line intact; only split it into
		// the visible regions that stay uncovered by the dialog.
		bg := baseLines[y]
		plain := stripANSI(bg)
		plainW := lipgloss.Width(plain)

		before := ""
		if left > 0 {
			if left >= plainW {
				before = bg
			} else {
				before = truncateStyled(bg, left)
			}
		}

		after := ""
		afterStart := left + boxW
		if afterStart < plainW {
			after = truncateStyledFrom(bg, afterStart)
		}
		baseLines[y] = before + bl + after
	}
	return strings.Join(baseLines, "\n")
}

// truncateStyledFrom returns the suffix of a styled string starting at visual
// column start, preserving ANSI styles.
func truncateStyledFrom(s string, start int) string {
	var out strings.Builder
	visible := 0
	inESC := false
	started := false
	for _, r := range s {
		if inESC {
			out.WriteRune(r)
			if r == 'm' {
				inESC = false
			}
			continue
		}
		if r == '\x1b' {
			out.WriteRune(r)
			inESC = true
			continue
		}
		rw := lipgloss.Width(string(r))
		if !started {
			if visible+rw > start {
				started = true
			} else {
				visible += rw
				continue
			}
		}
		out.WriteRune(r)
		visible += rw
	}
	return out.String()
}

// renderDetailModal is kept for any callers that expect the old full-screen API.
func renderDetailModal(m *model) string {
	return overlayDetailModal(m, m.dashBase())
}

// dashBase returns the current dashboard view without any overlays.
func (m *model) dashBase() string {
	lines := m.dashLines
	contentH := m.height - footerHeight
	if len(lines) > contentH {
		lines = lines[:contentH]
	}
	if len(lines) < contentH {
		for len(lines) < contentH {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n") + "\n" + m.dashFooter()
}

// detailExtra returns the card-specific metric label and value for the detail
// popup. Memory is already shown in the common info block, so it returns empty
// for the memory card.
func detailExtra(d detailProc, source cardKey) (label, value string) {
	switch source {
	case cardMem:
		return "", "" // memory already in common info
	case cardNet:
		return T("proc.detail.net"), fmtRate(d.NetUp) + " / " + fmtRate(d.NetDown)
	case cardDisk:
		return T("proc.detail.disk"), fmtRate(d.DiskRead) + " / " + fmtRate(d.DiskWrite)
	case cardGPU:
		return T("proc.detail.gpu"), fmtSize(d.GPUMem)
	}
	return "", ""
}

func center(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}
