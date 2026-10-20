package oauth

import (
	"errors"
	"strings"
)

// ErrUnknownProvider identifies a provider slug that is not registered.
var ErrUnknownProvider = errors.New("oauth provider is not registered")

// Registry holds the enabled providers in a deterministic order.
type Registry struct {
	providers map[string]Provider
	order     []string
}

// NewRegistry constructs an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

// Register adds a provider. Duplicate names are ignored so composition stays
// idempotent.
func (r *Registry) Register(provider Provider) {
	if r == nil || provider == nil {
		return
	}
	name := strings.TrimSpace(provider.Name())
	if name == "" {
		return
	}
	if _, exists := r.providers[name]; exists {
		return
	}
	r.providers[name] = provider
	r.order = append(r.order, name)
}

// Get returns the provider registered under a slug.
func (r *Registry) Get(name string) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.providers[strings.TrimSpace(name)]
	return provider, ok
}

// Providers returns the registered providers in registration order.
func (r *Registry) Providers() []Provider {
	if r == nil {
		return nil
	}
	list := make([]Provider, 0, len(r.order))
	for _, name := range r.order {
		list = append(list, r.providers[name])
	}
	return list
}

// Empty reports whether no provider is registered.
func (r *Registry) Empty() bool {
	return r == nil || len(r.providers) == 0
}
