package config

import (
	"strings"
	"testing"
)

func TestValkeyConfigValidate(t *testing.T) {
	cfg := ValkeyConfig{Host: "localhost", Port: 6379, Database: 0}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid Valkey config rejected: %v", err)
	}
}

func TestValkeyConfigProductionRequiresPassword(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := ValkeyConfig{Host: "valkey.example.test", Port: 6379}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "VALKEY_PASSWORD") {
		t.Fatalf("expected production password error, got %v", err)
	}
}
