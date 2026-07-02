package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"xtop/internal/collector"
)

func TestDetailModal(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()

	m.mergeSnap(collector.Snapshot{
		Mem: collector.MemStat{Total: 16 << 30, Used: 8 << 30, Cached: 2 << 30, Free: 6 << 30},
		Proc: collector.ProcStat{
			All: []collector.ProcInfo{
				{PID: 1234, Command: "firefox", User: "alice", Status: "R", CPU: 12.5, MemRSS: 1 << 30},
			},
			TopMem: []collector.ProcInfo{
				{PID: 1234, Command: "firefox", User: "alice", Status: "R", CPU: 12.5, MemRSS: 1 << 30},
			},
		},
	})
	m.recompute()

	m2, _ := m.Update(tea.MouseMsg{X: 10, Y: 27, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	mm := m2.(*model)
	_ = mm.View()
	t.Logf("killX0=%d killX1=%d btnY=%d", mm.detail.killX0, mm.detail.killX1, mm.detail.btnY)

	m3, _ := mm.Update(tea.MouseMsg{X: mm.detail.killX0, Y: mm.detail.btnY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	mm3 := m3.(*model)
	if !mm3.confirm.active {
		t.Fatalf("confirm should be active after clicking kill button at y=%d", mm.detail.btnY)
	}
}
