package tui

// scrollbarMargin is the number of columns reserved on the right side of a
// card only when a scrollbar is actually shown. It includes padding plus the
// scrollbar itself.
const scrollbarMargin = 2

// contentWidth returns the usable body width. When a scrollbar may be visible
// the card content should be laid out against this reduced width so text does
// not collide with the scrollbar. When no scrollbar is shown, renderCard fills
// the full innerWidth.
func contentWidth(innerWidth int) int {
	w := innerWidth - scrollbarMargin
	if w < 1 {
		w = 1
	}
	return w
}
