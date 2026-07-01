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

// hitCard maps absolute mouse coordinates to a card key and local body
// coordinates. It returns -1 if the point is outside any card body.
func (m *model) hitCard(x, y int) (cardKey, bool) {
	if m.width < 8 || m.height < 4 {
		return -1, false
	}
	if x < 0 || x >= m.width || y < 0 || y >= m.height {
		return -1, false
	}

	w, _ := m.dashboardGeometry()
	outerW := w + 4
	outerH := (m.height - footerHeight - 3) / 2 + 2 // matches dashboardGeometry

	col := x / (outerW + gapH)
	if col >= gridCols {
		return -1, false
	}
	row := y / outerH
	if row >= gridRows {
		return -1, false
	}

	key := cardKey(row*3 + col)
	if int(key) >= len(m.scrollBars) {
		return -1, false
	}
	return key, true
}
