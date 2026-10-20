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

// accountWideLockoutFactor is how many times more failures an account needs
// before it is locked for every client. The per-(account, client) lock trips
// first, so a single attacker cannot lock a known account on its own.
const accountWideLockoutFactor = 10

// lockoutRecord is the serialized failure state stored per key. Keys are
// hashes so the backing store never contains raw identities.
type lockoutRecord struct {
	Failures    int       `json:"failures"`
	LockedUntil time.Time `json:"locked_until"`
}

// Lockout throttles repeated failed password attempts. Failures are counted per
// (account, client) pair, with a much higher account-wide threshold so a
// distributed attack is still slowed down. It is intentionally best-effort: a
// storage outage must not make sign-in impossible, so read/write errors are
// treated as "not locked".
type Lockout struct {
	storage fiber.Storage
	max     int
	window  time.Duration
	now     func() time.Time
}

// NewLockout builds a lockout guard. A nil storage or non-positive maximum
// disables the guard.
func NewLockout(storage fiber.Storage, max int, window time.Duration) *Lockout {
	return &Lockout{storage: storage, max: max, window: window, now: time.Now}
}

func (l *Lockout) enabled() bool {
	return l != nil && l.storage != nil && l.max > 0 && l.window > 0
}

func (l *Lockout) accountThreshold() int {
	return l.max * accountWideLockoutFactor
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(sum[:])
}

// keys returns the account-wide and per-client key sets for one attempt. The
// client is an IP address (or another coarse identity) so sharing an account
// across devices does not lock the owner out of every device at once.
func (l *Lockout) keys(username, client string) (pair, account lockoutKeys) {
	accountID := hashKey(username)
	clientID := hashKey(client)
	if strings.TrimSpace(client) == "" {
		clientID = "unknown"
	}
	return pairKeys("auth:lockout:{" + accountID + "}:" + clientID),
		accountKeys("auth:lockout:{" + accountID + "}:all")
}

type lockoutKeys struct {
	failures string
	locked   string
}

func pairKeys(prefix string) lockoutKeys {
	return lockoutKeys{failures: prefix + ":failures", locked: prefix + ":locked"}
}

func accountKeys(prefix string) lockoutKeys {
	return pairKeys(prefix)
}

// valkeyFailureScript increments the failure counter and sets the lock key in
// one server-side operation. The hash-tagged keys share a slot in cluster mode.
// ARGV: ttl milliseconds, per-pair maximum, account maximum, account-wide flag.
const valkeyFailureScript = `
local function bump(failure_key, lock_key, limit)
  local failures = redis.call('INCR', failure_key)
  if failures == 1 then
    redis.call('PEXPIRE', failure_key, ARGV[1])
  end
  if failures >= tonumber(limit) then
    redis.call('SET', lock_key, '1', 'PX', ARGV[1])
    redis.call('DEL', failure_key)
    return 1
  end
  return 0
end

local pair = bump(KEYS[1], KEYS[2], ARGV[2])
if ARGV[4] == '1' then
  bump(KEYS[3], KEYS[4], ARGV[3])
end
return pair
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

func (l *Lockout) readRecord(ctx context.Context, key string) lockoutRecord {
	raw, err := l.storage.GetWithContext(ctx, key)
	if err != nil || len(raw) == 0 {
		return lockoutRecord{}
	}
	var record lockoutRecord
	if json.Unmarshal(raw, &record) != nil {
		return lockoutRecord{}
	}
	return record
}

func (l *Lockout) locked(ctx context.Context, key string) bool {
	raw, err := l.storage.GetWithContext(ctx, key)
	return err == nil && len(raw) > 0
}

// Allowed reports whether this account may attempt a password sign-in from the
// given client. Both the pair and the account-wide key must be clear.
func (l *Lockout) Allowed(ctx context.Context, username, client string) bool {
	return l.AllowedPair(ctx, username, client) && !l.AccountLocked(ctx, username)
}

// AllowedPair reports whether the (account, client) pair is still allowed. It
// deliberately ignores the account-wide lock so the break-glass account cannot
// be locked globally by a remote attacker.
func (l *Lockout) AllowedPair(ctx context.Context, username, client string) bool {
	if !l.enabled() {
		return true
	}
	pair, _ := l.keys(username, client)
	return !l.locked(ctx, pair.locked)
}

// AccountLocked reports whether the account is locked for every client.
func (l *Lockout) AccountLocked(ctx context.Context, username string) bool {
	if !l.enabled() {
		return false
	}
	_, account := l.keys(username, "")
	return l.locked(ctx, account.locked)
}

// Failure records one failed credential attempt. accountWide enables the
// higher account-wide counter; it is disabled for the break-glass account.
func (l *Lockout) Failure(ctx context.Context, username, client string, accountWide bool) {
	if !l.enabled() {
		return
	}
	pair, account := l.keys(username, client)
	if l.atomicClient() != nil {
		ttl := l.window.Milliseconds()
		if ttl < 1 {
			ttl = 1
		}
		wide := "0"
		if accountWide {
			wide = "1"
		}
		_ = lockoutFailureLua.Exec(
			ctx,
			l.atomicClient(),
			[]string{pair.failures, pair.locked, account.failures, account.locked},
			[]string{
				strconv.FormatInt(ttl, 10),
				strconv.Itoa(l.max),
				strconv.Itoa(l.accountThreshold()),
				wide,
			},
		)
		return
	}
	l.bump(ctx, pair, l.max)
	if accountWide {
		l.bump(ctx, account, l.accountThreshold())
	}
}

// bump increments one best-effort counter (storage without atomic scripting).
func (l *Lockout) bump(ctx context.Context, keys lockoutKeys, limit int) {
	record := l.readRecord(ctx, keys.failures)
	record.Failures++
	if record.Failures >= limit {
		record.LockedUntil = l.now().Add(l.window)
		record.Failures = 0
		if err := l.storage.SetWithContext(ctx, keys.locked, []byte("1"), l.window); err != nil {
			return
		}
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = l.storage.SetWithContext(ctx, keys.failures, payload, l.window)
}

// Success clears the pair counters after a valid sign-in. The account-wide
// counter is cleared too: a successful sign-in proves the owner is present.
func (l *Lockout) Success(ctx context.Context, username, client string) {
	if !l.enabled() {
		return
	}
	pair, account := l.keys(username, client)
	for _, key := range []string{
		pair.failures, pair.locked, account.failures, account.locked,
	} {
		_ = l.storage.DeleteWithContext(ctx, key)
	}
}
