//go:build darwin

package collector

import (
	"os/exec"
	"syscall"
)

// detach makes cmd run in its own session, so it has no controlling terminal.
//
// This is important for a TUI: helpers like nettop/ps/ioreg would otherwise
// share the program's controlling tty and can open /dev/tty directly. A child
// that touches the tty can steal keystrokes from the Bubble Tea input reader,
// which shows up as lost/laggy input (e.g. having to press q several times to
// quit) — especially for nettop, which we re-run every few seconds.
func detach(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd
}
