package config

import (
	"strings"
	"testing"
	"time"
)

func TestAppConfigLoadAndValidate(t *testing.T) {
	t.Setenv("APP_PORT", "3010")
	t.Setenv("AUTH_RATE_LIMIT_MAX", "17")
	t.Setenv("AUTH_RATE_LIMIT_WINDOW", "45s")

	var cfg AppConfig
	cfg.Load()

	if cfg.Port != 3010 || cfg.AuthRateLimitMax != 17 ||
		cfg.AuthRateLimitWindow != 45*time.Second {
		t.Fatalf("unexpected app config: %#v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid app config rejected: %v", err)
	}
}

func TestAppConfigRejectsMalformedPort(t *testing.T) {
	t.Setenv("APP_PORT", "not-a-port")
	defer func() {
		if recovered := recover(); recovered == nil ||
			!strings.Contains(recovered.(error).Error(), "APP_PORT") {
			t.Fatalf("panic=%v, want APP_PORT parsing panic", recovered)
		}
	}()
	var cfg AppConfig
	cfg.Load()
}

func TestAppConfigRejectsInvalidAuthRateLimit(t *testing.T) {
	cfg := AppConfig{
		Name:                "KinStack",
		Env:                 "development",
		URL:                 "http://localhost:3000",
		Port:                3000,
		SessionIdleTimeout:  time.Hour,
		AuthRateLimitMax:    0,
		AuthRateLimitWindow: time.Minute,
		APIRateLimitMax:     10,
		APIRateLimitWindow:  time.Minute,
		HTTPBodyLimit:       12 << 20,
		HTTPReadTimeout:     time.Second,
		HTTPWriteTimeout:    time.Second,
		HTTPIdleTimeout:     time.Second,
	}
	if err := cfg.Validate(); err == nil ||
		!strings.Contains(err.Error(), "auth rate limit") {
		t.Fatalf("expected auth rate limit validation error, got %v", err)
	}
}

func TestAppConfigProductionRequiresHTTPS(t *testing.T) {
	cfg := AppConfig{
		Name:                "KinStack",
		Env:                 "production",
		URL:                 "http://app.example.test",
		Port:                3000,
		SessionIdleTimeout:  time.Hour,
		AuthRateLimitMax:    5,
		AuthRateLimitWindow: time.Minute,
		APIRateLimitMax:     10,
		APIRateLimitWindow:  time.Minute,
		HTTPBodyLimit:       12 << 20,
		HTTPReadTimeout:     time.Second,
		HTTPWriteTimeout:    time.Second,
		HTTPIdleTimeout:     time.Second,
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected HTTPS validation error, got %v", err)
	}
}

func TestAppConfigRequiresStrongBreakGlassSecret(t *testing.T) {
	cfg := AppConfig{
		Name:                "KinStack",
		Env:                 "development",
		URL:                 "http://localhost:3000",
		Port:                3000,
		SessionIdleTimeout:  time.Hour,
		AuthRateLimitMax:    5,
		AuthRateLimitWindow: time.Minute,
		APIRateLimitMax:     10,
		APIRateLimitWindow:  time.Minute,
		HTTPBodyLimit:       12 << 20,
		HTTPReadTimeout:     time.Second,
		HTTPWriteTimeout:    time.Second,
		HTTPIdleTimeout:     time.Minute,
		AccessKey:           "superadmin",
		SecretKey:           strings.Repeat("x", 31),
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "SECRET_KEY") {
		t.Fatalf("expected short SECRET_KEY rejection, got %v", err)
	}

	cfg.SecretKey = strings.Repeat("x", 32)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("32-byte SECRET_KEY should be accepted: %v", err)
	}
}
