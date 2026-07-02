package tui

import tea "github.com/charmbracelet/bubbletea"

// ExportMiniListHit exposes miniListHit for testing.
func ExportMiniListHit(m tea.Model, x, y int) (pid int32, name string, source cardKey, ok bool) {
	mod := m.(*model)
	return mod.miniListHit(x, y)
}
