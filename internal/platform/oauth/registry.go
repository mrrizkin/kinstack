package oauth

import (
	"errors"
	"strings"
)

var ErrUnknownProvider = errors.New("oauth provider is not registered")

// Registry holds the enabled providers in a deterministic order.
type Registry struct {
	providers map[string]Provider
	order     []string
}

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

func (r *Registry) Get(name string) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.providers[strings.TrimSpace(name)]
	return provider, ok
}

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

func (r *Registry) Empty() bool {
	return r == nil || len(r.providers) == 0
}
