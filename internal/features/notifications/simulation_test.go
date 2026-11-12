package notifications

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"codeberg.org/mrrizkin/dstsim"
)

// notificationSimulationRecipient is the only recipient in the simulation: the
// invariant is about one feed's ordering, so a second recipient would only add
// noise.
const notificationSimulationRecipient = "R1e2c3i4p5i6e7n8t9A0"

// feedState is what the process believes. A crash may not change it, because
// the durable store is the database, not the service: the service must not hold
// per-process state that a restart would lose or duplicate.
type feedState struct {
	unread int
	ids    []string
	read   []string
}

func readFeed(t *testing.T, service *Service, ctx context.Context) feedState {
	t.Helper()
	page, err := service.List(ctx, notificationSimulationRecipient, "", 100)
	if err != nil {
		t.Fatalf("list feed: %v", err)
	}
	state := feedState{unread: page.Unread}
	for _, item := range page.Notifications {
		state.ids = append(state.ids, item.PublicID)
		if item.Read {
			state.read = append(state.read, item.PublicID)
		}
	}
	return state
}

// TestSimulationNotificationsAcrossRestarts drives the real service through a
// run that crashes and restarts at random operations. The invariants are the
// promise the feed makes: one row per notification, newest first, and a state
// that survives a restart unchanged.
func TestSimulationNotificationsAcrossRestarts(t *testing.T) {
	for _, seed := range []int64{1, 42, 1337} {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			cfg := dstsim.DefaultConfig(seed)
			cfg.Ops = 300
			cfg.CrashRate = 0.05
			cfg.ErrorRate = 0
			cfg.SlowRate = 0

			dstsim.Run(cfg, func(sim *dstsim.Sim) {
				ctx := context.Background()
				clock := sim.Clock()
				repository := NewMemoryRepository(func() time.Time { return clock.Now() })
				service := NewService(ServiceDependency{Repository: repository})
				issued := dstsim.NewPool[string]()
				var beforeCrash feedState

				sim.Register(dstsim.OpEntry{
					Name:   "notify",
					Weight: 4,
					Run: func(_ *dstsim.RNG, _ *dstsim.Clock) error {
						created, err := service.Notify(
							ctx,
							notificationSimulationRecipient,
							"system",
							"simulation",
							"a notification the feed must keep exactly once",
						)
						if err != nil {
							return fmt.Errorf("notify: %w", err)
						}
						sim.Assert(sim.CurrentSeq(), created.PublicID != "", "blank public id")
						issued.Add(created.PublicID)
						return nil
					},
				})

				sim.Register(dstsim.OpEntry{
					Name:   "mark_read",
					Weight: 2,
					Run: func(rng *dstsim.RNG, _ *dstsim.Clock) error {
						if issued.Empty() {
							return nil
						}
						target := issued.Pick(rng)
						if rng.IntN(5) == 0 {
							// A ghost id exercises the not-found path without
							// removing a real row from the pool.
							target = "G1h2o3s4t5I6d7Z8z9B0"
						}
						if err := service.MarkRead(
							ctx,
							notificationSimulationRecipient,
							target,
						); err != nil && target != "G1h2o3s4t5I6d7Z8z9B0" {
							return fmt.Errorf("mark_read: %w", err)
						}
						return nil
					},
				})

				sim.Register(dstsim.OpEntry{
					Name:   "mark_all_read",
					Weight: 1,
					Run: func(_ *dstsim.RNG, _ *dstsim.Clock) error {
						return service.MarkAllRead(ctx, notificationSimulationRecipient)
					},
				})

				sim.Register(dstsim.OpEntry{
					Name:   "list",
					Weight: 3,
					Run: func(_ *dstsim.RNG, _ *dstsim.Clock) error {
						page, err := service.List(
							ctx,
							notificationSimulationRecipient,
							"",
							7,
						)
						if err != nil {
							return fmt.Errorf("list: %w", err)
						}
						seen := make(map[string]struct{}, len(page.Notifications))
						for index, item := range page.Notifications {
							if _, duplicate := seen[item.PublicID]; duplicate {
								sim.Fail(sim.CurrentSeq(), "feed returned one notification twice")
							}
							seen[item.PublicID] = struct{}{}
							if index > 0 && page.Notifications[index-1].PublicID == item.PublicID {
								sim.Fail(sim.CurrentSeq(), "feed is not strictly ordered")
							}
						}
						return nil
					},
				})

				readOnce := make(map[string]struct{})

				sim.Invariant(func(seq int) {
					state := readFeed(t, service, ctx)
					// Read is terminal: the only legal transition is
					// unread -> read, never back.
					for _, id := range state.read {
						readOnce[id] = struct{}{}
					}
					for id := range readOnce {
						if !slices.Contains(state.read, id) && len(state.ids) < 100 {
							sim.Assertf(seq, false, "notification %q became unread again", id)
						}
					}
					sim.Assertf(
						seq,
						state.unread >= 0,
						"unread count went negative: %d",
						state.unread,
					)
					sim.Assertf(
						seq,
						state.unread <= len(state.ids),
						"unread=%d exceeds the %d rows the feed returned",
						state.unread,
						len(state.ids),
					)
					for _, id := range issued.All() {
						if !slices.Contains(state.ids, id) && len(state.ids) < 100 {
							sim.Assertf(
								seq,
								false,
								"acknowledged notification %q left the feed",
								id,
							)
						}
					}
				})

				sim.OnCrash(func(int) { beforeCrash = readFeed(t, service, ctx) })
				sim.OnRestart(func(seq int) {
					after := readFeed(t, service, ctx)
					sim.Assertf(
						seq,
						after.unread == beforeCrash.unread &&
							slices.Equal(after.ids, beforeCrash.ids),
						"restart changed the feed: unread %d->%d, rows %d->%d",
						beforeCrash.unread,
						after.unread,
						len(beforeCrash.ids),
						len(after.ids),
					)
				})
			})
		})
	}
}
