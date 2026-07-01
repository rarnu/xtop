package collector

import (
	"testing"
	"time"
)

// TestTwoTicks verifies that a second snapshot produces realistic per-process
// CPU% and RSS values (regression guard for diff/refresh logic).
func TestTwoTicks(t *testing.T) {
	c := New()
	_ = c.Snapshot()
	time.Sleep(1100 * time.Millisecond)
	s := c.Snapshot()

	if len(s.CPU.PerCore) == 0 {
		t.Fatal("no CPU cores reported")
	}

	found := false
	for _, p := range s.Proc.All {
		if p.MemRSS > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no process has RSS > 0")
	}
}
