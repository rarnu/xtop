package collector

// zeroTrafficThreshold is the bytes/sec below which a per-process network entry
// is considered effectively idle and hidden from the network card list.
const zeroTrafficThreshold = 0.1 // bytes per second

func filterNetProcList(list []NetProc) []NetProc {
	filtered := make([]NetProc, 0, len(list))
	for _, p := range list {
		if p.UploadPerSec >= zeroTrafficThreshold || p.DownloadPerSec >= zeroTrafficThreshold {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
