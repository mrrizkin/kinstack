package config

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func generatedSecret(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("read random: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

// Every credential shipped in .env.example must be rejected in production,
// otherwise a copied environment file would go live with published secrets.
func TestPlaceholderSecretRejectsExampleValues(t *testing.T) {
	examples := map[string]string{
		"valkey example password": "kinstack-local-password",
		"break glass username":    "superadmin",
		"break glass secret":      "replace-with-random-secret-at-least-32-characters",
		"rustfs access key":       "rustfsadmin",
		"rustfs secret key":       "rustfsadmin",
		"glitchtip secret":        "local-glitchtip-secret-change-me",
		"umami secret":            "local-umami-secret-change-me",
		"umami two factor key":    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"openobserve password":    "KinStack!2026",
		"database password":       "kinstack",
		"template value":          "replace-with-a-secret",
		"long template value":     "replace-with-random-secret-at-least-32-characters",
	}
	for name, value := range examples {
		if !placeholderSecret(value) {
			t.Errorf("%s %q should be rejected as a placeholder", name, value)
		}
	}
}

func TestPlaceholderSecretAcceptsGeneratedSecrets(t *testing.T) {
	for range 50 {
		value := generatedSecret(t)
		if placeholderSecret(value) {
			t.Fatalf("generated secret %q rejected as placeholder", value)
		}
	}
}

func TestPlaceholderSecretRejectsLowEntropy(t *testing.T) {
	for _, value := range []string{
		"",
		"aaaaaaaaaaaaaaaaaaaaaaaa",
		"abcabcabcabcabcabc",
		"1234567890123",
		"short1",
	} {
		if !placeholderSecret(value) {
			t.Errorf("%q should be rejected as low entropy", value)
		}
	}
}

func TestPlaceholderSecretAcceptsPassphrases(t *testing.T) {
	for _, value := range []string{
		"correct horse battery staple",
		"kucing-oranye-melompat-pagi",
	} {
		if placeholderSecret(value) {
			t.Errorf("passphrase %q rejected", value)
		}
	}
}

func TestProductionRejectsExampleValkeyPassword(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := ValkeyConfig{
		Host:     "valkey.example.test",
		Port:     6379,
		Password: "kinstack-local-password",
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "VALKEY_PASSWORD") {
		t.Fatalf("expected VALKEY_PASSWORD placeholder error, got %v", err)
	}
}

func TestProductionAcceptsGeneratedValkeyPassword(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := ValkeyConfig{Host: "valkey.example.test", Port: 6379, Password: generatedSecret(t)}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated password rejected: %v", err)
	}
}

func TestProductionRejectsExampleRustFSCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := RustFSConfig{
		Enabled:   true,
		Endpoint:  "https://storage.example.test",
		PublicURL: "https://storage.example.test",
		AccessKey: "rustfsadmin",
		SecretKey: "rustfsadmin",
		Bucket:    "kinstack",
		Region:    "us-east-1",
		UseSSL:    true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected rustfs example credentials to be rejected")
	}
}

func TestProductionRejectsExampleBreakGlassSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := AppConfig{
		Name:                "KinStack",
		Env:                 "production",
		URL:                 "https://app.example.test",
		AccessKey:           "superadmin",
		SecretKey:           "replace-with-random-secret-at-least-32-characters",
		Port:                3000,
		SessionIdleTimeout:  time.Hour,
		AuthRateLimitMax:    10,
		AuthRateLimitWindow: time.Minute,
		APIRateLimitMax:     10,
		APIRateLimitWindow:  time.Minute,
		HTTPBodyLimit:       12 << 20,
		HTTPReadTimeout:     time.Second,
		HTTPWriteTimeout:    time.Second,
		HTTPIdleTimeout:     time.Second,
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "SECRET_KEY") {
		t.Fatalf("expected SECRET_KEY placeholder error, got %v", err)
	}
}
