package database

import "testing"

func TestPoolSizeFallsBackAndHonoursOverrides(t *testing.T) {
	if got := poolSize(0, maxOpenConnections); got != maxOpenConnections {
		t.Fatalf("zero must keep the default, got %d", got)
	}
	if got := poolSize(-3, maxIdleConnections); got != maxIdleConnections {
		t.Fatalf("negative must keep the default, got %d", got)
	}
	if got := poolSize(42, maxOpenConnections); got != 42 {
		t.Fatalf("explicit value must win, got %d", got)
	}
}
