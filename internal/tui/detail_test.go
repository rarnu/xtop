package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"xtop/internal/collector"
)

func TestDetailDialogSizeAndColors(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.recompute()

	m.mergeSnap(collector.Snapshot{
		Mem: collector.MemStat{Total: 16 << 30, Used: 8 << 30, Cached: 2 << 30, Free: 6 << 30},
		Proc: collector.ProcStat{
			All: []collector.ProcInfo{{PID: 1234, Command: "firefox", User: "alice", Status: "R", CPU: 12.5, MemRSS: 1 << 30}},
			TopMem: []collector.ProcInfo{{PID: 1234, Command: "firefox", User: "alice", Status: "R", CPU: 12.5, MemRSS: 1 << 30}},
		},
	})
	m.recompute()

	m2, cmd := m.Update(tea.MouseMsg{X: 10, Y: 27, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	mm := m2.(*model)
	if cmd != nil {
		m3, _ := mm.Update(cmd())
		mm = m3.(*model)
	}
	v := mm.View()

	lines := strings.Split(v, "\n")
	var topY, bottomY int
	for i, l := range lines {
		plain := stripANSI(l)
		if strings.Contains(plain, "进程详情") {
			if topY == 0 {
				topY = i
			}
		}
		if strings.Contains(plain, "结束") && strings.Contains(plain, "强制结束") {
			bottomY = i
		}
	}
	boxH := bottomY - topY + 1
	if boxH > 12 {
		t.Fatalf("dialog too tall: %d", boxH)
	}

	// Check that the dialog title line has colored title text.
	titleLine := lines[topY]
	if !strings.Contains(titleLine, titleStyle.Render("进程详情")) {
		t.Fatalf("title line missing title style: %q", titleLine)
	}

	_ = lipgloss.Width
	_ = tea.MouseMsg{}
}
