//go:build !darwin

package collector

import "os/exec"

// detach is a no-op on non-Darwin platforms. Linux helpers do not steal the
// controlling terminal the way macOS nettop/top can, so setsid is unnecessary.
func detach(cmd *exec.Cmd) *exec.Cmd { return cmd }
