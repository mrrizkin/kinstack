package config

import (
	"strings"
	"testing"
)

func TestGlitchTipDisabledIgnoresSettings(t *testing.T) {
	cfg := GlitchTipConfig{DSN: "", TracesSampleRate: 5}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled GlitchTip rejected: %v", err)
	}
}

func TestGlitchTipConfigLoadAndValidate(t *testing.T) {
	t.Setenv("GLITCHTIP_DSN", "https://key@glitchtip.example.test/1")
	t.Setenv("GLITCHTIP_ENVIRONMENT", "staging")
	t.Setenv("GLITCHTIP_TRACES_SAMPLE_RATE", "0.25")

	var cfg GlitchTipConfig
	cfg.Load()

	if !cfg.Enabled || cfg.Environment != "staging" || cfg.TracesSampleRate != 0.25 {
		t.Fatalf("unexpected GlitchTip config: %#v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid GlitchTip config rejected: %v", err)
	}
}

func TestGlitchTipRejectsOutOfRangeSampleRate(t *testing.T) {
	cfg := GlitchTipConfig{
		Enabled:          true,
		DSN:              "https://key@glitchtip.example.test/1",
		TracesSampleRate: 1.5,
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "SAMPLE_RATE") {
		t.Fatalf("expected sample rate error, got %v", err)
	}
}

func TestGlitchTipProductionRequiresHTTPS(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := GlitchTipConfig{
		Enabled: true,
		DSN:     "http://key@glitchtip.example.test/1",
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected HTTPS error, got %v", err)
	}
}
