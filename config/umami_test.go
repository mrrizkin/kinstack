package config

import (
	"strings"
	"testing"
)

func TestUmamiDisabledIgnoresSettings(t *testing.T) {
	cfg := UmamiConfig{Enabled: false}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled Umami rejected: %v", err)
	}
}

func TestUmamiConfigLoadAndValidate(t *testing.T) {
	t.Setenv("UMAMI_ENABLED", "true")
	t.Setenv("UMAMI_SCRIPT_URL", "https://umami.example.test/script.js")
	t.Setenv("UMAMI_WEBSITE_ID", "website-1")
	t.Setenv("UMAMI_DOMAINS", "kinstack.example.test")
	t.Setenv("UMAMI_AUTO_TRACK", "false")

	var cfg UmamiConfig
	cfg.Load()

	if !cfg.Enabled || cfg.WebsiteID != "website-1" || cfg.AutoTrack {
		t.Fatalf("unexpected Umami config: %#v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid Umami config rejected: %v", err)
	}
}

func TestUmamiRequiresWebsiteID(t *testing.T) {
	cfg := UmamiConfig{
		Enabled:   true,
		ScriptURL: "https://umami.example.test/script.js",
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "UMAMI_WEBSITE_ID") {
		t.Fatalf("expected website id error, got %v", err)
	}
}

func TestUmamiProductionRequiresHTTPS(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := UmamiConfig{
		Enabled:   true,
		ScriptURL: "http://umami.example.test/script.js",
		WebsiteID: "website-1",
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected HTTPS error, got %v", err)
	}
}
