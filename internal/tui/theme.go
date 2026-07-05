package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// ThemeName identifies the available UI themes. Built-in names are "dark" and
// "light"; any other value is treated as a third-party theme loaded from
// ~/.xtop/themes/<name>.json.
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

// rawThemeColors is the JSON representation of a third-party theme file.
type rawThemeColors struct {
	Primary           string `json:"primary"`
	PrimaryHi         string `json:"primaryHi"`
	PrimaryDim        string `json:"primaryDim"`
	Text              string `json:"text"`
	Faint             string `json:"faint"`
	Gray              string `json:"gray"`
	GrayLite          string `json:"grayLite"`
	Track             string `json:"track"`
	Red               string `json:"red"`
	Yellow            string `json:"yellow"`
	Orange            string `json:"orange"`
	Blue              string `json:"blue"`
	RowButtonBG       string `json:"rowButtonBg"`
	RowButtonDangerBG string `json:"rowButtonDangerBg"`
	SelRowBG          string `json:"selRowBg"`
	SelRowFG          string `json:"selRowFg"`
}

var builtinThemes = map[ThemeName]themeColors{
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

// thirdPartyThemes caches loaded third-party themes so SetTheme can switch back
// to them without re-reading the file every frame. It is separate from the
// built-in map so that a stale entry is never mistaken for a built-in theme.
var thirdPartyThemes = map[ThemeName]themeColors{}

var currentThemeName = ThemeDark

// SetTheme switches the active UI theme and reinitializes all dependent styles.
// For built-in themes it uses the bundled palette. For any other name it attempts
// to load ~/.xtop/themes/<name>.json; if loading fails it falls back to dark.
func SetTheme(name ThemeName) {
	if tc, ok := builtinThemes[name]; ok {
		currentThemeName = name
		applyTheme(tc)
		return
	}

	// Always reload from disk and refresh the cache so that deleting or updating
	// a third-party theme file is reflected immediately.
	delete(thirdPartyThemes, name)

	tc, err := loadThirdPartyTheme(name)
	if err != nil {
		currentThemeName = ThemeDark
		applyTheme(builtinThemes[ThemeDark])
		return
	}
	thirdPartyThemes[name] = tc
	currentThemeName = name
	applyTheme(tc)
}

// CurrentThemeName returns the name of the active theme.
func CurrentThemeName() ThemeName {
	return currentThemeName
}

// IsBuiltInTheme reports whether name is one of the bundled themes.
func IsBuiltInTheme(name ThemeName) bool {
	_, ok := builtinThemes[name]
	return ok
}

// UserThemesDir returns the directory for third-party theme files.
func UserThemesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xtop", "themes")
}

// ListUserThemes returns the names of all third-party themes in UserThemesDir.
func ListUserThemes() []ThemeName {
	dir := UserThemesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []ThemeName
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		out = append(out, ThemeName(name[:len(name)-len(".json")]))
	}
	return out
}

func loadThirdPartyTheme(name ThemeName) (themeColors, error) {
	path := filepath.Join(UserThemesDir(), string(name)+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return themeColors{}, fmt.Errorf("read theme file: %w", err)
	}
	var raw rawThemeColors
	if err := json.Unmarshal(b, &raw); err != nil {
		return themeColors{}, fmt.Errorf("parse theme file: %w", err)
	}

	c := themeColors{
		Primary:           colorOr(raw.Primary, "#33FF66"),
		PrimaryHi:         colorOr(raw.PrimaryHi, "#7CFFA6"),
		PrimaryDim:        colorOr(raw.PrimaryDim, "#1F7A3D"),
		Text:              colorOr(raw.Text, "#C8E6D0"),
		Faint:             colorOr(raw.Faint, "#6C8A76"),
		Gray:              colorOr(raw.Gray, "#4A4A4A"),
		GrayLite:          colorOr(raw.GrayLite, "#8A8A8A"),
		Track:             colorOr(raw.Track, "238"),
		Red:               colorOr(raw.Red, "#FF5555"),
		Yellow:            colorOr(raw.Yellow, "#E6DB74"),
		Orange:            colorOr(raw.Orange, "#E0A54B"),
		Blue:              colorOr(raw.Blue, "#6D8CFF"),
		RowButtonBG:       colorOr(raw.RowButtonBG, "#123322"),
		RowButtonDangerBG: colorOr(raw.RowButtonDangerBG, "#3A1414"),
		SelRowBG:          colorOr(raw.SelRowBG, "#0F3D24"),
		SelRowFG:          colorOr(raw.SelRowFG, "#7CFFA6"),
	}
	return c, nil
}

func colorOr(value, fallback string) lipgloss.Color {
	if value == "" {
		return lipgloss.Color(fallback)
	}
	return lipgloss.Color(value)
}

func applyTheme(c themeColors) {
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
