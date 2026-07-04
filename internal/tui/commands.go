package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/collector"
)

// tickMsg triggers a once-per-second UI refresh so the footer clock updates.
type tickMsg struct{}

const tickInterval = 1 * time.Second

// tickCmd sleeps for one second and then returns tickMsg. Update() returns this
// command again, creating a perpetual one-second ticker without overlapping
// goroutines.
func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

// collectInterval is how often each subsystem is refreshed (spec: every 5 seconds).
const collectInterval = 5 * time.Second

// subsystemMsg carries one completed subsystem snapshot.
type subsystemMsg struct {
	part string
	snap collector.Snapshot
}

// recollectMsg asks for a single subsystem to be collected again. It is emitted
// collectInterval after that subsystem's previous collection *completed*, so a
// slow subsystem simply refreshes less often instead of piling up overlapping
// collections (which would exhaust CPU and freeze the UI).
type recollectMsg struct {
	part string
}

// collectAllCmd runs the on-demand collectors concurrently and starts the
// process-cache listener. Used for one-shot full refreshes.
func collectAllCmd(c *collector.Collector) tea.Cmd {
	return tea.Batch(
		collectCPU(c),
		collectMem(c),
		collectDisk(c),
		collectNet(c),
		collectGPU(c),
		procUpdateCmd(c),
		netProcUpdateCmd(c),
	)
}

// collectPart returns the collector command for a single subsystem.
func collectPart(c *collector.Collector, part string) tea.Cmd {
	switch part {
	case "cpu":
		return collectCPU(c)
	case "mem":
		return collectMem(c)
	case "disk":
		return collectDisk(c)
	case "net":
		return collectNet(c)
	case "gpu":
		return collectGPU(c)
	}
	return nil
}

// procUpdateCmd blocks until the collector's process cache is refreshed, then
// returns a subsystemMsg carrying the cached snapshot. Returning the command
// again from Update creates a perpetual listener: one background goroutine does
// the expensive walk, the UI simply re-renders whenever the cache changes.
func procUpdateCmd(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		<-c.ProcUpdate()
		return subsystemMsg{part: "proc", snap: collector.Snapshot{Proc: c.ProcCache()}}
	}
}

// netProcUpdateCmd blocks until the per-process network cache is refreshed,
// then returns a lightweight subsystemMsg so the network card can re-render
// without re-running the aggregate network collector.
func netProcUpdateCmd(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		<-c.NetProcUpdate()
		return subsystemMsg{part: "netprocs", snap: collector.Snapshot{Net: collector.NetStat{
			TopProcs:       c.NetProcCache(),
			ProcsSupported: c.NetProcSupported(),
		}}}
	}
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

// scheduleRecollect waits collectInterval after a subsystem finished, then asks
// for that same subsystem to be collected again. Because the timer starts only
// once the previous collection has completed, a subsystem never runs two
// collections concurrently.
func scheduleRecollect(part string) tea.Cmd {
	return tea.Tick(collectInterval, func(time.Time) tea.Msg { return recollectMsg{part: part} })
}
