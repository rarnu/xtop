package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/collector"
)

// collectInterval is how often each subsystem is refreshed (spec: every 5 seconds).
const collectInterval = 5 * time.Second

// subsystemMsg carries one completed subsystem snapshot.
type subsystemMsg struct {
	part string
	snap collector.Snapshot
}

// tickMsg fires collectInterval to trigger the next round of asynchronous
// subsystem collection.
type tickMsg struct{}

// collectAllCmd runs all six collectors concurrently. Each finished subsystem
// is delivered as a subsystemMsg so the UI can refresh cards as soon as their
// data is ready.
func collectAllCmd(c *collector.Collector) tea.Cmd {
	return tea.Batch(
		collectCPU(c),
		collectMem(c),
		collectDisk(c),
		collectNet(c),
		collectGPU(c),
		collectProc(c),
	)
}

func collectCPU(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return subsystemMsg{part: "cpu", snap: collector.Snapshot{CPU: c.CollectCPU()}}
	}
}

func collectMem(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return subsystemMsg{part: "mem", snap: collector.Snapshot{Mem: c.CollectMem()}}
	}
}

func collectDisk(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return subsystemMsg{part: "disk", snap: collector.Snapshot{Disk: c.CollectDisk()}}
	}
}

func collectNet(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return subsystemMsg{part: "net", snap: collector.Snapshot{Net: c.CollectNet()}}
	}
}

func collectGPU(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return subsystemMsg{part: "gpu", snap: collector.Snapshot{GPU: c.CollectGPU()}}
	}
}

func collectProc(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		return subsystemMsg{part: "proc", snap: collector.Snapshot{Proc: c.CollectProc()}}
	}
}

// scheduleTick waits collectInterval, then emits a tickMsg to start the next
// asynchronous collection round.
func scheduleTick() tea.Cmd {
	return tea.Tick(collectInterval, func(time.Time) tea.Msg { return tickMsg{} })
}
