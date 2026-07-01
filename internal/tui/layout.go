package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const colGap = 1 // columns between dashboard cards

// dashboardColumns picks a column count and per-card inner width for the width.
func (m *model) dashboardGeometry() (numCols, inner int) {
	numCols = clampInt(m.width/46, 1, 3)
	outer := (m.width - (numCols-1)*colGap) / numCols
	inner = outer - 4 // border (2) + horizontal padding (2)
	if inner < 20 {
		inner = 20
	}
	return numCols, inner
}

// renderDashboard renders the six cards and packs them into a balanced
// multi-column grid, returning the content as individual lines.
func (m *model) renderDashboard() []string {
	if m.width < 8 {
		return []string{""}
	}
	numCols, inner := m.dashboardGeometry()

	cards := []string{
		cpuCard(m.snap.CPU, m.cpuHist, inner, false),
		memCard(m.snap.Mem, inner, false),
		diskCard(m.snap.Disk, inner, false),
		netCard(m.snap.Net, m.downHist, inner, false),
		gpuCard(m.snap.GPU, m.gpuHist, inner, false),
		procCard(m.snap.Proc, inner, false),
	}

	// Masonry: drop each card into the currently shortest column for balance.
	colHeights := make([]int, numCols)
	colCards := make([][]string, numCols)
	for _, c := range cards {
		shortest := 0
		for i := 1; i < numCols; i++ {
			if colHeights[i] < colHeights[shortest] {
				shortest = i
			}
		}
		colCards[shortest] = append(colCards[shortest], c)
		colHeights[shortest] += lipgloss.Height(c) + 1 // +1 for the vertical gap
	}

	columns := make([]string, 0, numCols*2-1)
	for i := 0; i < numCols; i++ {
		if i > 0 {
			columns = append(columns, strings.Repeat(" ", colGap)) // gap column
		}
		columns = append(columns, strings.Join(colCards[i], "\n\n"))
	}

	grid := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	return strings.Split(grid, "\n")
}
