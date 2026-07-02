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

	visible, sbY0, sbY1, hasSB := cardBody(bodyLines, innerHeight, innerWidth, scroll)
	visible = applyScrollbar(visible, innerWidth, innerHeight, sbY0, sbY1, hasSB)

	header := joinLR(iconStyle.Render(icon)+" "+titleStyle.Render(title), headerRight, innerWidth)

	rows := make([]string, 0, len(visible)+2)
	rows = append(rows, header)
	rows = append(rows, divider(innerWidth))
	rows = append(rows, visible...)

	return styleCard(rows, innerWidth, innerHeight, focused)
}

// renderCardSplit draws a card whose body is split into a fixed region on top
// (fixedLines, never scrolled) and a scrollable list region below. Only the list
// region gets its own vertical scrollbar when it overflows; the fixed lines
// always stay visible. Used by the memory and network cards (feature: keep the
// summary pinned, scroll just the process list).
//
// The returned block is the same size as renderCard's: (innerWidth+4) x
// (innerHeight+2) cells.
func renderCardSplit(innerWidth, innerHeight int, icon, title, headerRight string, fixedLines, listLines []string, focused bool, listScroll *cardScroll) string {
	if innerHeight < 2 {
		innerHeight = 2
	}

	fixedH := len(fixedLines)
	if fixedH > innerHeight {
		fixedH = innerHeight
	}
	listH := innerHeight - fixedH // rows left for the scrollable list viewport

	fixed := make([]string, fixedH)
	for i := 0; i < fixedH; i++ {
		fixed[i] = padRow(fixedLines[i], innerWidth)
	}

	var listView []string
	if listH > 0 {
		visible, sbY0, sbY1, hasSB := cardBody(listLines, listH, innerWidth, listScroll)
		listView = applyScrollbar(visible, innerWidth, listH, sbY0, sbY1, hasSB)
	} else {
		listScroll.max = 0
		listScroll.offset = 0
	}

	header := joinLR(iconStyle.Render(icon)+" "+titleStyle.Render(title), headerRight, innerWidth)

	rows := make([]string, 0, innerHeight+2)
	rows = append(rows, header)
	rows = append(rows, divider(innerWidth))
	rows = append(rows, fixed...)
	rows = append(rows, listView...)

	return styleCard(rows, innerWidth, innerHeight, focused)
}

// applyScrollbar overlays a right-hand scrollbar onto the visible viewport lines
// (when hasSB) and pads every line to innerWidth. bodyH is the viewport height.
func applyScrollbar(visible []string, innerWidth, bodyH, sbY0, sbY1 int, hasSB bool) []string {
	if hasSB && innerWidth > scrollbarMargin {
		bar := renderScrollbar(bodyH, sbY0, sbY1)
		contentW := innerWidth - scrollbarMargin
		padCols := scrollbarMargin - 1
		for i := range visible {
			visible[i] = fitStyledLine(visible[i], contentW) + strings.Repeat(" ", padCols) + bar[i]
		}
	}
	for i := range visible {
		visible[i] = padRow(visible[i], innerWidth)
	}
	return visible
}

// disabledCard renders a placeholder for a deliberately disabled card.
func disabledCard(innerWidth, innerHeight int, icon, title, reason string) string {
	lines := []string{faintStyle.Render(reason), ""}
	header := joinLR(iconStyle.Render(icon)+" "+titleStyle.Render(title), "", innerWidth)
	rows := []string{header, divider(innerWidth)}
	for i := 0; i < innerHeight; i++ {
		if i < len(lines) {
			rows = append(rows, padRow(lines[i], innerWidth))
		} else {
			rows = append(rows, strings.Repeat(" ", innerWidth))
		}
	}
	return styleCard(rows, innerWidth, innerHeight, false)
}

// styleCard wraps assembled content rows in the (focused) card border/padding at
// a fixed size.
func styleCard(rows []string, innerWidth, innerHeight int, focused bool) string {
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
