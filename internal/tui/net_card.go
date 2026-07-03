package tui

import "xtop/internal/collector"

// netCard renders upload/download speed and cumulative traffic (fixed), plus a
// scrollable list of the top processes by network throughput below it. The
// summary stays pinned; only the process list scrolls (feature 2).
func netCard(n collector.NetStat, downHist []float64, innerWidth, innerHeight int, focused bool, scroll *cardScroll, selected selectedProc) string {
	cw := contentWidth(innerWidth)
	sparkW := clampInt(cw/2, 8, maxInt(cw-8, 8))
	header := sparkline(sparkW, downHist, colBlue)

	labelW := clampInt(12, 6, cw/2)
	rest := cw - labelW
	speedW := rest / 2
	totalW := rest - speedW

	head := padRow("", labelW) +
		labelStyle.Render(fitCell("速度", speedW, false)) +
		labelStyle.Render(fitCell("已用流量", totalW, false))

	up := padRow(dot(colGreen)+" "+textStyle.Render("上传"), labelW) +
		valueStyle.Render(fitCell(fmtRate(n.UploadPerSec), speedW, false)) +
		valueStyle.Render(fitCell(fmtSizeF(float64(n.TotalUpload)), totalW, false))

	down := padRow(dot(colBlue)+" "+textStyle.Render("下载"), labelW) +
		valueStyle.Render(fitCell(fmtRate(n.DownloadPerSec), speedW, false)) +
		valueStyle.Render(fitCell(fmtSizeF(float64(n.TotalDownload)), totalW, false))

	var fixed, list []string
	if n.ProcsSupported && len(n.TopProcs) == 0 {
		fixed = []string{head, "", up, down, miniTwoColHeaderLine(cw, nil, "上传", "下载")}
		list = []string{faintStyle.Render("无进程数据")}
	} else if !n.ProcsSupported {
		fixed = []string{head, "", up, down, miniTwoColHeaderLine(cw, nil, "上传", "下载")}
		list = []string{faintStyle.Render("无进程数据")}
	} else {
		rows := make([]miniRow, 0, len(n.TopProcs))
		for _, p := range n.TopProcs {
			rows = append(rows, miniRow{
				UpValue:   fmtRate(p.UploadPerSec),
				DownValue: fmtRate(p.DownloadPerSec),
				Command:   p.Command,
				PID:       p.PID,
				TwoCol:    true,
			})
		}
		fixed = []string{head, "", up, down, miniTwoColHeaderLine(cw, rows, "上传", "下载")}
		list = miniRowLines(cw, rows, selected)
	}
	return renderCardSplit(innerWidth, innerHeight, "◍", "网络", header, fixed, list, focused, scroll)
}
