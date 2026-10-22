package oauth

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/markbates/goth"
)

// GothProvider adapts a goth provider to the neutral Provider boundary. The
// marshaled goth session is stored in the continuation store between the
// authorization redirect and the callback.
type GothProvider struct {
	Provider     goth.Provider
	DisplayLabel string
}

// NewGothProvider wraps a goth provider with a human-readable label.
func NewGothProvider(provider goth.Provider, label string) *GothProvider {
	return &GothProvider{Provider: provider, DisplayLabel: label}
}

func (g *GothProvider) Name() string {
	if g == nil || g.Provider == nil {
		return "oauth"
	}
	return g.Provider.Name()
}

func (g *GothProvider) Label() string {
	if g == nil || g.DisplayLabel == "" {
		return g.Name()
	}
	return g.DisplayLabel
}

// Start begins the goth auth transaction and persists its session blob.
func (g *GothProvider) Start(
	_ context.Context,
	request AuthorizationRequest,
	store ContinuationStore,
) (string, error) {
	if g == nil || g.Provider == nil {
		return "", ErrUnavailable
	}
	if store == nil {
		return "", ErrInvalidConfig
	}
	session, err := g.Provider.BeginAuth(request.State)
	if err != nil {
		return "", fmt.Errorf("%w: begin auth: %v", ErrUnavailable, err)
	}
	authorizationURL, err := session.GetAuthURL()
	if err != nil {
		return "", fmt.Errorf("%w: build authorization url: %v", ErrUnavailable, err)
	}
	if err := store.Save(request.State, session.Marshal()); err != nil {
		return "", fmt.Errorf("%w: persist provider session: %v", ErrUnavailable, err)
	}
	return authorizationURL, nil
}

// Complete validates the callback and loads the verified provider identity.
func (g *GothProvider) Complete(
	_ context.Context,
	exchange ExchangeRequest,
	store ContinuationStore,
) (Identity, error) {
	if g == nil || g.Provider == nil {
		return Identity{}, ErrUnavailable
	}
	if store == nil {
		return Identity{}, ErrInvalidConfig
	}
	raw, ok := store.Load(exchange.State)
	if !ok {
		return Identity{}, ErrInvalidState
	}
	session, err := g.Provider.UnmarshalSession(raw)
	if err != nil {
		return Identity{}, ErrInvalidState
	}
	params := exchange.Query
	if params == nil {
		params = url.Values{
			"code":  {exchange.Code},
			"state": {exchange.State},
		}
	}
	if _, err := session.Authorize(g.Provider, params); err != nil {
		return Identity{}, fmt.Errorf("%w: authorize: %v", ErrInvalidCode, err)
	}
	user, err := g.Provider.FetchUser(session)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: fetch user: %v", ErrUnavailable, err)
	}
	email := strings.TrimSpace(user.Email)
	return Identity{
		Provider:      g.Provider.Name(),
		Subject:       strings.TrimSpace(user.UserID),
		Email:         email,
		EmailVerified: email != "",
		DisplayName:   strings.TrimSpace(user.Name),
	}, nil
}
