package tui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"xtop/internal/collector"
)

const cacheFileName = "cache.json"

// procCache is the on-disk format for the xtop process cache. It stores the
// process lists shown in the memory, network and disk cards so the UI can render
// something immediately on startup instead of waiting for the first background
// collection.
type procCache struct {
	SavedAt   time.Time           `json:"saved_at"`
	MemProcs  []collector.ProcInfo `json:"mem_procs"`
	NetProcs  []collector.NetProc  `json:"net_procs"`
	DiskProcs []collector.ProcInfo `json:"disk_procs"`
}

// cacheDir returns the xtop cache directory (~/.xtop).
func cacheDir() string {
	return xtopHome()
}

// cachePath returns the full path to the cache file.
func cachePath() string {
	return filepath.Join(cacheDir(), cacheFileName)
}

// loadCache reads the saved process cache from ~/.xtop/cache.json, if it exists.
func loadCache() (procCache, error) {
	pc := procCache{}
	path := cachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return pc, nil
		}
		return pc, err
	}
	if err := json.Unmarshal(data, &pc); err != nil {
		return pc, err
	}
	return pc, nil
}

// saveCache writes the current process lists to ~/.xtop/cache.json.
func saveCache(s collector.Snapshot) error {
	dir := cacheDir()
	if dir == "" {
		return errors.New("cannot determine home directory")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	pc := procCache{
		SavedAt:   time.Now(),
		MemProcs:  collector.CloneProcInfos(s.Proc.TopMem),
		NetProcs:  collector.CloneNetProcs(s.Net.TopProcs),
		DiskProcs: collector.CloneProcInfos(s.Proc.TopDisk),
	}
	data, err := json.MarshalIndent(pc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(), data, 0644)
}
