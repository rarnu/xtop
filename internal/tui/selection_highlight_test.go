package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"xtop/internal/collector"
)

func TestSelectionHighlight(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()

	m.mergeSnap(collector.Snapshot{
		Disk: collector.DiskStat{
			TotalBytes: 1 << 40,
			UsedBytes:  500 << 30,
			Mounts: []collector.DiskMount{
				{Mountpoint: "/System", Fstype: "apfs", Total: 1 << 40, Used: 500 << 30, Free: 500 << 30, UsedPercent: 50, ReadPerSec: 100, WritePerSec: 200},
			},
		},
		Proc: collector.ProcStat{
			All: []collector.ProcInfo{
				{PID: 1, Command: "visible1", DiskReadPerSec: 1000, DiskWritePerSec: 0},
				{PID: 2, Command: "visible2", DiskReadPerSec: 0, DiskWritePerSec: 500},
			},
			TopDisk: []collector.ProcInfo{
				{PID: 1, Command: "visible1", DiskReadPerSec: 1000, DiskWritePerSec: 0},
				{PID: 2, Command: "visible2", DiskReadPerSec: 0, DiskWritePerSec: 500},
			},
			DiskSupported: true,
		},
	})
	m.recompute()

	// Click on the first disk process row (y=10).
	m2, cmd := m.Update(tea.MouseMsg{X: 50, Y: 10, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	mm := m2.(*model)
	if !mm.selected.active || mm.selected.pid != 1 {
		t.Fatalf("expected selected pid 1, got active=%v pid=%d", mm.selected.active, mm.selected.pid)
	}
	// Detail should NOT open on the same frame as the click.
	if mm.detail.active {
		t.Fatal("detail should not be active immediately after click")
	}
	// The returned cmd should produce the delayed open message.
	if cmd == nil {
		t.Fatal("expected delayed open detail cmd")
	}

	// Simulate the next frame by sending the delayed message.
	m3, _ := mm.Update(cmd())
	mm3 := m3.(*model)
	if !mm3.detail.active || mm3.detail.proc.PID != 1 {
		t.Fatalf("expected detail open for pid 1 after delay, got active=%v pid=%d", mm3.detail.active, mm3.detail.proc.PID)
	}

	v := mm3.View()
	if !strings.Contains(v, "visible1") {
		t.Fatal("view missing visible1")
	}
	if !strings.Contains(v, selectedRowStyle.Render("")) {
		t.Fatal("selected row style not present in view")
	}
}
