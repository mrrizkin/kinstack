package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valkey-io/valkey-go"
)

// lockoutRecord is the serialized failure state stored per account. The key is
// a hash of the username so the backing store never contains raw identities.
type lockoutRecord struct {
	Failures    int       `json:"failures"`
	LockedUntil time.Time `json:"locked_until"`
}

// Lockout throttles repeated failed password attempts for one account. It is
// intentionally best-effort: a storage outage must not make sign-in impossible,
// so callers treat read/write errors as "not locked".
type Lockout struct {
	storage fiber.Storage
	max     int
	window  time.Duration
	now     func() time.Time
}

// NewLockout builds a per-account lockout guard. A nil storage or non-positive
// maximum disables the guard.
func NewLockout(storage fiber.Storage, max int, window time.Duration) *Lockout {
	return &Lockout{storage: storage, max: max, window: window, now: time.Now}
}

func (l *Lockout) enabled() bool {
	return l != nil && l.storage != nil && l.max > 0 && l.window > 0
}

func (l *Lockout) key(username string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(username))))
	return "auth:lockout:{" + hex.EncodeToString(sum[:]) + "}"
}

func (l *Lockout) failureKey(username string) string {
	return l.key(username) + ":failures"
}

func (l *Lockout) lockedKey(username string) string {
	return l.key(username) + ":locked"
}

// valkeyFailureScript increments the failure counter and sets the lock key in
// one server-side operation. The hash-tagged keys share a slot in cluster mode.
const valkeyFailureScript = `
local failures = redis.call('INCR', KEYS[1])
if failures == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if failures >= tonumber(ARGV[2]) then
  redis.call('SET', KEYS[2], '1', 'PX', ARGV[1])
  redis.call('DEL', KEYS[1])
  return 1
end
return 0
`

var lockoutFailureLua = valkey.NewLuaScript(valkeyFailureScript)

// atomicClient returns the underlying Valkey client when the configured
// storage supports it. Other storages keep the best-effort read/modify/write
// path below.
func (l *Lockout) atomicClient() valkey.Client {
	if !l.enabled() {
		return nil
	}
	connector, ok := l.storage.(interface{ Conn() valkey.Client })
	if !ok {
		return nil
	}
	return connector.Conn()
}

func (l *Lockout) read(ctx context.Context, username string) lockoutRecord {
	if !l.enabled() {
		return lockoutRecord{}
	}
	raw, err := l.storage.GetWithContext(ctx, l.key(username))
	if err != nil || len(raw) == 0 {
		return lockoutRecord{}
	}
	var record lockoutRecord
	if json.Unmarshal(raw, &record) != nil {
		return lockoutRecord{}
	}
	if !record.LockedUntil.IsZero() && !record.LockedUntil.After(l.now()) {
		_ = l.storage.DeleteWithContext(ctx, l.key(username))
		return lockoutRecord{}
	}
	return record
}

// Allowed reports whether the account may attempt a password sign-in now.
func (l *Lockout) Allowed(ctx context.Context, username string) bool {
	if l.atomicClient() != nil {
		raw, err := l.storage.GetWithContext(ctx, l.lockedKey(username))
		return err != nil || len(raw) == 0
	}
	record := l.read(ctx, username)
	return record.LockedUntil.IsZero() || !record.LockedUntil.After(l.now())
}

// Failure records one failed credential attempt and locks the account after the
// configured threshold. Valkey uses a Lua script so concurrent attempts share
// one atomic counter; other stores use a best-effort read/modify/write.
func (l *Lockout) Failure(ctx context.Context, username string) {
	if !l.enabled() {
		return
	}
	if client := l.atomicClient(); client != nil {
		ttl := l.window.Milliseconds()
		if ttl < 1 {
			ttl = 1
		}
		_ = lockoutFailureLua.Exec(
			ctx,
			client,
			[]string{l.failureKey(username), l.lockedKey(username)},
			[]string{strconv.FormatInt(ttl, 10), strconv.Itoa(l.max)},
		)
		return
	}
	record := l.read(ctx, username)
	record.Failures++
	if record.Failures >= l.max {
		record.LockedUntil = l.now().Add(l.window)
		record.Failures = 0
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = l.storage.SetWithContext(ctx, l.key(username), payload, l.window)
}

// Success clears any accumulated failures after a valid sign-in.
func (l *Lockout) Success(ctx context.Context, username string) {
	if !l.enabled() {
		return
	}
	_ = l.storage.DeleteWithContext(ctx, l.failureKey(username))
	_ = l.storage.DeleteWithContext(ctx, l.lockedKey(username))
	_ = l.storage.DeleteWithContext(ctx, l.key(username))
}
