package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"xtop/internal/collector"
)

const histLen = 80 // sparkline history samples

// temporary kill-switches for profiling. Set to true to disable the network
// card and/or the process card (and background process collection). Useful for
// isolating whether nettop / the process walk is responsible for UI lag.
const (
	disableNet  = false
	disableProc = false
)

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

	// actual rendered card size (outer, incl. border), captured by
	// renderDashboard and used by hitCard to map mouse coordinates.
	cardW, cardH int

	// dashboard state
	dashLines []string
	btnLine   int // full-content line index of the "open process manager" button
	btnX0     int
	btnX1     int
	btnFound  bool

	// process modal state (feature 7)
	modal      bool
	procRows   []collector.ProcInfo
	sortCol    sortCol
	sortAsc    bool
	procSel    int
	procScroll int

	confirm confirmState

	// scrollbar drag state
	dragScroll    *cardScroll // nil when not dragging
	dragCard      cardKey     // card being scrolled
	dragStartY    int         // terminal y where drag started
	dragStartOff  int         // scroll offset when drag started
	dragThumbY0   int         // scrollbar thumb top at drag start
	dragThumbY1   int         // scrollbar thumb bottom at drag start
	dragBodyY0    int         // card body top y in terminal
	dragBodyH     int         // card body height in terminal
	dragContentH  int         // total content lines for the dragged card

	// process detail popup state (feature: click mini-list process)
	detail detailState

	// geometry of each card's mini process list, rebuilt after renderDashboard.
	miniMeta [6]miniListMeta
}

// detailState holds the popup shown when a mini-list process row is clicked.
type detailState struct {
	active bool
	proc   detailProc
	source cardKey

	// button hit boxes in terminal coordinates, set by renderDetailModal.
	btnY              int
	killX0, killX1    int
	forceX0, forceX1  int
}

// detailProc is the information shown in the detail popup.
type detailProc struct {
	PID                 int32
	Command             string
	User                string
	Status              string
	CPU                 float64
	MemRSS              uint64
	NetUp, NetDown      float64
	DiskRead, DiskWrite float64
	GPUMem              uint64
}

// miniListMeta describes the clickable process list inside one dashboard card.
type miniListMeta struct {
	hasList    bool
	startBodyY int // first list row relative to card body top (before scroll)
	pids       []int32
	names      []string
}

// New builds the root model. It implements tea.Model via a pointer receiver.
func New() *model {
	m := &model{
		col:     collector.New(),
		sortCol: sortCPU,
		sortAsc: false,
	}
	if !disableProc {
		m.col.StartProcLoop()
	}
	if !disableNet {
		m.col.StartNetProcLoop()
	}
	return m
}

func (m *model) Init() tea.Cmd {
	// CPU/mem/disk/GPU are still collected on demand via tea commands. Process
	// data is maintained by a dedicated collector goroutine writing to a cache;
	// the UI simply listens for cache updates and re-renders.
	cmds := []tea.Cmd{
		collectCPU(m.col),
		collectMem(m.col),
		collectDisk(m.col),
		collectGPU(m.col),
	}
	if !disableNet {
		cmds = append(cmds, collectNet(m.col), netProcUpdateCmd(m.col))
	}
	if !disableProc {
		cmds = append(cmds, procUpdateCmd(m.col))
	}
	return tea.Batch(cmds...)
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
		// Cache-backed subsystems re-subscribe to the next cache update.
		// On-demand subsystems schedule their next collection.
		switch msg.part {
		case "proc":
			return m, procUpdateCmd(m.col)
		case "netprocs":
			return m, netProcUpdateCmd(m.col)
		case "net":
			if disableNet {
				return m, nil
			}
			return m, scheduleRecollect(msg.part)
		default:
			return m, scheduleRecollect(msg.part)
		}

	case recollectMsg:
		if msg.part == "net" && disableNet {
			return m, nil
		}
		return m, collectPart(m.col, msg.part)

	case tea.KeyMsg:
		if m.detail.active {
			return m.updateDetailKey(msg)
		}
		if m.modal {
			return m.updateModalKey(msg)
		}
		return m.updateDashKey(msg)

	case tea.MouseMsg:
		if m.detail.active {
			return m.updateDetailMouse(msg)
		}
		if m.modal {
			return m.updateModalMouse(msg)
		}
		return m.updateDashMouse(msg)
	}
	return m, nil
}

