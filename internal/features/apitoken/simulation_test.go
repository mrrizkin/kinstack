package apitoken

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"kinstack/internal/pkg/pagination"

	"codeberg.org/mrrizkin/dstsim"
)

// A token ceiling keeps every invariant check bounded, so a run cannot grow
// into an accidental load test.
const simulationTokenCeiling = 24

// sandboxToken is the simulation's model of one issued token. It records what
// the service promised, so an invariant can compare the promise with behavior.
type sandboxToken struct {
	raw          string
	publicID     string
	userPublicID string
	expiresAt    *time.Time
	scopes       []string
	revoked      bool
}

func (t *sandboxToken) expiredAt(now time.Time) bool {
	return t.expiresAt != nil && !t.expiresAt.After(now)
}

// TestSimulationAPITokenLifecycle drives the real service and the in-memory
// repository through create, authenticate, revoke, revocation of every token,
// and expiry, with a simulated clock and a lossy link in front of authentication.
func TestSimulationAPITokenLifecycle(t *testing.T) {
	for _, seed := range []int64{1, 42, 1337} {
		seed := seed
		t.Run(fmt.Sprintf("seed%02d", seed), func(t *testing.T) {
			cfg := dstsim.DefaultConfig(seed)
			cfg.Ops = 300
			cfg.Net = dstsim.NetPolicy{
				LatencyMax: 5 * time.Millisecond,
				DropRate:   0.05,
				DupRate:    0.02,
				LossRate:   0.05,
			}
			cfg.ReplayHint = func(seed int64) string {
				return fmt.Sprintf(
					"go test ./internal/features/apitoken -run 'TestSimulationAPITokenLifecycle/seed%02d'",
					seed,
				)
			}
			dstsim.NewHarness(cfg, apitokenSandbox).Run()
		})
	}
}

