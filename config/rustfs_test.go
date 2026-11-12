package config

import (
	"strings"
	"testing"
	"time"
)

func TestRustFSConfigDefaultsTimeout(t *testing.T) {
	t.Setenv("RUSTFS_TIMEOUT", "")
	var cfg RustFSConfig
	cfg.Load()
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("default RUSTFS_TIMEOUT = %v, want 30s", cfg.Timeout)
	}
}

// Object storage is an external dependency; a missing bound must fail at
// startup rather than hang a request (PILLARS §2).
func TestRustFSConfigRejectsMissingTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg := RustFSConfig{
		Enabled:   true,
		Endpoint:  "http://localhost:9000",
		PublicURL: "http://localhost:9000",
		AccessKey: "access",
		SecretKey: "secret",
		Bucket:    "kinstack",
		Region:    "us-east-1",
		Timeout:   0,
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "RUSTFS_TIMEOUT") {
		t.Fatalf("expected RUSTFS_TIMEOUT validation error, got %v", err)
	}
}

func TestRustFSConfigValidateSkipsDisabledStorage(t *testing.T) {
	var cfg RustFSConfig
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled storage must not validate: %v", err)
	}
}
