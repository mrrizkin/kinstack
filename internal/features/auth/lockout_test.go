package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/storage/memory/v2"
)

func TestLockoutLocksOneClientAndClears(t *testing.T) {
	guard := NewLockout(memory.New(), 2, time.Minute)
	ctx := t.Context()

	if !guard.Allowed(ctx, "alice", "10.0.0.1") {
		t.Fatal("fresh account should be allowed")
	}
	guard.Failure(ctx, "alice", "10.0.0.1", true)
	if !guard.Allowed(ctx, "alice", "10.0.0.1") {
		t.Fatal("one failure should not lock")
	}
	guard.Failure(ctx, "alice", "10.0.0.1", true)
	if guard.AllowedPair(ctx, "alice", "10.0.0.1") {
		t.Fatal("pair should be locked after threshold")
	}
	guard.Success(ctx, "alice", "10.0.0.1")
	if !guard.Allowed(ctx, "alice", "10.0.0.1") {
		t.Fatal("successful login should clear the pair lockout")
	}
}

// One client must not be able to lock a known account for everybody else.
func TestLockoutIsolatesClients(t *testing.T) {
	guard := NewLockout(memory.New(), 2, time.Minute)
	ctx := t.Context()

	for range 2 {
		guard.Failure(ctx, "alice", "10.0.0.1", true)
	}
	if guard.AllowedPair(ctx, "alice", "10.0.0.1") {
		t.Fatal("attacker client should be locked")
	}
	if !guard.Allowed(ctx, "alice", "10.0.0.2") {
		t.Fatal("a different client must still be allowed")
	}
	if guard.AccountLocked(ctx, "alice") {
		t.Fatal("the account should not be locked account-wide yet")
	}
}

func TestLockoutAccountWideThreshold(t *testing.T) {
	guard := NewLockout(memory.New(), 1, time.Minute)
	ctx := t.Context()

	for i := range guard.accountThreshold() {
		guard.Failure(ctx, "alice", fmt.Sprintf("10.0.0.%d", i), true)
	}
	if !guard.AccountLocked(ctx, "alice") {
		t.Fatal("expected the account-wide lock after the higher threshold")
	}
	if guard.Allowed(ctx, "alice", "10.0.0.200") {
		t.Fatal("account-wide lock must apply to new clients")
	}
}

func TestLockoutDisabledWithoutStorage(t *testing.T) {
	guard := NewLockout(nil, 1, time.Minute)
	ctx := t.Context()
	guard.Failure(ctx, "alice", "10.0.0.1", true)
	if !guard.Allowed(ctx, "alice", "10.0.0.1") {
		t.Fatal("nil storage guard must fail open")
	}
}
