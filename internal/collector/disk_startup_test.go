package collector

import (
	"testing"
)

func TestDiskProcsImmediate(t *testing.T) {
	c := New()
	c.StartProcLoop()

	// First disk collection should include per-process disk data immediately,
	// even before the background proc loop has produced its second sample.
	d := c.CollectDisk()
	if len(d.TopDiskProcs) == 0 {
		t.Fatal("expected immediate disk process data on first CollectDisk")
	}
}
