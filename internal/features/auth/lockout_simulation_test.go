package auth

import (
	"context"
	"testing"
	"time"

	"codeberg.org/mrrizkin/dstsim"
)

const (
	simulationFailureRecord    = 3
	simulationLockoutWindow    = 15 * time.Minute
	simulationAccountThreshold = simulationFailureRecord * accountWideLockoutFactor
)

// simulatedStorage is a fiber.Storage whose expiry follows the simulation clock.
// The in-memory storage shipped with Fiber expires keys against the wall clock,
// which would make a lock window impossible to test deterministically.
type simulatedStorage struct {
	clock *dstsim.Clock
	items map[string]simulatedItem
}

type simulatedItem struct {
	value     []byte
	expiresAt time.Time
}

func newSimulatedStorage(clock *dstsim.Clock) *simulatedStorage {
	return &simulatedStorage{clock: clock, items: map[string]simulatedItem{}}
}

func (s *simulatedStorage) live(key string) (simulatedItem, bool) {
	item, ok := s.items[key]
	if !ok {
		return simulatedItem{}, false
	}
	if !item.expiresAt.IsZero() && !s.clock.Now().Before(item.expiresAt) {
		delete(s.items, key)
		return simulatedItem{}, false
	}
	return item, true
}

func (s *simulatedStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *simulatedStorage) GetWithContext(_ context.Context, key string) ([]byte, error) {
	item, ok := s.live(key)
	if !ok {
		return nil, nil
	}
	return item.value, nil
}

func (s *simulatedStorage) Set(key string, value []byte, expiry time.Duration) error {
	return s.SetWithContext(context.Background(), key, value, expiry)
}

func (s *simulatedStorage) SetWithContext(
	_ context.Context,
	key string,
	value []byte,
	expiry time.Duration,
) error {
	if key == "" || len(value) == 0 {
		return nil
	}
	item := simulatedItem{value: value}
	if expiry > 0 {
		item.expiresAt = s.clock.Now().Add(expiry)
	}
	s.items[key] = item
	return nil
}

func (s *simulatedStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *simulatedStorage) DeleteWithContext(_ context.Context, key string) error {
	delete(s.items, key)
	return nil
}

func (s *simulatedStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (s *simulatedStorage) ResetWithContext(_ context.Context) error {
	s.items = map[string]simulatedItem{}
	return nil
}

func (s *simulatedStorage) Close() error { return nil }

// lockoutModel is the guard's promise written down independently: a run of
// failures inside one window locks, a success clears, and silence expires.
type lockoutModel struct {
	failures map[string]int
	last     map[string]time.Time
	lockedTo map[string]time.Time
}

func newLockoutModel() *lockoutModel {
	return &lockoutModel{
		failures: map[string]int{},
		last:     map[string]time.Time{},
		lockedTo: map[string]time.Time{},
	}
}

func (m *lockoutModel) expire(key string, now time.Time) {
	last, ok := m.last[key]
	if !ok || now.Sub(last) <= simulationLockoutWindow {
		return
	}
	delete(m.failures, key)
	delete(m.last, key)
	delete(m.lockedTo, key)
}

func (m *lockoutModel) failure(key string, limit int, now time.Time) {
	m.expire(key, now)
	m.failures[key]++
	m.last[key] = now
	if m.failures[key] >= limit {
		m.failures[key] = 0
		m.lockedTo[key] = now.Add(simulationLockoutWindow)
	}
}

func (m *lockoutModel) success(keys ...string) {
	for _, key := range keys {
		delete(m.failures, key)
		delete(m.last, key)
		delete(m.lockedTo, key)
	}
}

func (m *lockoutModel) locked(key string, now time.Time) bool {
	m.expire(key, now)
	until, ok := m.lockedTo[key]
	return ok && now.Before(until)
}

// TestSimulationLockout drives the real guard against an independent model of
// its promise, with a clock the simulation controls.
func TestSimulationLockout(t *testing.T) {
	for _, seed := range []int64{7, 21, 99} {
		seed := seed
		t.Run("seed"+string(rune('a'+seed%26)), func(t *testing.T) {
			cfg := dstsim.DefaultConfig(seed)
			cfg.Ops = 250
			cfg.ReplayHint = func(seed int64) string {
				return "go test ./internal/features/auth -run TestSimulationLockout"
			}
			dstsim.NewHarness(cfg, lockoutSimulation).Run()
		})
	}
}

func lockoutSimulation(sim *dstsim.Sim) {
	clock := sim.Clock()
	guard := NewLockoutAt(
		newSimulatedStorage(clock),
		simulationFailureRecord,
		simulationLockoutWindow,
		clock.Now,
	)
	model := newLockoutModel()
	ctx := context.Background()

	users := []string{"alpha", "beta"}
	clients := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	pairKey := func(user, client string) string { return user + "|" + client }
	accountKey := func(user string) string { return user + "|*" }

	register := func(name string, weight int, run func(rng *dstsim.RNG) error) {
		sim.Register(dstsim.OpEntry{
			Name:   name,
			Weight: weight,
			Run:    func(rng *dstsim.RNG, _ *dstsim.Clock) error { return run(rng) },
		})
	}

	register("fail", 6, func(rng *dstsim.RNG) error {
		user := dstsim.Pick(rng, users)
		client := dstsim.Pick(rng, clients)
		accountWide := rng.IntN(2) == 0
		now := clock.Now()
		guard.Failure(ctx, user, client, accountWide)
		model.failure(pairKey(user, client), simulationFailureRecord, now)
		if accountWide {
			model.failure(accountKey(user), simulationAccountThreshold, now)
		}
		return nil
	})

	register("succeed", 2, func(rng *dstsim.RNG) error {
		user := dstsim.Pick(rng, users)
		client := dstsim.Pick(rng, clients)
		guard.Success(ctx, user, client)
		model.success(pairKey(user, client), accountKey(user))
		return nil
	})

	register("advance_clock", 3, func(rng *dstsim.RNG) error {
		clock.Advance(time.Duration(rng.IntN(20)+1) * time.Minute)
		return nil
	})

	sim.Invariant(func(seq int) {
		now := clock.Now()
		for _, user := range users {
			wantAccountLocked := model.locked(accountKey(user), now)
			gotAccountLocked := guard.AccountLocked(ctx, user)
			if wantAccountLocked != gotAccountLocked {
				sim.Fail(seq, "account lock mismatch for "+user)
			}
			for _, client := range clients {
				wantLocked := model.locked(pairKey(user, client), now)
				wantAllowed := !wantLocked
				if got := guard.AllowedPair(ctx, user, client); got != wantAllowed {
					sim.Fail(seq, "pair lock mismatch for "+pairKey(user, client))
				}
			}
		}
	})
}
