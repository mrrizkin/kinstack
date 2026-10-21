package oauth

import (
	"context"
	"testing"
)

type fakeProvider struct {
	name  string
	label string
}

func (f fakeProvider) Name() string  { return f.name }
func (f fakeProvider) Label() string { return f.label }

func (f fakeProvider) Start(
	context.Context,
	AuthorizationRequest,
	ContinuationStore,
) (string, error) {
	return "https://example.test/auth", nil
}

func (f fakeProvider) Complete(
	context.Context,
	ExchangeRequest,
	ContinuationStore,
) (Identity, error) {
	return Identity{Subject: f.name}, nil
}

func TestRegistryKeepsRegistrationOrderAndDeduplicates(t *testing.T) {
	registry := NewRegistry()
	registry.Register(fakeProvider{name: "google", label: "Google"})
	registry.Register(fakeProvider{name: "github", label: "GitHub"})
	registry.Register(fakeProvider{name: "google", label: "Google again"})

	providers := registry.Providers()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].Name() != "google" || providers[1].Name() != "github" {
		t.Fatalf("unexpected order: %v, %v", providers[0].Name(), providers[1].Name())
	}
	if provider, ok := registry.Get("github"); !ok || provider.Label() != "GitHub" {
		t.Fatalf("unexpected lookup: %v %v", provider, ok)
	}
	if _, ok := registry.Get("missing"); ok {
		t.Fatalf("missing provider should not resolve")
	}
}

func TestNilRegistryIsEmpty(t *testing.T) {
	var registry *Registry
	if !registry.Empty() {
		t.Fatalf("nil registry should be empty")
	}
}
