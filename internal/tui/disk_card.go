package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtop/internal/collector"
)

// diskCard renders one block per mount: a fill "tank", read/write rates and
// capacity, matching DISK.png. Below the mounts it appends the top processes by
// disk read+write rate (feature 3); the whole card scrolls when it overflows.
func diskCard(d collector.DiskStat, procs []collector.ProcInfo, diskSupported bool, innerWidth, innerHeight int, focused bool, scroll *cardScroll, selected selectedProc) string {
	header := pillStyle.Render(fmtSizeF(float64(d.UsedBytes)) + " / " + fmtSizeF(float64(d.TotalBytes)))
	cw := contentWidth(innerWidth)

	if len(d.Mounts) == 0 {
		return renderCard(innerWidth, innerHeight, "▦", T("card.disk"), header, []string{faintStyle.Render(T("no_data.disk"))}, focused, scroll)
	}

	tw := maxInt(cw-5, 8) // text block width beside the 3-wide tank
	half := tw / 2

	var lines []string
	for i, m := range d.Mounts {
		if i > 0 {
			lines = append(lines, "")
		}

		head := joinLR(
			dot(levelColor(m.UsedPercent))+" "+textStyle.Render(truncPlain(m.Mountpoint, tw-8)),
			labelStyle.Render(T("label.type"))+" "+badgeStyle.Render(fstypeLabel(m.Fstype)),
			cw)
		lines = append(lines, head)

		textLines := []string{
			twoCol(
				labelStyle.Render(T("label.read"))+"\n"+valueStyle.Render(fmtBytesF(m.ReadPerSec)),
				labelStyle.Render(T("label.used"))+"\n"+valueStyle.Render(fmtSize(m.Used)+"("+formatFloat0(m.UsedPercent)+"%)"),
				half, tw,
			),
			twoCol(
				labelStyle.Render(T("label.write"))+"\n"+valueStyle.Render(fmtBytesF(m.WritePerSec)),
				labelStyle.Render(T("label.free"))+"\n"+valueStyle.Render(fmtSize(m.Free)+" / "+fmtSize(m.Total)),
				half, tw,
			),
		}

		tank := strings.Join(vTank(2, m.UsedPercent), "\n")
		block := lipgloss.JoinHorizontal(lipgloss.Top, tank, "  ", strings.Join(textLines, "\n"))
		lines = append(lines, strings.Split(block, "\n")...)
	}

	// Process list (top by disk read+write rate) appended below the mounts.
	lines = append(lines, "")
	if !diskSupported {
		lines = append(lines, miniTwoColHeaderLine(cw, nil, T("label.read"), T("label.write")))
		lines = append(lines, faintStyle.Render(T("no_data.net_procs")))
	} else {
		rows := make([]miniRow, 0, len(procs))
		for _, p := range procs {
			if p.DiskReadPerSec == 0 && p.DiskWritePerSec == 0 {
				continue
			}
			rows = append(rows, miniRow{
				UpValue:    fmtRate(p.DiskReadPerSec),
				DownValue:  fmtRate(p.DiskWritePerSec),
				Command:    p.Command,
				PID:        p.PID,
				TwoCol:     true,
				upValueW:   rateW(p.DiskReadPerSec),
				downValueW: rateW(p.DiskWritePerSec),
			})
		}
		lines = append(lines, miniTwoColHeaderLine(cw, rows, T("label.read"), T("label.write")))
		lines = append(lines, miniRowLines(cw, rows, selected)...)
	}

	return renderCard(innerWidth, innerHeight, "▦", T("card.disk"), header, lines, focused, scroll)
}

// twoCol lays two styled blocks side by side. Each block may contain multiple
// lines; the left block is padded to `left` cells and the pair is padded to
// `total` cells.
func twoCol(a, b string, left, total int) string {
	la := strings.Split(a, "\n")
	lb := strings.Split(b, "\n")
	maxLines := maxInt(len(la), len(lb))
	for len(la) < maxLines {
		la = append(la, "")
	}
	for len(lb) < maxLines {
		lb = append(lb, "")
	}
	var out []string
	for i := 0; i < maxLines; i++ {
		out = append(out, padRow(la[i], left)+lb[i])
	}
	block := strings.Join(out, "\n")
	return padRow(block, total)
}

func fstypeLabel(fs string) string {
	if fs == "" {
		return T("unknown")
	}
	return fs
}
