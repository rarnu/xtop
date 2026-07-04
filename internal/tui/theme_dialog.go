package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// themeDialogState tracks the theme picker popup.
type themeDialogState struct {
	active   bool
	selected int
}

var themeOptions = []ThemeName{ThemeDark, ThemeLight}

// overlayThemeDialog renders the theme picker popup on top of the dashboard.
func overlayThemeDialog(m *model, base string) string {
	width, height := m.width, m.height
	innerW := width / 3
	if innerW > 40 {
		innerW = 40
	}
	if innerW < 24 {
		innerW = 24
	}
	innerH := len(themeOptions) + 5 // title, divider, options, gap, hint

	rows := []string{
		center(titleStyle.Render(T("theme.title")), innerW),
		divider(innerW),
	}

	for i, t := range themeOptions {
		label := T("theme." + string(t))
		line := "  " + label
		if i == m.themeDialog.selected {
			line = selectedRowStyle.Render(fitCell("> "+label, innerW, false))
		}
		rows = append(rows, padRow(line, innerW))
	}

	rows = append(rows, "")
	rows = append(rows, center(faintStyle.Render(T("theme.hint")), innerW))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colGreenDim).
		Width(innerW).
		Height(innerH).
		Render(strings.Join(rows, "\n"))

	boxW := lipgloss.Width(box)
	boxH := len(strings.Split(box, "\n"))
	left := (width - boxW) / 2
	top := (height - boxH) / 2
	m.themeDialogBox = dialogBox{left: left, top: top, width: boxW, height: boxH}
	return overlayBox(base, box, left, top)
}

// updateThemeDialogKey handles keyboard input for the theme picker.
func (m *model) updateThemeDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "t":
		m.themeDialog.active = false
	case "up", "k":
		m.themeDialog.selected--
		if m.themeDialog.selected < 0 {
			m.themeDialog.selected = len(themeOptions) - 1
		}
	case "down", "j":
		m.themeDialog.selected++
		if m.themeDialog.selected >= len(themeOptions) {
			m.themeDialog.selected = 0
		}
	case "enter":
		return m.applyTheme(themeOptions[m.themeDialog.selected])
	}
	return m, nil
}

// updateThemeDialogMouse handles mouse input for the theme picker.
func (m *model) updateThemeDialogMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	x, y := msg.X, msg.Y
	db := m.themeDialogBox
	if x < db.left || x >= db.left+db.width || y < db.top || y >= db.top+db.height {
		m.themeDialog.active = false
		return m, nil
	}

	// Content starts two rows below the top border (title + divider).
	row := y - db.top - 2
	if row >= 0 && row < len(themeOptions) {
		return m.applyTheme(themeOptions[row])
	}
	return m, nil
}

func (m *model) applyTheme(name ThemeName) (tea.Model, tea.Cmd) {
	SetTheme(name)
	m.themeDialog.active = false
	m.recompute()
	cfg := LoadConfig()
	cfg.Theme = name
	_ = SaveConfig(cfg)
	return m, nil
}

// dialogBox records the screen geometry of a centered popup.
type dialogBox struct {
	left, top, width, height int
}