// scrollbarHit returns whether (x,y) is on a scrollbar, and if so which card,
// the thumb bounds, scroll-region bounds, and content height. The model must have
// rendered dashLines first so cardW/cardH and dashLines are current.
func (m *model) scrollbarHit(x, y int) (key cardKey, onThumb bool, thumbY0, thumbY1, regionY0, regionH, contentH int, ok bool) {
	key, hit := m.hitCard(x, y)
	if !hit {
		return -1, false, 0, 0, 0, 0, 0, false
	}
	cs := &m.scrollBars[key]
	if cs.max <= 0 {
		return -1, false, 0, 0, 0, 0, 0, false
	}

	// Find the card's top-left in terminal coordinates.
	col := int(key) % gridCols
	row := int(key) / gridCols
	x0 := col * (m.cardW + gapH)
	y0 := row * m.cardH
	bodyY0 := y0 + 3 // top border + header + divider
	bodyH := m.cardH - 4

	// Split cards (memory / network) render the scrollbar only over the list
	// region, not the pinned summary. Full-card scrollers use the whole body.
	fixedH := 0
	if key == cardMem || key == cardNet {
		fixedH = 5
	}
	regionY0 = bodyY0 + fixedH
	regionH = bodyH - fixedH
	if regionH <= 0 {
		return -1, false, 0, 0, 0, 0, 0, false
	}

	// Scrollbar is on the rightmost content column (content starts at x0+2 to
	// leave room for the left border and left padding).
	sbX := x0 + m.cardW - 3
	if x != sbX {
		return -1, false, 0, 0, 0, 0, 0, false
	}
	if y < regionY0 || y >= regionY0+regionH {
		return -1, false, 0, 0, 0, 0, 0, false
	}

	contentH = cs.max + regionH
	thumbRatio := float64(regionH) / float64(contentH)
	thumbH := int(float64(regionH) * thumbRatio)
	if thumbH < 1 {
		thumbH = 1
	}
	thumbPos := 0
	if cs.max > 0 {
		thumbPos = int(float64(cs.offset) / float64(cs.max) * float64(regionH-thumbH))
	}
	thumbY0 = regionY0 + thumbPos
	thumbY1 = thumbY0 + thumbH
	onThumb = y >= thumbY0 && y < thumbY1
	return key, onThumb, thumbY0, thumbY1, regionY0, regionH, contentH, true
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "正在初始化 XTOP..."
	}

	// Always render the dashboard as the bottom layer.
	lines := m.dashLines
	contentH := m.height - footerHeight
	if len(lines) > contentH {
		lines = lines[:contentH]
	}
	if len(lines) < contentH {
		for len(lines) < contentH {
			lines = append(lines, "")
		}
	}
	base := strings.Join(lines, "\n") + "\n" + m.dashFooter()

	switch {
	case m.detail.active:
		return overlayDetailModal(m, base)
	case m.modal:
		if m.confirm.active {
			return overlayConfirmOnBase(m, base)
		}
		return renderProcModal(m)
	default:
		return base
	}
}

const footerHeight = 1 // status/help bar at the very bottom

// ---- dashboard input ---------------------------------------------------

func (m *model) updateDashKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.stopLoops()
		return m, tea.Quit
	case "p", "enter":
		return m, m.openModal()
	}
	return m, nil
}

func (m *model) stopLoops() {
	m.col.StopProcLoop()
	m.col.StopNetProcLoop()
}

func (m *model) updateDashMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Drag in progress: any motion or release updates the scroll.
	if m.dragScroll != nil {
		if msg.Action == tea.MouseActionRelease {
			m.dragScroll = nil
			return m, nil
		}
		if msg.Button == tea.MouseButtonLeft || msg.Button == tea.MouseButtonNone {
			m.updateDragScroll(msg.Y)
			m.recompute()
		}
		return m, nil
	}

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
		// Scrollbar drag takes precedence over buttons/rows.
		if key, onThumb, thumbY0, thumbY1, bodyY0, bodyH, contentH, ok := m.scrollbarHit(msg.X, msg.Y); ok && onThumb {
			m.dragScroll = &m.scrollBars[key]
			m.dragCard = key
			m.dragStartY = msg.Y
			m.dragStartOff = m.scrollBars[key].offset
			m.dragThumbY0 = thumbY0
			m.dragThumbY1 = thumbY1
			m.dragBodyY0 = bodyY0
			m.dragBodyH = bodyH
			m.dragContentH = contentH
			m.recompute()
			return m, nil
		}

		// Mini-list click opens the process detail popup.
		if pid, name, source, ok := m.miniListHit(msg.X, msg.Y); ok {
			return m, m.openDetail(pid, name, source)
		}

		if m.btnFound {
			if msg.Y >= m.btnLine-1 && msg.Y <= m.btnLine+1 &&
				msg.X >= m.btnX0-2 && msg.X <= m.btnX1+2 {
				return m, m.openModal()
			}
		}
	}
	return m, nil
}

