package tui

import "xtop/internal/collector"

// netCard renders upload/download speed and cumulative traffic with a download
// sparkline, matching NETWORK.png.
func netCard(n collector.NetStat, downHist []float64, innerWidth, innerHeight int, focused bool, scroll *cardScroll) string {
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

	lines := []string{head, "", up, down}
	return renderCard(innerWidth, innerHeight, "◍", "网络", header, lines, focused, scroll)
}
