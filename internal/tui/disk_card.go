package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtop/internal/collector"
)

// diskCard renders one block per mount: a fill "tank", read/write rates and
// capacity, matching DISK.png.
func diskCard(d collector.DiskStat, innerWidth int, focused bool) string {
	header := pillStyle.Render(fmtSizeF(float64(d.UsedBytes)) + " / " + fmtSizeF(float64(d.TotalBytes)))

	if len(d.Mounts) == 0 {
		return renderCard(innerWidth, "▦", "磁盘", header, []string{faintStyle.Render("无磁盘数据")}, focused)
	}

	tw := maxInt(innerWidth-5, 8) // text block width beside the 3-wide tank
	half := tw / 2

	var lines []string
	for i, m := range d.Mounts {
		if i > 0 {
			lines = append(lines, "")
		}

		head := joinLR(
			dot(levelColor(m.UsedPercent))+" "+textStyle.Render(truncPlain(m.Mountpoint, tw-8)),
			labelStyle.Render("类型 ")+badgeStyle.Render(fstypeLabel(m.Fstype)),
			innerWidth)
		lines = append(lines, head)

		textLines := []string{
			twoCol(labelStyle.Render("读/s"), labelStyle.Render("写/s"), half, tw),
			twoCol(valueStyle.Render(fmtBytesF(m.ReadPerSec)), valueStyle.Render(fmtBytesF(m.WritePerSec)), half, tw),
			twoCol(labelStyle.Render("已用"), labelStyle.Render("可用"), half, tw),
			twoCol(valueStyle.Render(fmt.Sprintf("%.0f%%", m.UsedPercent)), valueStyle.Render(fmtSize(m.Free)), half, tw),
		}

		tank := strings.Join(vTank(2, m.UsedPercent), "\n")
		block := lipgloss.JoinHorizontal(lipgloss.Top, tank, "  ", strings.Join(textLines, "\n"))
		lines = append(lines, strings.Split(block, "\n")...)
	}

	return renderCard(innerWidth, "▦", "磁盘", header, lines, focused)
}

// twoCol lays two styled values into columns of width `left` and `total-left`.
func twoCol(a, b string, left, total int) string {
	return padRow(a, left) + b
}

func fstypeLabel(fs string) string {
	if fs == "" {
		return "unknown"
	}
	return fs
}