func (m *model) updateDragScroll(y int) {
	if m.dragScroll == nil || m.dragBodyH <= 0 || m.dragContentH <= m.dragBodyH {
		return
	}
	thumbH := m.dragThumbY1 - m.dragThumbY0
	trackH := m.dragBodyH
	maxThumbPos := trackH - thumbH
	if maxThumbPos <= 0 {
		m.dragScroll.offset = 0
		return
	}

	// Map thumb top position to offset.
	thumbTop := y - m.dragStartY + m.dragThumbY0
	if thumbTop < m.dragBodyY0 {
		thumbTop = m.dragBodyY0
	}
	if thumbTop > m.dragBodyY0+maxThumbPos {
		thumbTop = m.dragBodyY0 + maxThumbPos
	}
	thumbPos := thumbTop - m.dragBodyY0
	m.dragScroll.offset = int(float64(thumbPos) / float64(maxThumbPos) * float64(m.dragScroll.max))
	if m.dragScroll.offset < 0 {
		m.dragScroll.offset = 0
	}
	if m.dragScroll.offset > m.dragScroll.max {
		m.dragScroll.offset = m.dragScroll.max
	}
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
	m.rebuildMiniListMeta()
	if line, x0, x1, ok := findText(m.dashLines, openProcNeedle); ok {
		m.btnLine, m.btnX0, m.btnX1, m.btnFound = line, x0, x1, true
	} else {
		m.btnFound = false
	}
}

// rebuildMiniListMeta records the on-screen location of each card's process
// mini-list so mouse clicks can be mapped back to a PID.
func (m *model) rebuildMiniListMeta() {
	for i := range m.miniMeta {
		m.miniMeta[i] = miniListMeta{}
	}

	// Memory and network cards use a fixed summary height with the list below it.
	const splitFixedH = 5 // header lines before the process list
	m.miniMeta[cardMem] = miniListMeta{
		hasList:    true,
		startBodyY: splitFixedH,
		pids:       procPIDs(m.snap.Proc.TopMem),
		names:      procNames(m.snap.Proc.TopMem),
	}
	m.miniMeta[cardNet] = miniListMeta{
		hasList:    m.snap.Net.ProcsSupported && len(m.snap.Net.TopProcs) > 0,
		startBodyY: splitFixedH,
		pids:       netPIDs(m.snap.Net.TopProcs),
		names:      netNames(m.snap.Net.TopProcs),
	}

	// Disk card: list starts after the per-mount blocks, a blank line and header.
	diskStart := 0
	for i := range m.snap.Disk.Mounts {
		if i > 0 {
			diskStart++ // blank separator between mounts
		}
		diskStart += 5 // mount head (1) + tank block (4)
	}
	diskStart += 2 // blank + mini-list header
	m.miniMeta[cardDisk] = miniListMeta{
		hasList:    m.snap.Proc.DiskSupported && len(m.snap.Proc.TopDisk) > 0,
		startBodyY: diskStart,
		pids:       procPIDs(m.snap.Proc.TopDisk),
		names:      procNames(m.snap.Proc.TopDisk),
	}

	// GPU card: list starts after the per-GPU stats, a blank line and header.
	gpuStart := 0
	for i, c := range m.snap.GPU.Cards {
		if i > 0 {
			gpuStart++ // blank separator between GPUs
		}
		gpuStart += 5 // name, power, mem, temp, load
		if c.MemTotal > 0 {
			gpuStart++ // memory bar
		}
	}
	gpuStart += 2 // blank + mini-list header
	m.miniMeta[cardGPU] = miniListMeta{
		hasList:    m.snap.GPU.Available && m.snap.GPU.ProcsSupported && len(m.snap.GPU.TopProcs) > 0,
		startBodyY: gpuStart,
		pids:       gpuPIDs(m.snap.GPU.TopProcs),
		names:      gpuNames(m.snap.GPU.TopProcs),
	}
}

func procPIDs(list []collector.ProcInfo) []int32 {
	out := make([]int32, len(list))
	for i, p := range list {
		out[i] = p.PID
	}
	return out
}

func procNames(list []collector.ProcInfo) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.Command
	}
	return out
}

func netPIDs(list []collector.NetProc) []int32 {
	out := make([]int32, len(list))
	for i, p := range list {
		out[i] = p.PID
	}
	return out
}

func netNames(list []collector.NetProc) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.Command
	}
	return out
}

func gpuPIDs(list []collector.GPUProc) []int32 {
	out := make([]int32, len(list))
	for i, p := range list {
		out[i] = p.PID
	}
	return out
}

func gpuNames(list []collector.GPUProc) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.Command
	}
	return out
}

