package tui

import (
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ---- style cache -------------------------------------------------------

var (
	fgStyleMu   sync.RWMutex
	fgStyleCache = map[lipgloss.Color]lipgloss.Style{}
)

// ---- render caches ------------------------------------------------------

// barKey uniquely identifies a cached one-dimensional bar/gauge render.
type barKey struct {
	width int
	pct   int // percentage rounded to nearest integer
	kind  string
}

var (
	barCacheMu sync.RWMutex
	barCache   = map[barKey]string{}
)

func cachedBar(width int, pct float64, kind string, render func(int, float64) string) string {
	if width < 1 {
		width = 1
	}
	k := barKey{width: width, pct: int(math.Round(pct)), kind: kind}
	barCacheMu.RLock()
	v, ok := barCache[k]
	barCacheMu.RUnlock()
	if ok {
		return v
	}
	v = render(width, pct)
	barCacheMu.Lock()
	barCache[k] = v
	barCacheMu.Unlock()
	return v
}

// sparkKey caches a sparkline render. pctDigest stores the auto-scaled max as
// an integer percentage of the absolute max sample; this is approximate but
// enough to detect meaningful changes in the sparkline shape.
type sparkKey struct {
	width   int
	color   lipgloss.Color
	last8   uint64 // digest of the last up to 8 samples
	maxPct  int
}

func digestFloats(v []float64) uint64 {
	// Fast, order-sensitive hash of float values quantized to integers.
	var h uint64
	for _, f := range v {
		h = h*31 + uint64(int64(f*1000+0.5))
	}
	return h
}

var (
	sparkCacheMu sync.RWMutex
	sparkCache   = map[sparkKey]string{}
)

func cachedSparkline(width int, data []float64, color lipgloss.Color, render func(int, []float64, lipgloss.Color) string) string {
	if width < 1 {
		width = 1
	}
	max := 0.0
	for _, v := range data {
		if v > max {
			max = v
		}
	}
	maxPct := 0
	if max > 0 {
		maxPct = int(max*100 + 0.5)
	}
	keyData := data
	if len(keyData) > width {
		keyData = keyData[len(keyData)-width:]
	}
	if len(keyData) > 8 {
		keyData = keyData[len(keyData)-8:]
	}
	k := sparkKey{
		width:  width,
		color:  color,
		last8:  digestFloats(keyData),
		maxPct: maxPct,
	}
	sparkCacheMu.RLock()
	v, ok := sparkCache[k]
	sparkCacheMu.RUnlock()
	if ok {
		return v
	}
	v = render(width, data, color)
	sparkCacheMu.Lock()
	sparkCache[k] = v
	sparkCacheMu.Unlock()
	return v
}

func init() {
	// Warm caches for common bar widths and percentages on startup to avoid
	// first-frame allocation spikes.
	for pct := 0; pct <= 100; pct += 10 {
		for w := 1; w <= 40; w++ {
			_ = cachedBar(w, float64(pct), "meter", renderMeterBar)
			_ = cachedBar(w, float64(pct), "block", renderBlockBar)
			_ = cachedBar(w, float64(pct), "line", renderLineGauge)
		}
	}
}

// fgStyle returns a cached lipgloss.Style with the requested foreground colour.
// This avoids allocating a new style on every render frame in hot paths.
func fgStyle(c lipgloss.Color) lipgloss.Style {
	fgStyleMu.RLock()
	s, ok := fgStyleCache[c]
	fgStyleMu.RUnlock()
	if ok {
		return s
	}
	fgStyleMu.Lock()
	defer fgStyleMu.Unlock()
	if s, ok := fgStyleCache[c]; ok {
		return s
	}
	s = lipgloss.NewStyle().Foreground(c)
	fgStyleCache[c] = s
	return s
}

// ---- number formatting -------------------------------------------------

var sizeUnits = []string{"B", "K", "M", "G", "T", "P"}
var rateUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// fmtSize renders a byte count with a single-letter unit, e.g. "6.8 T".
func fmtSize(v uint64) string { return scale(float64(v), sizeUnits, "") }

// fmtSizeF is fmtSize for a float input (used for aggregate sums).
func fmtSizeF(v float64) string { return scale(v, sizeUnits, "") }

// fmtBytesF renders a byte value with a two-letter unit and no rate suffix,
// e.g. "0.0 B", "2.1 MB" (the caller's label carries any "/s").
func fmtBytesF(v float64) string { return scale(v, rateUnits, "") }

// fmtRate renders a throughput, e.g. "2.1 MB/s".
func fmtRate(v float64) string { return scale(v, rateUnits, "/s") }

// scaleCache caches formatted scale outputs keyed by (value*100, unit index,
// suffix). Most displayed byte/rate values fall into a small range, so this
// removes a large fraction of fmt.Sprintf calls from the render hot path.
var (
	scaleCacheMu sync.RWMutex
	scaleCache   = map[[3]string]string{}
)

func cachedScale(v float64, unit, suffix string) string {
	key := [3]string{strconv.FormatInt(int64(v*100+0.5), 10), unit, suffix}
	scaleCacheMu.RLock()
	out, ok := scaleCache[key]
	scaleCacheMu.RUnlock()
	if ok {
		return out
	}
	out = formatFloat1(v) + " " + unit + suffix
	scaleCacheMu.Lock()
	scaleCache[key] = out
	scaleCacheMu.Unlock()
	return out
}

func scale(v float64, units []string, suffix string) string {
	if v < 0 {
		v = 0
	}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return cachedScale(v, units[i], suffix)
}



// rateW returns the display cell width of fmtRate(v). Used when building
// miniRows so the column width can be cached without re-measuring each frame.
func rateW(v float64) int {
	return runewidth.StringWidth(fmtRate(v))
}

// ---- bars & meters -----------------------------------------------------

func clampPct(p float64) float64 {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// meterBar renders a CPU-style meter of vertical ticks: filled ticks coloured by
// utilisation level, the remainder in a dark track. (see CPU.png)
func meterBar(width int, pct float64) string {
	return cachedBar(width, pct, "meter", renderMeterBar)
}

func renderMeterBar(width int, pct float64) string {
	if width < 1 {
		width = 1
	}
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled > width {
		filled = width
	}
	on := fgStyle(levelColor(pct)).Render(strings.Repeat("│", filled))
	off := fgStyle(colTrack).Render(strings.Repeat("│", width-filled))
	return on + off
}

// blockBar renders a solid horizontal bar ('█') filled to pct, coloured by level,
// with a dark track behind. (disk usage, gpu memory)
func blockBar(width int, pct float64) string {
	return cachedBar(width, pct, "block", renderBlockBar)
}

func renderBlockBar(width int, pct float64) string {
	if width < 1 {
		width = 1
	}
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled > width {
		filled = width
	}
	on := fgStyle(levelColor(pct)).Render(strings.Repeat("█", filled))
	off := fgStyle(colTrack).Render(strings.Repeat("─", width-filled))
	return on + off
}

// lineGauge renders a thin rounded gauge used by GPU temperature/load. (GPU.png)
func lineGauge(width int, pct float64) string {
	return cachedBar(width, pct, "line", renderLineGauge)
}

func renderLineGauge(width int, pct float64) string {
	if width < 2 {
		width = 2
	}
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled > width {
		filled = width
	}
	on := fgStyle(levelColor(pct)).Render(strings.Repeat("━", filled))
	off := fgStyle(colGray).Render(strings.Repeat("━", width-filled))
	return on + off
}

// segPart is one coloured slice of a segmented bar.
type segPart struct {
	frac  float64 // fraction of the whole (0-1)
	color lipgloss.Color
}

// segBar renders proportional coloured segments across width cells. Used for the
// memory used/cached/free composition. (MEM.png)
func segBar(width int, parts []segPart) string {
	if width < 1 {
		width = 1
	}
	var b strings.Builder
	used := 0
	for i, p := range parts {
		n := int(math.Round(p.frac * float64(width)))
		if i == len(parts)-1 {
			n = width - used // last segment fills remainder to avoid rounding gaps
		}
		if n < 0 {
			n = 0
		}
		if used+n > width {
			n = width - used
		}
		if n > 0 {
			b.WriteString(fgStyle(p.color).Render(strings.Repeat("█", n)))
			used += n
		}
	}
	return b.String()
}

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

// sparkline renders the last `width` samples as a coloured block sparkline,
// auto-scaled to the maximum sample. (chart strip at the top of cards)
func sparkline(width int, data []float64, color lipgloss.Color) string {
	return cachedSparkline(width, data, color, renderSparkline)
}

func renderSparkline(width int, data []float64, color lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	if len(data) == 0 {
		return fgStyle(colTrack).Render(strings.Repeat("▁", width))
	}
	if len(data) > width {
		data = data[len(data)-width:]
	}
	max := 0.0
	for _, v := range data {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		max = 1
	}
	var b strings.Builder
	// left-pad with empty track when fewer samples than width
	if pad := width - len(data); pad > 0 {
		b.WriteString(fgStyle(colTrack).Render(strings.Repeat("▁", pad)))
	}
	style := fgStyle(color)
	for _, v := range data {
		idx := int(v / max * float64(len(sparkRunes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkRunes) {
			idx = len(sparkRunes) - 1
		}
		b.WriteString(style.Render(string(sparkRunes[idx])))
	}
	return b.String()
}

var tankFrame = lipgloss.NewStyle().Foreground(colGreenDim)

// vTank renders a small vertical "tank" filled from the bottom to pct, returning
// innerRows+2 lines (rounded frame included). (disk cards in DISK.png)
func vTank(innerRows int, pct float64) []string {
	pct = clampPct(pct)
	filled := int(math.Round(pct / 100 * float64(innerRows)))
	col := fgStyle(levelColor(pct))

	lines := make([]string, 0, innerRows+2)
	lines = append(lines, tankFrame.Render("╭─╮"))
	for r := 0; r < innerRows; r++ {
		rowFromBottom := innerRows - 1 - r
		cell := " "
		if rowFromBottom < filled {
			cell = col.Render("█")
		}
		lines = append(lines, tankFrame.Render("│")+cell+tankFrame.Render("│"))
	}
	lines = append(lines, tankFrame.Render("╰─╯"))
	return lines
}

// dot returns a coloured bullet used in legends.
func dot(color lipgloss.Color) string {
	return fgStyle(color).Render("●")
}
