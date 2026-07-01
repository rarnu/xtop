package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderCard draws a bordered card of fixed inner size and renders an optional
// vertical scrollbar when body content overflows. Mouse wheel over the card
// scrolls the body.
//
// The returned block is (innerWidth+4) x (innerHeight+2) cells.
func renderCard(innerWidth, innerHeight int, icon, title, headerRight string, bodyLines []string, focused bool, scroll *cardScroll) string {
	if innerHeight < 2 {
		innerHeight = 2
	}

	bodyH := innerHeight
	visible, sbY0, sbY1, hasSB := cardBody(bodyLines, bodyH, innerWidth, scroll)

	if hasSB && innerWidth > scrollbarMargin {
		bar := renderScrollbar(bodyH, sbY0, sbY1)
		contentW := innerWidth - scrollbarMargin
		padCols := scrollbarMargin - 1
		for i := range visible {
			visible[i] = fitStyledLine(visible[i], contentW) + strings.Repeat(" ", padCols) + bar[i]
		}
	}

	header := joinLR(iconStyle.Render(icon)+" "+titleStyle.Render(title), headerRight, innerWidth)

	rows := make([]string, 0, len(visible)+2)
	rows = append(rows, header)
	rows = append(rows, divider(innerWidth))
	for _, l := range visible {
		rows = append(rows, padRow(l, innerWidth))
	}

	content := strings.Join(rows, "\n")
	style := cardStyle
	if focused {
		style = cardFocusStyle
	}
	return style.Width(innerWidth + 2).Height(innerHeight + 2).Render(content)
}

// fitStyledLine truncates a possibly-styled line to exactly target visible
// cells, preserving ANSI escape sequences. It pads with spaces if shorter.
func fitStyledLine(s string, target int) string {
	plain := stripANSI(s)
	w := lipgloss.Width(plain)
	if w <= target {
		return s + strings.Repeat(" ", target-w)
	}
	// Need to truncate while keeping styles.
	out := truncateStyled(s, target)
	return out
}

// truncateStyled shortens a styled string to fit target visible cells.
// It walks rune-by-rune, toggling ANSI state, and stops before exceeding target.
func truncateStyled(s string, target int) string {
	var out strings.Builder
	visible := 0
	inESC := false
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
		if visible+rw > target {
			break
		}
		out.WriteRune(r)
		visible += rw
	}
	if visible < target {
		out.WriteString(strings.Repeat(" ", target-visible))
	}
	return out.String()
}

// cardBody returns the visible body lines, clipping/padding to visibleH and
// computing scrollbar geometry. It mutates scroll.offset/max.
func cardBody(body []string, visibleH, innerWidth int, scroll *cardScroll) (lines []string, sbY0, sbY1 int, hasSB bool) {
	if visibleH < 1 {
		visibleH = 1
	}
	contentH := len(body)
	if contentH <= visibleH {
		lines = make([]string, visibleH)
		for i := 0; i < visibleH; i++ {
			if i < contentH && body[i] != "" {
				lines[i] = padRow(body[i], innerWidth)
			} else {
				lines[i] = strings.Repeat(" ", innerWidth)
			}
		}
		scroll.max = 0
		scroll.offset = 0
		return lines, 0, 0, false
	}

	scroll.max = contentH - visibleH
	if scroll.offset < 0 {
		scroll.offset = 0
	}
	if scroll.offset > scroll.max {
		scroll.offset = scroll.max
	}

	lines = make([]string, visibleH)
	for i := 0; i < visibleH; i++ {
		line := body[scroll.offset+i]
		if line == "" {
			lines[i] = strings.Repeat(" ", innerWidth)
		} else {
			lines[i] = padRow(line, innerWidth)
		}
	}

	trackH := visibleH
	thumbRatio := float64(visibleH) / float64(contentH)
	thumbH := int(float64(trackH) * thumbRatio)
	if thumbH < 1 {
		thumbH = 1
	}
	thumbPos := 0
	if scroll.max > 0 {
		thumbPos = int(float64(scroll.offset) / float64(scroll.max) * float64(trackH-thumbH))
	}

	return lines, thumbPos, thumbPos + thumbH, true
}

func renderScrollbar(visibleH, y0, y1 int) []string {
	col := make([]string, visibleH)
	for i := 0; i < visibleH; i++ {
		if i >= y0 && i < y1 {
			col[i] = lipgloss.NewStyle().Foreground(colGreen).Render("█")
		} else {
			col[i] = lipgloss.NewStyle().Foreground(colTrack).Render("│")
		}
	}
	return col
}
