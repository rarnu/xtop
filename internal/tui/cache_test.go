package tui

import (
	"os"
	"path/filepath"
	"testing"

	"xtop/internal/collector"
)

func TestCacheLoadSave(t *testing.T) {
	// Use a temporary home dir so we don't touch the real ~/.xtop.
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	snap := collector.Snapshot{
		Proc: collector.ProcStat{
			TopMem: []collector.ProcInfo{{PID: 1, Command: "memproc", MemRSS: 1 << 20}},
			TopDisk: []collector.ProcInfo{{PID: 2, Command: "diskproc", DiskReadPerSec: 100}},
		},
		Net: collector.NetStat{
			TopProcs: []collector.NetProc{{PID: 3, Command: "netproc", UploadPerSec: 50}},
		},
	}

	if err := saveCache(snap); err != nil {
		t.Fatalf("saveCache failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpHome, ".xtop", "cache.json")); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	m := New()
	if len(m.snap.Proc.TopMem) != 1 || m.snap.Proc.TopMem[0].PID != 1 {
		t.Fatalf("mem procs not loaded from cache: %v", m.snap.Proc.TopMem)
	}
	if len(m.snap.Net.TopProcs) != 1 || m.snap.Net.TopProcs[0].PID != 3 {
		t.Fatalf("net procs not loaded from cache: %v", m.snap.Net.TopProcs)
	}
	if len(m.snap.Proc.TopDisk) != 1 || m.snap.Proc.TopDisk[0].PID != 2 {
		t.Fatalf("disk procs not loaded from cache: %v", m.snap.Proc.TopDisk)
	}
}
