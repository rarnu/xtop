//go:build darwin

package collector

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	reGPUUtil = regexp.MustCompile(`"Device Utilization %"=(\d+)`)
	reGPUMem  = regexp.MustCompile(`"In use system memory"=(\d+)`)
)

// collectGPU reads the Apple Silicon integrated GPU via ioreg (no sudo needed).
// Only utilisation (and, when present, in-use memory) is available this way;
// power/temperature are reported as N/A.
func collectGPU() GPUStat {
	out, err := detach(exec.Command("ioreg", "-r", "-d", "1", "-w", "0", "-c", "AGXAccelerator")).Output()
	if err != nil {
		return GPUStat{Available: false, Message: "无 GPU 数据 (ioreg 不可用)"}
	}
	text := string(out)

	m := reGPUUtil.FindStringSubmatch(text)
	if len(m) != 2 {
		return GPUStat{Available: false, Message: "无 GPU 数据"}
	}
	load, _ := strconv.ParseFloat(m[1], 64)

	card := GPUCard{
		Name:     gpuName(),
		PowerW:   -1,
		TempC:    -1,
		LoadPct:  load,
		MemTotal: 0,
	}
	if mm := reGPUMem.FindStringSubmatch(text); len(mm) == 2 {
		if v, err := strconv.ParseUint(mm[1], 10, 64); err == nil {
			card.MemUsed = v
		}
	}

	return GPUStat{Available: true, Cards: []GPUCard{card}}
}

// gpuName derives a friendly name from the chip brand (e.g. "Apple M5 Max").
func gpuName() string {
	out, err := detach(exec.Command("sysctl", "-n", "machdep.cpu.brand_string")).Output()
	if err != nil {
		return "Apple GPU"
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "Apple GPU"
	}
	return name + " GPU"
}
