//go:build linux

package collector

import (
	"bufio"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// collectDisk gathers per-mount capacity/usage from `df -T` on Linux, filtering
// out tmpfs partitions. IO rates still come from gopsutil's IOCounters.
func (c *Collector) collectDisk(dt float64) DiskStat {
	// Per-device IO rates, diffed against the previous tick.
	rates := map[string]ioCounter{}
	curIO, ioErr := disk.IOCounters()
	if ioErr == nil {
		next := map[string]ioCounter{}
		for name, io := range curIO {
			cur := ioCounter{read: io.ReadBytes, write: io.WriteBytes}
			next[name] = cur
			if dt > 0 {
				if prev, ok := c.prevDiskIO[name]; ok {
					var r, w uint64
					if cur.read >= prev.read {
						r = cur.read - prev.read
					}
					if cur.write >= prev.write {
						w = cur.write - prev.write
					}
					rates[name] = ioCounter{read: r, write: w}
				}
			}
		}
		c.prevDiskIO = next
	}

	st := parseDfT(diskDfCmd(), rates, dt)
	sort.Slice(st.Mounts, func(i, j int) bool {
		return st.Mounts[i].Mountpoint < st.Mounts[j].Mountpoint
	})
	return st
}

func diskDfCmd() *exec.Cmd {
	return exec.Command("df", "-T")
}

func parseDfT(cmd *exec.Cmd, rates map[string]ioCounter, dt float64) DiskStat {
	out, err := cmd.Output()
	if err != nil {
		return DiskStat{}
	}

	var st DiskStat
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 7 {
			continue
		}
		// df -T header: Filesystem Type 1K-blocks Used Available Use% Mounted on
		fsType := fields[1]
		if fsType == "tmpfs" || fsType == "devtmpfs" {
			continue
		}

		sizeK, err1 := strconv.ParseUint(fields[2], 10, 64)
		usedK, err2 := strconv.ParseUint(fields[3], 10, 64)
		availK, err3 := strconv.ParseUint(fields[4], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || sizeK == 0 {
			continue
		}

		device := fields[0]
		mountpoint := strings.Join(fields[6:], " ")
		if seen[mountpoint] {
			continue
		}
		seen[mountpoint] = true

		const kb = 1024
		total := sizeK * kb
		used := usedK * kb
		free := availK * kb
		usedPct := float64(used) / float64(total) * 100

		m := DiskMount{
			Mountpoint:  mountpoint,
			Fstype:      fsType,
			Total:       total,
			Used:        used,
			Free:        free,
			UsedPercent: usedPct,
		}

		if dev := matchIODevice(device, rates); dev != "" {
			r := rates[dev]
			if dt > 0 {
				m.ReadPerSec = float64(r.read) / dt
				m.WritePerSec = float64(r.write) / dt
			}
		}

		st.Mounts = append(st.Mounts, m)
		st.TotalBytes += total
		st.UsedBytes += used
	}
	return st
}

// matchIODevice maps a partition device path (e.g. /dev/disk3s1s1 or /dev/sda1)
// to the best matching IO-counter key (e.g. disk3 / sda1 / sda).
func matchIODevice(device string, rates map[string]ioCounter) string {
	base := device
	base = strings.TrimPrefix(base, "/dev/")
	base = strings.TrimPrefix(base, "mapper/")

	if _, ok := rates[base]; ok {
		return base
	}
	best := ""
	for name := range rates {
		if strings.HasPrefix(base, name) && len(name) > len(best) {
			best = name
		}
	}
	return best
}
