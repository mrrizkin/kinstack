package auth

import (
	"testing"
	"time"

	"github.com/gofiber/storage/memory/v2"
)

func TestLockoutLocksAndClears(t *testing.T) {
	guard := NewLockout(memory.New(), 2, time.Minute)
	ctx := t.Context()

	if !guard.Allowed(ctx, "alice") {
		t.Fatal("fresh account should be allowed")
	}
	guard.Failure(ctx, "alice")
	if !guard.Allowed(ctx, "alice") {
		t.Fatal("one failure should not lock")
	}
	guard.Failure(ctx, "alice")
	if guard.Allowed(ctx, "alice") {
		t.Fatal("account should be locked after threshold")
	}
	guard.Success(ctx, "alice")
	if !guard.Allowed(ctx, "alice") {
		t.Fatal("successful login should clear lockout")
	}
}

func TestLockoutDisabledWithoutStorage(t *testing.T) {
	guard := NewLockout(nil, 1, time.Minute)
	ctx := t.Context()
	guard.Failure(ctx, "alice")
	if !guard.Allowed(ctx, "alice") {
		t.Fatal("nil storage guard must fail open")
	}
}
