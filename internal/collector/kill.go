package collector

import "github.com/shirou/gopsutil/v4/process"

// Terminate sends SIGTERM to the given pid (the "KILL" action).
func Terminate(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Terminate()
}

// ForceKill sends SIGKILL to the given pid (the "FORCE KILL" action).
func ForceKill(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
