package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// truncPlain truncates a plain (unstyled) string to width runes, adding an
// ellipsis when it overflows.
func truncPlain(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	// Trim rune-by-rune until it fits with room for the ellipsis.
	for len(r) > 0 && lipgloss.Width(string(r))+1 > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// padRow right-pads a (possibly styled) line with spaces to exactly width cells.
func padRow(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// joinLR places left and right on one line separated by filler spaces so the
// total is exactly width cells. Right content is dropped if there is no room.
func joinLR(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if lw+rw+1 > width {
		return padRow(left, width)
	}
	gap := width - lw - rw
	return left + strings.Repeat(" ", gap) + right
}

// divider returns a full-width dim horizontal rule.
func divider(width int) string {
	if width < 1 {
		width = 1
	}
	return dividerStyle.Render(strings.Repeat("─", width))
}

// renderCard draws a bordered card: header (icon + title, optional right-aligned
// summary), a divider, then the body lines. innerWidth is the content width
// inside the border+padding; the returned block is innerWidth+4 cells wide.
func renderCard(innerWidth int, icon, title, headerRight string, bodyLines []string, focused bool) string {
	header := joinLR(iconStyle.Render(icon)+" "+titleStyle.Render(title), headerRight, innerWidth)

	rows := make([]string, 0, len(bodyLines)+2)
	rows = append(rows, header)
	rows = append(rows, divider(innerWidth))
	for _, l := range bodyLines {
		rows = append(rows, padRow(l, innerWidth))
	}

	content := strings.Join(rows, "\n")
	style := cardStyle
	if focused {
		style = cardFocusStyle
	}
	// lipgloss Width is the content box *including* horizontal padding, so add
	// the 2 padding cells to keep the text area exactly innerWidth wide.
	return style.Width(innerWidth + 2).Render(content)
}
