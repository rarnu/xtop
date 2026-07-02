package collector

import (
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

const topProcCount = 20

// hiddenProcs lists process names that should be hidden from dashboard process
// lists because they are the collectors/tools used by xtop itself.
var hiddenProcs = map[string]bool{
	"nettop": true,
	"top":    true,
	"xtop":   true,
}

func shouldHideProc(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	if hiddenProcs[base] {
		return true
	}
	// Also match commands that are absolute paths ending with these names.
	if base == "" {
		return false
	}
	// strip a trailing .exe for Windows/Wine compatibility
	if strings.HasSuffix(base, ".exe") {
		base = strings.TrimSuffix(base, ".exe")
		return hiddenProcs[base]
	}
	return false
}

func filterProcList(list []ProcInfo) []ProcInfo {
	filtered := make([]ProcInfo, 0, len(list))
	for _, p := range list {
		if !shouldHideProc(p.Command) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// topN returns a copy of the first n entries of an already-sorted slice.
func topN(sorted []ProcInfo, n int) []ProcInfo {
	if len(sorted) < n {
		n = len(sorted)
	}
	return append([]ProcInfo(nil), sorted[:n]...)
}

// shortStatus reduces one process's status to a single display letter.
func shortStatus(p *process.Process) string {
	ss, err := p.Status()
	if err != nil || len(ss) == 0 {
		return "?"
	}
	return shortStatusCode(ss[0])
}

// shortStatusCode maps a raw status string to a single display letter. It
// accepts both gopsutil's letters/words and BSD `ps` codes (e.g. "Ss", "R+"),
// for which the leading character is the state.
func shortStatusCode(s string) string {
	if s == "" {
		return "?"
	}
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	switch s {
	case process.Running:
		return "R"
	case process.Sleep:
		return "S"
	case process.Idle:
		return "I"
	case process.Stop:
		return "T"
	case process.Zombie:
		return "Z"
	case process.Blocked:
		return "D"
	case process.Wait:
		return "W"
	case process.Lock:
		return "L"
	default:
		return strings.ToUpper(s[:1])
	}
}

// commandOf prefers the full command line, falling back to the process name.
func commandOf(p *process.Process) string {
	if cmd, err := p.Cmdline(); err == nil {
		if cmd = strings.TrimSpace(cmd); cmd != "" {
			return cmd
		}
	}
	if name, err := p.Name(); err == nil {
		return name
	}
	return "?"
}
