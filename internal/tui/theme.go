package tui

import "github.com/charmbracelet/lipgloss"

// ThemeName identifies the available UI themes.
type ThemeName string

const (
	ThemeDark  ThemeName = "dark"
	ThemeLight ThemeName = "light"
)

// themeColors holds the palette for one theme.
type themeColors struct {
	Primary           lipgloss.Color
	PrimaryHi         lipgloss.Color
	PrimaryDim        lipgloss.Color
	Text              lipgloss.Color
	Faint             lipgloss.Color
	Gray              lipgloss.Color
	GrayLite          lipgloss.Color
	Track             lipgloss.Color
	Red               lipgloss.Color
	Yellow            lipgloss.Color
	Orange            lipgloss.Color
	Blue              lipgloss.Color
	RowButtonBG       lipgloss.Color
	RowButtonDangerBG lipgloss.Color
	SelRowBG          lipgloss.Color
	SelRowFG          lipgloss.Color
}

var themes = map[ThemeName]themeColors{
	ThemeDark: {
		Primary:           lipgloss.Color("#33FF66"),
		PrimaryHi:         lipgloss.Color("#7CFFA6"),
		PrimaryDim:        lipgloss.Color("#1F7A3D"),
		Text:              lipgloss.Color("#C8E6D0"),
		Faint:             lipgloss.Color("#6C8A76"),
		Gray:              lipgloss.Color("#4A4A4A"),
		GrayLite:          lipgloss.Color("#8A8A8A"),
		Track:             lipgloss.Color("238"),
		Red:               lipgloss.Color("#FF5555"),
		Yellow:            lipgloss.Color("#E6DB74"),
		Orange:            lipgloss.Color("#E0A54B"),
		Blue:              lipgloss.Color("#6D8CFF"),
		RowButtonBG:       lipgloss.Color("#123322"),
		RowButtonDangerBG: lipgloss.Color("#3A1414"),
		SelRowBG:          lipgloss.Color("#0F3D24"),
		SelRowFG:          lipgloss.Color("#7CFFA6"),
	},
	ThemeLight: {
		Primary:           lipgloss.Color("#1A1A1A"),
		PrimaryHi:         lipgloss.Color("#000000"),
		PrimaryDim:        lipgloss.Color("#666666"),
		Text:              lipgloss.Color("#333333"),
		Faint:             lipgloss.Color("#666666"),
		Gray:              lipgloss.Color("#9E9E9E"),
		GrayLite:          lipgloss.Color("#BDBDBD"),
		Track:             lipgloss.Color("#E0E0E0"),
		Red:               lipgloss.Color("#D32F2F"),
		Yellow:            lipgloss.Color("#F9A825"),
		Orange:            lipgloss.Color("#F57C00"),
		Blue:              lipgloss.Color("#1976D2"),
		RowButtonBG:       lipgloss.Color("#E8E8E8"),
		RowButtonDangerBG: lipgloss.Color("#FFEBEE"),
		SelRowBG:          lipgloss.Color("#E0E0E0"),
		SelRowFG:          lipgloss.Color("#000000"),
	},
}

var currentThemeName = ThemeDark

// SetTheme switches the active UI theme and reinitializes all dependent styles.
func SetTheme(name ThemeName) {
	if _, ok := themes[name]; !ok {
		name = ThemeDark
	}
	currentThemeName = name
	applyTheme()
}

// CurrentThemeName returns the name of the active theme.
func CurrentThemeName() ThemeName {
	return currentThemeName
}

func applyTheme() {
	c := themes[currentThemeName]
	colGreen = c.Primary
	colGreenHi = c.PrimaryHi
	colGreenDim = c.PrimaryDim
	colText = c.Text
	colFaint = c.Faint
	colGray = c.Gray
	colGrayLite = c.GrayLite
	colTrack = c.Track
	colRed = c.Red
	colYellow = c.Yellow
	colOrange = c.Orange
	colBlue = c.Blue

	titleStyle = lipgloss.NewStyle().Foreground(colGreenHi).Bold(true)
	iconStyle = lipgloss.NewStyle().Foreground(colGreen).Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(colFaint)
	textStyle = lipgloss.NewStyle().Foreground(colText)
	valueStyle = lipgloss.NewStyle().Foreground(colText).Bold(true)
	faintStyle = lipgloss.NewStyle().Foreground(colFaint)

	pillStyle = lipgloss.NewStyle().
			Foreground(colText).
			Background(colTrack).
			Padding(0, 1)

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

	rowButtonStyle = lipgloss.NewStyle().
			Foreground(colGreenHi).
			Background(c.RowButtonBG).
			Padding(0, 1)

	rowButtonDangerStyle = lipgloss.NewStyle().
				Foreground(colRed).
				Background(c.RowButtonDangerBG).
				Padding(0, 1)

	helpBarStyle = lipgloss.NewStyle().Foreground(colFaint)

	selRowStyle = lipgloss.NewStyle().
			Background(c.SelRowBG).
			Foreground(c.SelRowFG)
}
