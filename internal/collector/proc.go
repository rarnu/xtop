package collector

import (
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

const topProcCount = 20

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
