package config

import (
	"strings"
	"testing"
)

func TestEmailConfigValidateOptionalInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg := EmailConfig{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development email config should be optional: %v", err)
	}
}

func TestEmailConfigProductionRequiresSMTP(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if err := (EmailConfig{}).Validate(); err == nil ||
		!strings.Contains(err.Error(), "SMTP host") {
		t.Fatalf("expected production SMTP error, got %v", err)
	}
}

func TestEmailConfigRejectsPartialCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg := EmailConfig{
		Host:     "smtp.example.test",
		Port:     587,
		From:     "no-reply@example.test",
		Username: "user",
	}
	if err := cfg.Validate(); err == nil ||
		!strings.Contains(err.Error(), "username and password") {
		t.Fatalf("expected partial credentials error, got %v", err)
	}
}
