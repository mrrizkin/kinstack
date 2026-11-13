package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/clock"

	"github.com/gofiber/fiber/v3"
	"github.com/valkey-io/valkey-go"
)

// accountWideLockoutFactor is how many times more failures lock the account
// for every client, so one attacker cannot lock a known account alone.
const accountWideLockoutFactor = 10

// lockoutRecord is per key; keys are hashes, so the store holds no identities.
type lockoutRecord struct {
	Failures    int       `json:"failures"`
	LockedUntil time.Time `json:"locked_until"`
}

// Lockout throttles failed password attempts per (account, client) pair, with a
// much higher account-wide threshold for distributed attacks. It is
// best-effort: a storage outage must not make sign-in impossible.
type Lockout struct {
	storage fiber.Storage
	max     int
	window  time.Duration
	now     func() time.Time
	logger  applog.Log
}

// NewLockout disables the guard on nil storage or a non-positive maximum.
func NewLockout(
	storage fiber.Storage,
	max int,
	window time.Duration,
	logger applog.Log,
) *Lockout {
	return NewLockoutAt(storage, max, window, clock.System, logger)
}

// NewLockoutAt takes the clock a simulation advances, so windows expire in it.
func NewLockoutAt(
	storage fiber.Storage,
	max int,
	window time.Duration,
	now func() time.Time,
	logger applog.Log,
) *Lockout {
	assert.Assert(now != nil, "lockout clock is required")
	assert.Assert(logger != nil, "lockout logger is required")
	return &Lockout{storage: storage, max: max, window: window, now: now, logger: logger}
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

// keys is per client (a coarse identity such as IP), so sharing an account does
// not lock the owner out of every device.
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

// valkeyFailureScript counts and sets the lock in one server operation; the
// hash-tagged keys share a cluster slot. ARGV: ttl ms, pair max, account max,
// account-wide flag.
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

// atomicClient is nil for storages without server-side scripting; those take
// the best-effort path below.
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
	if err != nil {
		// A store outage must not block sign-in, but it must be logged.
		l.logger.Debugw("lockout record read failed", "error", err)
		return lockoutRecord{}
	}
	if len(raw) == 0 {
		return lockoutRecord{}
	}
	var record lockoutRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		l.logger.Warnw("lockout record undecodable", "error", err)
		return lockoutRecord{}
	}
	return record
}

func (l *Lockout) locked(ctx context.Context, key string) bool {
	raw, err := l.storage.GetWithContext(ctx, key)
	if err != nil {
		l.logger.Debugw("lockout lookup failed", "error", err)
		return false
	}
	return len(raw) > 0
}

func (l *Lockout) Allowed(ctx context.Context, username, client string) bool {
	return l.AllowedPair(ctx, username, client) && !l.AccountLocked(ctx, username)
}

// AllowedPair ignores the account-wide lock: the break-glass account must not be
// lockable globally by a remote attacker.
func (l *Lockout) AllowedPair(ctx context.Context, username, client string) bool {
	if !l.enabled() {
		return true
	}
	pair, _ := l.keys(username, client)
	return !l.locked(ctx, pair.locked)
}

func (l *Lockout) AccountLocked(ctx context.Context, username string) bool {
	if !l.enabled() {
		return false
	}
	_, account := l.keys(username, "")
	return l.locked(ctx, account.locked)
}

// Failure counts one attempt; accountWide selects the higher threshold and is
// off for the break-glass account.
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

func (l *Lockout) bump(ctx context.Context, keys lockoutKeys, limit int) {
	record := l.readRecord(ctx, keys.failures)
	record.Failures++
	if record.Failures >= limit {
		record.LockedUntil = l.now().Add(l.window)
		record.Failures = 0
		if err := l.storage.SetWithContext(ctx, keys.locked, []byte("1"), l.window); err != nil {
			l.logger.Warnw("lockout record write failed", "error", err)
			return
		}
	}
	payload, err := json.Marshal(record)
	if err != nil {
		l.logger.Warnw("lockout record encode failed", "error", err)
		return
	}
	if err := l.storage.SetWithContext(ctx, keys.failures, payload, l.window); err != nil {
		l.logger.Warnw("lockout counter write failed", "error", err)
	}
}

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
