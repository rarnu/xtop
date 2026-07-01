package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/collector"
)

const histLen = 80 // sparkline history samples

// confirmState tracks a pending KILL / FORCE KILL confirmation.
type confirmState struct {
	active bool
	force  bool
	pid    int32
	name   string
}

type model struct {
	col      *collector.Collector
	snap     collector.Snapshot
	haveSnap bool

	width, height int

	// sparkline history
	cpuHist  []float64
	downHist []float64
	gpuHist  []float64

	// per-card body scroll offsets
	scrollBars cardScrollBars

	// dashboard state
	dashLines []string
	btnLine   int // full-content line index of the "open process manager" button
	btnX0     int
	btnX1     int
	btnFound  bool

	scheduled bool // true while waiting for the next 5s tick

	// process modal state (feature 7)
	modal      bool
	procRows   []collector.ProcInfo
	sortCol    sortCol
	sortAsc    bool
	procSel    int
	procScroll int

	confirm confirmState
}

// New builds the root model. It implements tea.Model via a pointer receiver.
func New() *model {
	return &model{
		col:     collector.New(),
		sortCol: sortCPU,
		sortAsc: false,
	}
}

func (m *model) Init() tea.Cmd {
	return collectAllCmd(m.col)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.recompute()
		return m, nil

	case subsystemMsg:
		m.mergeSnap(msg.snap)
		m.haveSnap = true
		m.pushHistoryFor(msg.part)
		if m.modal {
			m.rebuildProcRows()
		}
		m.recompute()
		// Schedule the next 5-second refresh once per collection round.
		if !m.scheduled {
			m.scheduled = true
			return m, scheduleTick()
		}
		return m, nil

	case tickMsg:
		m.scheduled = false
		return m, collectAllCmd(m.col)

	case tea.KeyMsg:
		if m.modal {
			return m.updateModalKey(msg)
		}
		return m.updateDashKey(msg)

	case tea.MouseMsg:
		if m.modal {
			return m.updateModalMouse(msg)
		}
		return m.updateDashMouse(msg)
	}
	return m, nil
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "正在初始化 XTOP..."
	}
	if m.modal {
		return renderProcModal(m)
	}

	lines := m.dashLines
	contentH := m.height - footerHeight
	if len(lines) > contentH {
		lines = lines[:contentH]
	}
	if len(lines) < contentH {
		// Should not happen because geometry sizes cards to fit.
		for len(lines) < contentH {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n") + "\n" + m.dashFooter()
}

const footerHeight = 1 // status/help bar at the very bottom

// ---- dashboard input ---------------------------------------------------

func (m *model) updateDashKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "p", "enter":
		m.openModal()
		return m, nil
	}
	return m, nil
}

func (m *model) updateDashMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		if key, ok := m.hitCard(msg.X, msg.Y); ok {
			cs := &m.scrollBars[key]
			delta := 1
			if msg.Button == tea.MouseButtonWheelUp {
				delta = -1
			}
			cs.offset += delta
			m.recompute()
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if m.btnFound {
			if msg.Y >= m.btnLine-1 && msg.Y <= m.btnLine+1 &&
				msg.X >= m.btnX0-2 && msg.X <= m.btnX1+2 {
				m.openModal()
				return m, nil
			}
		}
	}
	return m, nil
}

// ---- modal input -------------------------------------------------------

func (m *model) updateModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		switch msg.String() {
		case "y", "Y":
			return m, m.execConfirm()
		case "n", "N", "esc":
			m.confirm.active = false
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "q":
		m.closeModal()
	case "up":
		m.moveSel(-1)
	case "down":
		m.moveSel(1)
	case "pgup":
		m.moveSel(-m.modalVisibleRows())
	case "pgdown":
		m.moveSel(m.modalVisibleRows())
	case "home":
		m.moveSelTo(0)
	case "end":
		m.moveSelTo(len(m.procRows) - 1)
	case "1":
		m.setSort(sortPID)
	case "2":
		m.setSort(sortUser)
	case "3":
		m.setSort(sortStat)
	case "4":
		m.setSort(sortCPU)
	case "5":
		m.setSort(sortMem)
	case "6":
		m.setSort(sortStart)
	case "7":
		m.setSort(sortCmd)
	case "k":
		m.startConfirm(false)
	case "K", "f", "F":
		m.startConfirm(true)
	}
	return m, nil
}

func (m *model) updateModalMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.moveSel(-1)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.moveSel(1)
		return m, nil
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}

	cols, actX := procColumns(m.width)
	x, y := msg.X, msg.Y

	if y == 1 { // header: click to sort
		for _, c := range cols {
			if x >= c.x && x < c.x+c.w {
				m.setSort(c.key)
				return m, nil
			}
		}
		return m, nil
	}

	if y >= 2 {
		idx := m.procScroll + (y - 2)
		if idx >= 0 && idx < len(m.procRows) {
			termX0, termX1, killX0, killX1 := actionHit(actX)
			switch {
			case x >= termX0 && x < termX1:
				m.procSel = idx
				m.startConfirm(false)
			case x >= killX0 && x < killX1:
				m.procSel = idx
				m.startConfirm(true)
			default:
				m.procSel = idx
				m.ensureProcVisible()
			}
		}
	}
	return m, nil
}

// ---- state helpers -----------------------------------------------------

