package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// detailBoxSize returns the inner dimensions of the detail popup.
func detailBoxSize(termW, termH int) (innerW, innerH int) {
	innerW = termW - 8
	if innerW > 70 {
		innerW = 70
	}
	if innerW < 30 {
		innerW = 30
	}
	innerH = termH - 6
	if innerH > 14 {
		innerH = 14
	}
	if innerH < 8 {
		innerH = 8
	}
	return innerW, innerH
}

// renderDetailModal draws the centered process-detail popup with KILL / FORCE
// KILL buttons. It also records the button hit boxes in m.detail.
func renderDetailModal(m *model) string {
	width, height := m.width, m.height
	innerW, innerH := detailBoxSize(width, height)
	d := m.detail.proc

	rows := []string{
		center(titleStyle.Render("进程详情"), innerW),
		divider(innerW),
	}

	info := []struct {
		label, value string
	}{
		{"PID", fmt.Sprintf("%d", d.PID)},
		{"命令", d.Command},
		{"用户", d.User},
		{"状态", d.Status},
		{"CPU", fmt.Sprintf("%.1f%%", d.CPU)},
		{"内存", fmtSize(d.MemRSS)},
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

	// Button row: compute widths so hit boxes can be stored.
	termLabel := "结束"
	killLabel := "强制结束"
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
	frame := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)

	// Store button hit boxes in terminal coordinates.
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
		return overlayConfirm(m, width, height)
	}
	return frame
}

// detailExtra returns the card-specific metric label and value for the detail
// popup. Memory is already shown in the common info block, so it returns empty
// for the memory card.
func detailExtra(d detailProc, source cardKey) (label, value string) {
	switch source {
	case cardMem:
		return "", "" // memory already in common info
	case cardNet:
		return "网络", fmtRate(d.NetUp) + " / " + fmtRate(d.NetDown)
	case cardDisk:
		return "磁盘", fmtRate(d.DiskRead) + " / " + fmtRate(d.DiskWrite)
	case cardGPU:
		return "显存", fmtSize(d.GPUMem)
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
