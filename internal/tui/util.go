package tui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// spacesCache pre-generates common-width space strings to avoid repeated small
// allocations in render hot paths. Widths beyond the cache fall back to
// strings.Repeat.
var spacesCache = func() []string {
	const max = 200
	s := make([]string, max+1)
	for i := 0; i <= max; i++ {
		s[i] = strings.Repeat(" ", i)
	}
	return s
}()

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	if n < len(spacesCache) {
		return spacesCache[n]
	}
	return strings.Repeat(" ", n)
}

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

// padLeftInt formats i as a right-aligned decimal string of width cells,
// padding with spaces on the left. Avoids fmt.Sprintf in hot paths.
func padLeftInt(i, width int) string {
	s := strconv.Itoa(i)
	if len(s) >= width {
		return s
	}
	return spaces(width-len(s)) + s
}

// padFloat1 formats v with one decimal place and right-aligns it to width cells.
// Avoids fmt.Sprintf in hot paths.
func padFloat1(v float64, width int) string {
	s := formatFloat1(v)
	if len(s) >= width {
		return s
	}
	return spaces(width-len(s)) + s
}

// formatPct0 formats v with no decimal places and appends '%'.
func formatPct0(v float64) string {
	if v < 0 {
		v = 0
	}
	if v >= 100 {
		return "100%"
	}
	return strconv.FormatInt(int64(v+0.5), 10) + "%"
}

// formatFloat0 formats v with no decimal places (no suffix).
func formatFloat0(v float64) string {
	if v < 0 {
		v = 0
	}
	return strconv.FormatInt(int64(v+0.5), 10)
}

// formatFloat1Pct formats v with one decimal place followed by '%'.
func formatFloat1Pct(v float64) string {
	return formatFloat1(v) + "%"
}

// formatFloat1 formats v with one decimal place without using fmt.Sprintf.
func formatFloat1(v float64) string {
	if v < 0 {
		v = 0
	}
	v += 0.05 // round to one decimal place
	whole := int64(v)
	frac := int64((v - float64(whole)) * 10)
	if frac < 0 {
		frac = 0
	}
	if frac >= 10 {
		frac = 0
		whole++
	}
	if frac == 0 {
		return strconv.FormatInt(whole, 10) + ".0"
	}
	return strconv.FormatInt(whole, 10) + "." + strconv.FormatInt(frac, 10)
}

// stringWidth returns the visible cell width of s, stripping any ANSI SGR
// sequences first so styled strings are measured as plain text. This is a faster
// replacement for lipgloss.Width in hot paths.
func stringWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

// truncPlain truncates a plain (unstyled) string to width cells, adding an
// ellipsis when it overflows.
func truncPlain(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	r := []rune(s)
	w := 0
	for i, c := range r {
		rw := runewidth.RuneWidth(c)
		if w+rw+1 > width {
			return string(r[:i]) + "…"
		}
		w += rw
	}
	return s
}

// padRow right-pads a (possibly styled) line with spaces to exactly width cells.
func padRow(s string, width int) string {
	w := stringWidth(s)
	if w >= width {
		return s
	}
	return s + spaces(width-w)
}

// joinLR places left and right on one line separated by filler spaces so the
// total is exactly width cells. Right content is dropped if there is no room.
func joinLR(left, right string, width int) string {
	lw := stringWidth(left)
	rw := stringWidth(right)
	if lw+rw+1 > width {
		return padRow(left, width)
	}
	gap := width - lw - rw
	return left + spaces(gap) + right
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
	pad := width - runewidth.StringWidth(s)
	if pad < 0 {
		pad = 0
	}
	if rightAlign {
		return spaces(pad) + s
	}
	return s + spaces(pad)
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes SGR escape sequences so a rendered line can be searched or
// measured as plain text.
func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// findText locates needle inside a slice of rendered (possibly styled) lines,
// returning the line index and the visible column span [x0,x1) of the match.
func findText(lines []string, needle string) (line, x0, x1 int, ok bool) {
	needleW := runewidth.StringWidth(needle)
	for i, l := range lines {
		plain := stripANSI(l)
		idx := strings.Index(plain, needle)
		if idx < 0 {
			continue
		}
		x0 = runewidth.StringWidth(plain[:idx])
		x1 = x0 + needleW
		return i, x0, x1, true
	}
	return 0, 0, 0, false
}

// lipglossFg returns a cached foreground-only style to avoid repeated allocations
// in render hot paths.
func lipglossFg(c lipgloss.Color) lipgloss.Style {
	return fgStyle(c)
}