// apitokenSandbox builds a fresh world for one run. dstsim calls it once per
// shrink attempt, so every repository, service and pool is recreated here.
func apitokenSandbox(sim *dstsim.Sim) {
	clock := sim.Clock()
	service := NewService(ServiceDependency{
		Repository: NewMemoryRepositoryAt(clock.Now),
		Now:        clock.Now,
	})
	network := sim.Network()
	ctx := context.Background()

	tokens := make([]*sandboxToken, 0, simulationTokenCeiling)
	users := []string{"user-alpha", "user-beta"}
	scopeChoices := [][]string{{}, {"users.view"}, {"users.view", "audit.view"}}
	expiryChoices := []int{0, 1, 30}

	activeSince := func(now time.Time) (*sandboxToken, bool) {
		for index := len(tokens) - 1; index >= 0; index-- {
			token := tokens[index]
			if token.revoked || token.expiredAt(now) {
				continue
			}
			return token, true
		}
		return nil, false
	}
	liveFor := func(userPublicID string) []*sandboxToken {
		out := make([]*sandboxToken, 0, len(tokens))
		for _, token := range tokens {
			if !token.revoked && token.userPublicID == userPublicID {
				out = append(out, token)
			}
		}
		return out
	}
	register := func(name string, weight int, run func(rng *dstsim.RNG) error) {
		sim.Register(dstsim.OpEntry{
			Name:   name,
			Weight: weight,
			Run:    func(rng *dstsim.RNG, _ *dstsim.Clock) error { return run(rng) },
		})
	}

	register("create", 4, func(rng *dstsim.RNG) error {
		if len(tokens) >= simulationTokenCeiling {
			return nil
		}
		fake := dstsim.NewFake(rng)
		userPublicID := dstsim.Pick(rng, users)
		scopes := scopeChoices[rng.IntN(len(scopeChoices))]
		expiresInDays := expiryChoices[rng.IntN(len(expiryChoices))]
		created, err := service.Create(
			ctx,
			userPublicID,
			fake.Name(),
			expiresInDays,
			slices.Clone(scopes),
		)
		if err != nil {
			return fmt.Errorf("create: %w", err)
		}

		seq := sim.CurrentSeq()
		sim.Assertf(
			seq,
			strings.HasPrefix(created.Token, created.APIToken.Prefix),
			"create: token %q does not start with prefix %q",
			created.Token,
			created.APIToken.Prefix,
		)
		sim.Assertf(
			seq,
			created.APIToken.Prefix == tokenScheme+created.APIToken.PublicID[:tokenPrefixChars],
			"create: prefix %q does not derive from public id %q",
			created.APIToken.Prefix,
			created.APIToken.PublicID,
		)
		sim.Assertf(
			seq,
			slices.Equal(created.APIToken.Scopes, scopes),
			"create: scopes %v, want %v",
			created.APIToken.Scopes,
			scopes,
		)

		var expiresAt *time.Time
		if created.APIToken.ExpiresAt != "" {
			parsed, parseErr := time.Parse(time.RFC3339, created.APIToken.ExpiresAt)
			if parseErr != nil {
				sim.Fail(
					seq,
					fmt.Sprintf("create: expires_at %q: %v", created.APIToken.ExpiresAt, parseErr),
				)
			}
			expiresAt = &parsed
		}
		tokens = append(tokens, &sandboxToken{
			raw:          created.Token,
			publicID:     created.APIToken.PublicID,
			userPublicID: userPublicID,
			expiresAt:    expiresAt,
			scopes:       created.APIToken.Scopes,
		})
		return nil
	})

	register("authenticate_live", 5, func(rng *dstsim.RNG) error {
		token, ok := activeSince(clock.Now())
		if !ok {
			return nil
		}
		var (
			userPublicID string
			scopes       []string
		)
		err := network.Do(func() error {
			var callErr error
			userPublicID, scopes, callErr = service.Authenticate(ctx, token.raw)
			return callErr
		})
		if err != nil {
			if errors.Is(err, dstsim.ErrDropped) ||
				errors.Is(err, dstsim.ErrLostReply) ||
				errors.Is(err, dstsim.ErrTimeout) {
				return nil
			}
			return fmt.Errorf("authenticate live token: %w", err)
		}
		seq := sim.CurrentSeq()
		sim.Assertf(
			seq,
			userPublicID == token.userPublicID,
			"authenticate: user %q, want %q",
			userPublicID,
			token.userPublicID,
		)
		sim.Assertf(
			seq,
			slices.Equal(scopes, token.scopes),
			"authenticate: scopes %v, want %v",
			scopes,
			token.scopes,
		)
		return nil
	})

	register("revoke", 3, func(rng *dstsim.RNG) error {
		candidates := make([]*sandboxToken, 0, len(tokens))
		for _, token := range tokens {
			if !token.revoked {
				candidates = append(candidates, token)
			}
		}
		if len(candidates) == 0 {
			return nil
		}
		token := dstsim.Pick(rng, candidates)
		if err := service.Revoke(ctx, token.userPublicID, token.publicID); err != nil {
			return fmt.Errorf("revoke: %w", err)
		}
		token.revoked = true
		return nil
	})

	register("revoke_all", 1, func(rng *dstsim.RNG) error {
		userPublicID := dstsim.Pick(rng, users)
		if err := service.RevokeAll(ctx, userPublicID); err != nil {
			return fmt.Errorf("revoke_all: %w", err)
		}
		for _, token := range tokens {
			if token.userPublicID == userPublicID {
				token.revoked = true
			}
		}
		return nil
	})

	register("advance_clock", 2, func(rng *dstsim.RNG) error {
		clock.Advance(time.Duration(rng.IntN(48)+1) * time.Hour)
		return nil
	})

	register("list", 2, func(rng *dstsim.RNG) error {
		userPublicID := dstsim.Pick(rng, users)
		page, err := service.List(
			ctx,
			userPublicID,
			pagination.Query{Page: 1, PerPage: simulationTokenCeiling + 1},
		)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		expected := liveFor(userPublicID)
		seq := sim.CurrentSeq()
		sim.Assertf(
			seq,
			page.Page.Total == len(expected),
			"list: total %d, want %d",
			page.Page.Total,
			len(expected),
		)
		sim.Assertf(
			seq,
			len(page.Tokens) == len(expected),
			"list: returned %d tokens, want %d",
			len(page.Tokens),
			len(expected),
		)
		return nil
	})

	sim.Invariant(func(seq int) {
		now := clock.Now()
		seen := make(map[string]struct{}, len(tokens))
		for _, token := range tokens {
			if _, duplicate := seen[token.publicID]; duplicate {
				sim.Fail(seq, fmt.Sprintf("duplicate public id %q", token.publicID))
			}
			seen[token.publicID] = struct{}{}

			switch {
			case token.revoked:
				assertRejected(sim, seq, service, token, "revoked")
			case token.expiredAt(now):
				assertRejected(sim, seq, service, token, "expired")
			default:
				userPublicID, scopes, err := service.Authenticate(ctx, token.raw)
				if err != nil {
					sim.Fail(seq, fmt.Sprintf("live token %q rejected: %v", token.publicID, err))
				}
				if userPublicID != token.userPublicID {
					sim.Fail(
						seq,
						fmt.Sprintf(
							"live token %q authenticated as %q, want %q",
							token.publicID,
							userPublicID,
							token.userPublicID,
						),
					)
				}
				if !slices.Equal(scopes, token.scopes) {
					sim.Fail(
						seq,
						fmt.Sprintf(
							"live token %q scopes %v, want %v",
							token.publicID,
							scopes,
							token.scopes,
						),
					)
				}
			}
		}
	})
}

func assertRejected(
	sim *dstsim.Sim,
	seq int,
	service *Service,
	token *sandboxToken,
	reason string,
) {
	if _, _, err := service.Authenticate(
		context.Background(),
		token.raw,
	); !errors.Is(
		err,
		ErrInvalidToken,
	) {
		sim.Fail(seq, fmt.Sprintf("%s token %q authenticated: err=%v", reason, token.publicID, err))
	}
}
