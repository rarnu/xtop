package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtop/internal/version"
)

// aboutInfo describes the metadata shown in the about dialog.
type aboutInfo struct {
	Version   string
	GitHub    string
	License   string
	GoVersion string
	BuildTime string
}

// defaultAbout returns the static about metadata. These values can be overridden
// at link time with -ldflags if desired.
func defaultAbout() aboutInfo {
	return aboutInfo{
		Version:   version.Version,
		GitHub:    "github.com/rarnu/xtop",
		License:   "GPLv3",
		GoVersion: "",
		BuildTime: "",
	}
}

// aboutDialogSize returns the inner dimensions of the about dialog.
func aboutDialogSize(termW, termH int) (innerW, innerH int) {
	innerW = termW / 3
	if innerW > 56 {
		innerW = 56
	}
	if innerW < 34 {
		innerW = 34
	}
	innerH = 10
	if innerH > termH-4 {
		innerH = termH - 4
	}
	if innerH < 8 {
		innerH = 8
	}
	return innerW, innerH
}

// overlayAboutModal renders the about dialog over the existing dashboard base.
func overlayAboutModal(m *model, base string) string {
	width, height := m.width, m.height
	innerW, innerH := aboutDialogSize(width, height)
	info := m.about.info

	rows := []string{
		center(titleStyle.Render("XTOP"), innerW),
		center(faintStyle.Render(info.Version), innerW),
		divider(innerW),
	}

	lines := []struct{ label, value string }{
		{T("about.github"), info.GitHub},
		{T("about.license"), info.License},
	}
	if info.GoVersion != "" {
		lines = append(lines, struct{ label, value string }{T("about.go"), info.GoVersion})
	}
	if info.BuildTime != "" {
		lines = append(lines, struct{ label, value string }{T("about.build"), info.BuildTime})
	}

	maxLabel := 0
	maxValue := 0
	for _, it := range lines {
		if w := lipgloss.Width(it.label); w > maxLabel {
			maxLabel = w
		}
		if w := lipgloss.Width(it.value); w > maxValue {
			maxValue = w
		}
	}

	for _, it := range lines {
		labelPart := labelStyle.Render(fitCell(it.label, maxLabel, false))
		valuePart := textStyle.Render(fitCell(it.value, maxValue, false))
		line := labelPart + "  " + valuePart + " "
		rows = append(rows, line)
	}

	rows = append(rows, "")
	rows = append(rows, textStyle.Render(T("about.description")))

	for len(rows) < innerH-1 {
		rows = append(rows, strings.Repeat(" ", innerW))
	}

	okLabel := T("about.ok")
	okBtn := rowButtonStyle.Render(okLabel)
	btnW := lipgloss.Width(okBtn)
	btnLine := strings.Repeat(" ", (innerW-btnW)/2) + okBtn
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

	m.about.okX0 = boxLeft + 1 + (innerW-btnW)/2
	m.about.okX1 = m.about.okX0 + btnW
	m.about.btnY = boxTop + innerH

	return overlayBox(base, box, boxLeft, boxTop)
}