func (m *model) recompute() {
	m.dashLines = m.renderDashboard()
	if line, x0, x1, ok := findText(m.dashLines, openProcNeedle); ok {
		m.btnLine, m.btnX0, m.btnX1, m.btnFound = line, x0, x1, true
	} else {
		m.btnFound = false
	}
}

func (m *model) dashFooter() string {
	help := helpBarStyle.Render("P/Enter 进程管理 · 鼠标滚轮滚动卡片 · q 退出")
	status := ""
	if m.haveSnap {
		memPct := 0.0
		if m.snap.Mem.Total > 0 {
			memPct = float64(m.snap.Mem.Used) / float64(m.snap.Mem.Total) * 100
		}
		status = faintStyle.Render(fmt.Sprintf("CPU %.0f%% · 内存 %.0f%% · %s",
			m.snap.CPU.Overall, memPct, m.snap.Time.Format("15:04:05")))
	}
	return joinLR(help, status, m.width)
}


func (m *model) mergeSnap(s collector.Snapshot) {
	if len(s.CPU.PerCore) > 0 || s.CPU.Overall != 0 {
		m.snap.CPU = s.CPU
	}
	if s.Mem.Total > 0 {
		m.snap.Mem = s.Mem
	}
	if len(s.Disk.Mounts) > 0 {
		m.snap.Disk = s.Disk
	}
	if s.Net.TotalUpload > 0 || s.Net.TotalDownload > 0 {
		m.snap.Net = s.Net
	}
	if s.GPU.Available || len(s.GPU.Cards) > 0 {
		m.snap.GPU = s.GPU
	}
	if len(s.Proc.All) > 0 {
		m.snap.Proc = s.Proc
	}
	m.snap.Time = s.Time
}

func (m *model) pushHistoryFor(part string) {
	switch part {
	case "cpu":
		m.cpuHist = pushHist(m.cpuHist, m.snap.CPU.Overall)
	case "net":
		m.downHist = pushHist(m.downHist, m.snap.Net.DownloadPerSec)
	case "gpu":
		m.gpuHist = pushHist(m.gpuHist, gpuAvgLoad(m.snap.GPU))
	}
}

func (m *model) pushHistory() {
	m.cpuHist = pushHist(m.cpuHist, m.snap.CPU.Overall)
	m.downHist = pushHist(m.downHist, m.snap.Net.DownloadPerSec)
	m.gpuHist = pushHist(m.gpuHist, gpuAvgLoad(m.snap.GPU))
}

func pushHist(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > histLen {
		h = h[len(h)-histLen:]
	}
	return h
}

func gpuAvgLoad(g collector.GPUStat) float64 {
	var sum float64
	var n int
	for _, c := range g.Cards {
		if c.LoadPct >= 0 {
			sum += c.LoadPct
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// ---- modal helpers -----------------------------------------------------

func (m *model) openModal() {
	m.modal = true
	m.procSel = 0
	m.procScroll = 0
	m.confirm.active = false
	m.rebuildProcRows()
}

func (m *model) closeModal() {
	m.modal = false
	m.confirm.active = false
}

func (m *model) modalVisibleRows() int {
	return maxInt(m.height-3, 1)
}

func (m *model) selectedPID() int32 {
	if m.procSel >= 0 && m.procSel < len(m.procRows) {
		return m.procRows[m.procSel].PID
	}
	return 0
}

func (m *model) rebuildProcRows() {
	prevPID := m.selectedPID()
	m.procRows = append([]collector.ProcInfo(nil), m.snap.Proc.All...)
	sortProcs(m.procRows, m.sortCol, m.sortAsc)

	if prevPID != 0 {
		for i, pr := range m.procRows {
			if pr.PID == prevPID {
				m.procSel = i
				break
			}
		}
	}
	m.procSel = clampInt(m.procSel, 0, maxInt(len(m.procRows)-1, 0))
	m.ensureProcVisible()
}

func (m *model) setSort(col sortCol) {
	if m.sortCol == col {
		m.sortAsc = !m.sortAsc
	} else {
		m.sortCol = col
		m.sortAsc = false
	}
	m.rebuildProcRows()
}

func (m *model) moveSel(delta int) { m.moveSelTo(m.procSel + delta) }

func (m *model) moveSelTo(idx int) {
	if len(m.procRows) == 0 {
		return
	}
	m.procSel = clampInt(idx, 0, len(m.procRows)-1)
	m.ensureProcVisible()
}

func (m *model) ensureProcVisible() {
	visible := m.modalVisibleRows()
	if m.procSel < m.procScroll {
		m.procScroll = m.procSel
	}
	if m.procSel >= m.procScroll+visible {
		m.procScroll = m.procSel - visible + 1
	}
	maxScroll := maxInt(len(m.procRows)-visible, 0)
	m.procScroll = clampInt(m.procScroll, 0, maxScroll)
}

func (m *model) startConfirm(force bool) {
	if m.procSel < 0 || m.procSel >= len(m.procRows) {
		return
	}
	pr := m.procRows[m.procSel]
	m.confirm = confirmState{active: true, force: force, pid: pr.PID, name: pr.Command}
}

func (m *model) execConfirm() tea.Cmd {
	pid := m.confirm.pid
	force := m.confirm.force
	m.confirm.active = false
	return func() tea.Msg {
		if force {
			_ = collector.ForceKill(pid)
		} else {
			_ = collector.Terminate(pid)
		}
		return nil
	}
}
