package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/collector"
)

// collectInterval is how often data is refreshed (spec: every 5 seconds).
const collectInterval = 5 * time.Second

// snapshotMsg carries a freshly collected snapshot back to the UI.
type snapshotMsg struct{ snap collector.Snapshot }

// tickMsg fires collectInterval after a snapshot to schedule the next collect.
type tickMsg struct{}

// collectCmd runs the (blocking) collection in Bubble Tea's goroutine and
// delivers the result as a snapshotMsg.
func collectCmd(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return snapshotMsg{snap: c.Snapshot()}
	}
}

// scheduleTick waits collectInterval, then emits a tickMsg.
func scheduleTick() tea.Cmd {
	return tea.Tick(collectInterval, func(time.Time) tea.Msg { return tickMsg{} })
}
