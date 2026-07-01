package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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

// fitCell truncates a plain string to width and pads it (left or right aligned)
// so the result is exactly width cells. Style the result afterwards.
func fitCell(s string, width int, rightAlign bool) string {
	if width <= 0 {
		return ""
	}
	s = truncPlain(s, width)
	pad := width - lipgloss.Width(s)
	if pad < 0 {
		pad = 0
	}
	if rightAlign {
		return strings.Repeat(" ", pad) + s
	}
	return s + strings.Repeat(" ", pad)
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes SGR escape sequences so a rendered line can be searched or
// measured as plain text.
func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// findText locates needle inside a slice of rendered (possibly styled) lines,
// returning the line index and the visible column span [x0,x1) of the match.
func findText(lines []string, needle string) (line, x0, x1 int, ok bool) {
	for i, l := range lines {
		plain := stripANSI(l)
		idx := strings.Index(plain, needle)
		if idx < 0 {
			continue
		}
		x0 = lipgloss.Width(plain[:idx])
		x1 = x0 + lipgloss.Width(needle)
		return i, x0, x1, true
	}
	return 0, 0, 0, false
}

// lipglossFg is a shorthand for a foreground-only style.
func lipglossFg(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}
