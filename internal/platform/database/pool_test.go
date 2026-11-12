package database

import "testing"

func TestPoolSizeFallsBackAndHonoursOverrides(t *testing.T) {
	if got := poolSize(0, openConnectionsMax); got != openConnectionsMax {
		t.Fatalf("zero must keep the default, got %d", got)
	}
	if got := poolSize(-3, idleConnectionsMax); got != idleConnectionsMax {
		t.Fatalf("negative must keep the default, got %d", got)
	}
	if got := poolSize(42, openConnectionsMax); got != 42 {
		t.Fatalf("explicit value must win, got %d", got)
	}
}
