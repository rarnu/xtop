//go:build !linux

package collector

import (
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// collectDisk gathers per-mount capacity/usage plus per-device read/write rates.
func (c *Collector) collectDisk(dt float64) DiskStat {
	parts, err := disk.Partitions(false)
	if err != nil {
		return DiskStat{}
	}

	// Per-device IO rates, diffed against the previous tick.
	rates := map[string]ioCounter{} // device -> bytes/sec (stored in the fields)
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

	seen := map[string]bool{} // dedupe mountpoints
	var st DiskStat
	for _, p := range parts {
		if seen[p.Mountpoint] || skipMount(p) {
			continue
		}
		seen[p.Mountpoint] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}

		m := DiskMount{
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		}

		if dev := matchIODevice(p.Device, rates); dev != "" {
			r := rates[dev]
			if dt > 0 {
				m.ReadPerSec = float64(r.read) / dt
				m.WritePerSec = float64(r.write) / dt
			}
		}

		st.Mounts = append(st.Mounts, m)
		st.TotalBytes += usage.Total
		st.UsedBytes += usage.Used
	}

	sort.Slice(st.Mounts, func(i, j int) bool {
		return st.Mounts[i].Mountpoint < st.Mounts[j].Mountpoint
	})
	return st
}

// skipMount filters out pseudo filesystems and macOS system/hidden volumes.
// The "nobrowse" option marks Finder-hidden system volumes on darwin (VM,
// Preboot, Update, simulator images, …) and does not exist on Linux, so this
// filter is effectively macOS-only noise reduction and safe cross-platform.
func skipMount(p disk.PartitionStat) bool {
	switch p.Fstype {
	case "devfs", "autofs", "devicefs", "none", "":
		return true
	}
	for _, o := range p.Opts {
		if o == "nobrowse" {
			return true
		}
	}
	return false
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
	// Longest IO key that is a prefix of the device base name wins
	// (handles disk3s1s1 -> disk3, sda1 -> sda).
	best := ""
	for name := range rates {
		if strings.HasPrefix(base, name) && len(name) > len(best) {
			best = name
		}
	}
	return best
}
