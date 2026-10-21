package config

import (
	"strings"
	"testing"
)

func TestOpenObserveDisabledIgnoresSettings(t *testing.T) {
	cfg := OpenObserveConfig{Enabled: false, TracesSampleRate: 9}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled OpenObserve rejected: %v", err)
	}
}

func TestOpenObserveConfigLoadNormalizesEndpoint(t *testing.T) {
	t.Setenv("OPENOBSERVE_ENABLED", "true")
	t.Setenv("OPENOBSERVE_ENDPOINT", "https://observe.example.test/")
	t.Setenv("OPENOBSERVE_ORG", "kinstack")
	t.Setenv("OPENOBSERVE_AUTH_HEADER", "Basic dGVzdDp0ZXN0")
	t.Setenv("OPENOBSERVE_TRACES_SAMPLE_RATE", "0.5")

	var cfg OpenObserveConfig
	cfg.Load()

	if cfg.Endpoint != "https://observe.example.test" {
		t.Fatalf("endpoint not normalized: %q", cfg.Endpoint)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid OpenObserve config rejected: %v", err)
	}
}

func TestOpenObserveRequiresOrganization(t *testing.T) {
	cfg := OpenObserveConfig{
		Enabled:          true,
		Endpoint:         "https://observe.example.test",
		StreamName:       "kinstack",
		ServiceName:      "kinstack",
		TracesSampleRate: 1,
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "OPENOBSERVE_ORG") {
		t.Fatalf("expected organization error, got %v", err)
	}
}

func TestOpenObserveProductionRequiresAuth(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := OpenObserveConfig{
		Enabled:          true,
		Endpoint:         "https://observe.example.test",
		Organization:     "kinstack",
		StreamName:       "kinstack",
		ServiceName:      "kinstack",
		TracesSampleRate: 1,
	}
	if err := cfg.Validate(); err == nil ||
		!strings.Contains(err.Error(), "OPENOBSERVE_AUTH_HEADER") {
		t.Fatalf("expected auth error, got %v", err)
	}
}
