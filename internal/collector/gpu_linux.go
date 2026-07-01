//go:build linux

package collector

import (
	"os/exec"
	"strconv"
	"strings"
)

// collectGPU queries NVIDIA GPUs via nvidia-smi. Any failure degrades to an
// "unavailable" stat rather than an error.
func collectGPU() GPUStat {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,power.draw,memory.used,memory.total,temperature.gpu,utilization.gpu",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return GPUStat{Available: false, Message: "未检测到 NVIDIA GPU (nvidia-smi 不可用)"}
	}

	var cards []GPUCard
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Split(line, ",")
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		if len(f) < 6 {
			continue
		}
		cards = append(cards, GPUCard{
			Name:     f[0],
			PowerW:   parseFloatNA(f[1]),
			MemUsed:  parseMiB(f[2]),
			MemTotal: parseMiB(f[3]),
			TempC:    parseFloatNA(f[4]),
			LoadPct:  parseFloatNA(f[5]),
		})
	}
	if len(cards) == 0 {
		return GPUStat{Available: false, Message: "未检测到 GPU"}
	}
	return GPUStat{Available: true, Cards: cards}
}

// parseFloatNA returns -1 (the N/A sentinel) when the value is not numeric,
// covering nvidia-smi's "[N/A]" placeholders.
func parseFloatNA(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return -1
	}
	return v
}

// parseMiB parses a mebibyte value and returns it in bytes (0 on error).
func parseMiB(s string) uint64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return 0
	}
	return uint64(v) * 1024 * 1024
}