// miniListHit maps a mouse coordinate to a PID in a card's process mini-list.
func (m *model) miniListHit(x, y int) (pid int32, name string, source cardKey, ok bool) {
	key, hit := m.hitCard(x, y)
	if !hit {
		return 0, "", -1, false
	}
	meta := &m.miniMeta[key]
	if !meta.hasList || len(meta.pids) == 0 {
		return 0, "", -1, false
	}

	row := int(key) / gridCols
	bodyY0 := row*m.cardH + 3 // top border + header + divider
	bodyY := y - bodyY0
	scroll := m.scrollBars[key].offset
	rowIdx := bodyY - meta.startBodyY + scroll
	if rowIdx < 0 || rowIdx >= len(meta.pids) {
		return 0, "", -1, false
	}
	return meta.pids[rowIdx], meta.names[rowIdx], key, true
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
	if s.Net.TotalUpload > 0 || s.Net.TotalDownload > 0 || len(s.Net.TopProcs) > 0 {
		if s.Net.TotalUpload > 0 || s.Net.TotalDownload > 0 {
			m.snap.Net = s.Net
		} else {
			m.snap.Net.TopProcs = s.Net.TopProcs
			m.snap.Net.ProcsSupported = s.Net.ProcsSupported
		}
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

func (m *model) openModal() tea.Cmd {
	m.modal = true
	m.procSel = 0
	m.procScroll = 0
	m.confirm.active = false
	m.rebuildProcRows()
	return nil
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
	m.closeDetail() // also closes the detail popup if it was open
	return func() tea.Msg {
		if force {
			_ = collector.ForceKill(pid)
		} else {
			_ = collector.Terminate(pid)
		}
		return nil
	}
}

// ---- detail popup helpers ----------------------------------------------

func (m *model) openDetail(pid int32, name string, source cardKey) tea.Cmd {
	m.detail = detailState{active: true, proc: m.buildDetail(pid, name, source), source: source}
	return nil
}

func (m *model) closeDetail() {
	m.detail.active = false
	m.confirm.active = false
}

func (m *model) buildDetail(pid int32, fallback string, source cardKey) detailProc {
	d := detailProc{PID: pid, Command: fallback}
	for _, p := range m.snap.Proc.All {
		if p.PID == pid {
			d.Command = p.Command
			d.User = p.User
			d.Status = p.Status
			d.CPU = p.CPU
			d.MemRSS = p.MemRSS
			d.DiskRead = p.DiskReadPerSec
			d.DiskWrite = p.DiskWritePerSec
			break
		}
	}
	switch source {
	case cardMem:
		for _, p := range m.snap.Proc.TopMem {
			if p.PID == pid {
				d.MemRSS = p.MemRSS
				break
			}
		}
	case cardNet:
		for _, p := range m.snap.Net.TopProcs {
			if p.PID == pid {
				d.NetUp = p.UploadPerSec
				d.NetDown = p.DownloadPerSec
				if d.Command == "" {
					d.Command = p.Command
				}
				break
			}
		}
	case cardDisk:
		for _, p := range m.snap.Proc.TopDisk {
			if p.PID == pid {
				d.DiskRead = p.DiskReadPerSec
				d.DiskWrite = p.DiskWritePerSec
				break
			}
		}
	case cardGPU:
		for _, p := range m.snap.GPU.TopProcs {
			if p.PID == pid {
				d.GPUMem = p.MemBytes
				if d.Command == "" {
					d.Command = p.Command
				}
				break
			}
		}
	}
	return d
}

func (m *model) startDetailConfirm(force bool) {
	m.confirm = confirmState{active: true, force: force, pid: m.detail.proc.PID, name: m.detail.proc.Command}
}

func (m *model) updateDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.closeDetail()
	case "k":
		m.startDetailConfirm(false)
	case "K", "f", "F":
		m.startDetailConfirm(true)
	}
	return m, nil
}

func (m *model) updateDetailMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		return m, nil
	}
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	x, y := msg.X, msg.Y

	// Click outside the popup closes it.
	boxW, boxH := detailBoxSize(m.width, m.height)
	x0 := (m.width - boxW) / 2
	x1 := x0 + boxW
	y0 := (m.height - boxH) / 2
	y1 := y0 + boxH
	if x < x0 || x >= x1 || y < y0 || y >= y1 {
		m.closeDetail()
		return m, nil
	}

	if y == m.detail.btnY {
		if x >= m.detail.killX0 && x < m.detail.killX1 {
			m.startDetailConfirm(false)
			return m, nil
		}
		if x >= m.detail.forceX0 && x < m.detail.forceX1 {
			m.startDetailConfirm(true)
			return m, nil
		}
	}
	return m, nil
}
