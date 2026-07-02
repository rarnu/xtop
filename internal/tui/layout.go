package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	gridCols = 3
	gridRows = 2
	gapH     = 1 // horizontal gap between cards
	gapV     = 1 // vertical gap between cards
)

// dashboardGeometry returns the fixed inner dimensions for every card in the
// 2x3 grid. Cards resize automatically with the terminal window.
func (m *model) dashboardGeometry() (innerW, innerH int) {
	// Card outer width = (terminal width - gaps) / 3.
	outerW := (m.width - (gridCols-1)*gapH) / gridCols
	innerW = outerW - 4 // border(2) + horizontal padding(2)
	if innerW < 8 {
		innerW = 8
	}

	// Available height: terminal height minus the reserved bottom offset and
	// the vertical gap between the two card rows. Each card gets half of the
	// remainder. offset is in terminal cells (rows); 64px is approximated as
	// 3 terminal rows.
	const offsetRows = 3
	availH := m.height - footerHeight - offsetRows - (gridRows-1)*gapV
	if availH < gridRows*4 {
		availH = gridRows * 4 // minimum 4 rows per card outer height
	}
	outerH := availH / gridRows
	innerH = outerH - 2 // border only (padding is horizontal)
	if innerH < 2 {
		innerH = 2
	}
	return innerW, innerH
}

// renderDashboard lays the six cards in a fixed 2x3 grid:
//   CPU    Disk   GPU
//   Mem    Net    Proc
func (m *model) renderDashboard() []string {
	if m.width < 8 {
		return []string{""}
	}
	w, h := m.dashboardGeometry()

	var netCardStr, procCardStr string
	if disableNet {
		netCardStr = disabledCard(w, h, "◍", "网络", "已停用")
	} else {
		netCardStr = netCard(m.snap.Net, m.downHist, w, h, false, &m.scrollBars[cardNet])
	}
	if disableProc {
		procCardStr = disabledCard(w, h, "☰", "进程", "已停用")
	} else {
		procCardStr = procCard(m.snap.Proc, w, h, false, &m.scrollBars[cardProc])
	}

	cards := []string{
		cpuCard(m.snap.CPU, m.cpuHist, w, h, false, &m.scrollBars[cardCPU]),
		diskCard(m.snap.Disk, m.snap.Proc.TopDisk, m.snap.Proc.DiskSupported, w, h, false, &m.scrollBars[cardDisk]),
		gpuCard(m.snap.GPU, m.gpuHist, w, h, false, &m.scrollBars[cardGPU]),
		memCard(m.snap.Mem, m.snap.Proc.TopMem, w, h, false, &m.scrollBars[cardMem]),
		netCardStr,
		procCardStr,
	}

	gap := strings.Repeat(" ", gapH)
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, cards[0], gap, cards[1], gap, cards[2])
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, cards[3], gap, cards[4], gap, cards[5])

	// Record the *actual* rendered card size so hitCard can map mouse
	// coordinates to the right card. Deriving this from a re-guessed geometry
	// drifts from what lipgloss actually produces (border/padding), which made
	// wheel events land on the wrong card.
	cardLines := strings.Split(cards[0], "\n")
	m.cardH = len(cardLines)
	if len(cardLines) > 0 {
		m.cardW = lipgloss.Width(cardLines[0])
	}

	// Row height from geometry already accounts for the footer and offset, so
	// no extra gap line is needed. Join rows directly; the resulting height
	// equals m.height - footerHeight - offsetRows and leaves that space free.
	grid := lipgloss.JoinVertical(lipgloss.Left, row1, row2)
	return strings.Split(grid, "\n")
}
