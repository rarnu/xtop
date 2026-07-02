package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"xtop/internal/collector"
)

func TestDiskCardHiddenZeroRateProcNoClick(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()

	// Two visible procs plus two hidden zero-rate procs.
	procs := []collector.ProcInfo{
		{PID: 1, Command: "visible1", DiskReadPerSec: 1000, DiskWritePerSec: 0},
		{PID: 2, Command: "visible2", DiskReadPerSec: 0, DiskWritePerSec: 500},
		{PID: 99, Command: "hidden1", DiskReadPerSec: 0, DiskWritePerSec: 0},
		{PID: 100, Command: "hidden2", DiskReadPerSec: 0, DiskWritePerSec: 0},
	}

	m.mergeSnap(collector.Snapshot{
		Disk: collector.DiskStat{
			TotalBytes: 1 << 40,
			UsedBytes:  500 << 30,
			Mounts: []collector.DiskMount{
				{Mountpoint: "/System", Fstype: "apfs", Total: 1 << 40, Used: 500 << 30, Free: 500 << 30, UsedPercent: 50, ReadPerSec: 100, WritePerSec: 200},
			},
		},
		Proc: collector.ProcStat{
			All:           procs,
			TopDisk:       procs,
			DiskSupported: true,
		},
	})
	m.recompute()

	if len(m.miniMeta[cardDisk].pids) != 2 {
		t.Fatalf("expected 2 visible disk procs in miniMeta, got %d: %v", len(m.miniMeta[cardDisk].pids), m.miniMeta[cardDisk].pids)
	}

	// Disk card is first row, col 1. Body starts at y=3.
	bodyY0 := 3
	diskStart := m.miniMeta[cardDisk].startBodyY

	// Click on each displayed row; should match visible procs.
	for dy := 0; dy < 2; dy++ {
		mCopy := *m
		y := bodyY0 + diskStart + dy
		m2, _ := mCopy.Update(tea.MouseMsg{X: 50, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		mm := m2.(*model)
		if !mm.detail.active {
			t.Fatalf("row dy=%d (y=%d) should open detail", dy, y)
		}
		wantPID := m.miniMeta[cardDisk].pids[dy]
		if mm.detail.proc.PID != wantPID {
			t.Fatalf("row dy=%d opened pid=%d, want %d", dy, mm.detail.proc.PID, wantPID)
		}
	}

	// Click below visible rows (blank area); should not open detail.
	for dy := 2; dy < 10; dy++ {
		mCopy := *m
		y := bodyY0 + diskStart + dy
		m2, _ := mCopy.Update(tea.MouseMsg{X: 50, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		mm := m2.(*model)
		if mm.detail.active {
			t.Fatalf("blank area dy=%d (y=%d) should not open detail, got pid=%d name=%q", dy, y, mm.detail.proc.PID, mm.detail.proc.Command)
		}
	}
}
