package tui

import "github.com/charmbracelet/lipgloss"

// Matrix-green palette.
const (
	colGreen    = lipgloss.Color("#33FF66") // primary
	colGreenHi  = lipgloss.Color("#7CFFA6") // bright accents / titles
	colGreenDim = lipgloss.Color("#1F7A3D") // borders, dim fills
	colText     = lipgloss.Color("#C8E6D0") // body text
	colFaint    = lipgloss.Color("#6C8A76") // labels / secondary text
	colGray     = lipgloss.Color("#4A4A4A") // empty track
	colGrayLite = lipgloss.Color("#8A8A8A") // cache legend
	// colTrack is the dim "empty" fill behind meters, sparklines and the
	// scrollbar. It must be a *visibly* dark gray, not near-black: a near-black
	// glyph on a dark terminal background disappears, leaving only each thin
	// │/─ character's anti-alias halo, which reads as faint white lines on some
	// displays (DPI / font-smoothing dependent). Using an ANSI-256 gray (238)
	// keeps the glyph body visible as gray and avoids that artifact; it also
	// degrades to a safe ANSI "bright black" (90) rather than a truecolor escape
	// on limited terminals.
	colTrack = lipgloss.Color("238") // very dark fill track
	colRed      = lipgloss.Color("#FF5555")
	colYellow   = lipgloss.Color("#E6DB74")
	colOrange   = lipgloss.Color("#E0A54B")
	colBlue     = lipgloss.Color("#6D8CFF") // sparkline secondary (upload/idle line)
)

var (
	titleStyle = lipgloss.NewStyle().Foreground(colGreenHi).Bold(true)
	iconStyle  = lipgloss.NewStyle().Foreground(colGreen).Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(colFaint)
	textStyle  = lipgloss.NewStyle().Foreground(colText)
	valueStyle = lipgloss.NewStyle().Foreground(colText).Bold(true)
	faintStyle = lipgloss.NewStyle().Foreground(colFaint)

	// Single-line pill shown to the right of a card title (totals/summaries).
	pillStyle = lipgloss.NewStyle().
			Foreground(colText).
			Background(colTrack).
			Padding(0, 1)

	// Single-line orange filesystem-type badge (e.g. ext4).
	badgeStyle = lipgloss.NewStyle().
			Foreground(colOrange).
			Background(lipgloss.Color("#2A2410")).
			Padding(0, 1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colGreenDim).
			Padding(0, 1)

	cardFocusStyle = cardStyle.
			BorderForeground(colGreen)

	dividerStyle = lipgloss.NewStyle().Foreground(colGreenDim)

	buttonStyle = lipgloss.NewStyle().
			Foreground(colGreenHi).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colGreen).
			Padding(0, 2)

	buttonDangerStyle = lipgloss.NewStyle().
				Foreground(colRed).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colRed).
				Padding(0, 1)

	// Single-line action buttons used inside process-table rows.
	rowButtonStyle = lipgloss.NewStyle().
			Foreground(colGreenHi).
			Background(lipgloss.Color("#123322")).
			Padding(0, 1)

	rowButtonDangerStyle = lipgloss.NewStyle().
				Foreground(colRed).
				Background(lipgloss.Color("#3A1414")).
				Padding(0, 1)

	helpBarStyle = lipgloss.NewStyle().Foreground(colFaint)
)

// levelColor maps a 0-100 utilisation level to green / yellow / red.
func levelColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 90:
		return colRed
	case pct >= 70:
		return colYellow
	default:
		return colGreen
	}
}
