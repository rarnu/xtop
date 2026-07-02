package tui

// cardScroll tracks per-card body scroll offset (line index).
type cardScroll struct {
	offset int
	max    int // total body lines - visible body lines
}

// cardScrollBars holds scroll offsets for all dashboard cards.
type cardScrollBars [6]cardScroll

// cardKey identifies a card for scroll state.
type cardKey int

const (
	cardCPU cardKey = iota
	cardDisk
	cardGPU
	cardMem
	cardNet
	cardProc
)

// hitCard maps absolute mouse coordinates to a card key using the actual
// rendered card size captured by renderDashboard. Returns false outside the grid.
func (m *model) hitCard(x, y int) (cardKey, bool) {
	if m.cardW <= 0 || m.cardH <= 0 {
		return -1, false
	}
	if x < 0 || y < 0 {
		return -1, false
	}

	col := x / (m.cardW + gapH)
	if col >= gridCols {
		return -1, false
	}
	row := y / m.cardH
	if row >= gridRows {
		return -1, false
	}

	key := cardKey(row*gridCols + col)
	if int(key) >= len(m.scrollBars) {
		return -1, false
	}
	return key, true
}
