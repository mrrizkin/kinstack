package config

import (
	"strings"
	"testing"
	"time"
)

func TestDatabaseConfigValidateProductionTransport(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := DatabaseConfig{
		URL: "postgres://app:" + generatedSecret(t) + "@db/kinstack?sslmode=disable",
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "sslmode") {
		t.Fatalf("expected sslmode validation error, got %v", err)
	}
}

func TestDatabaseConfigLoadDoesNotDefaultURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	var cfg DatabaseConfig
	cfg.Load()
	if cfg.URL != "" {
		t.Fatalf("DATABASE_URL must not have a default: %q", cfg.URL)
	}
}

func TestLoadDatabaseOnlyRequiresDatabaseConfiguration(t *testing.T) {
	t.Setenv("DATABASE_MODE", "postgres")
	t.Setenv("DATABASE_URL", "postgres://localhost/kinstack")
	cfg := LoadDatabase()
	if cfg.URL != "postgres://localhost/kinstack" {
		t.Fatalf("unexpected database config: %#v", cfg)
	}
}

func TestDatabaseConfigLoadsStatementTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/kinstack")
	t.Setenv("DB_STATEMENT_TIMEOUT", "7s")
	var cfg DatabaseConfig
	cfg.Load()
	if cfg.StatementTimeout != 7*time.Second {
		t.Fatalf("unexpected statement timeout: %s", cfg.StatementTimeout)
	}
}
