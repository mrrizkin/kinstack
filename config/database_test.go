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

func TestDatabaseConfigPoolDefaultsAndValidation(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/kinstack")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	var cfg DatabaseConfig
	cfg.Load()
	if cfg.MaxOpenConns != 10 || cfg.MaxIdleConns != 5 {
		t.Fatalf("unexpected pool defaults: %+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("defaults must validate: %v", err)
	}

	cfg.MaxIdleConns = 12
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "DB_MAX_IDLE_CONNS") {
		t.Fatalf("expected idle pool validation error, got %v", err)
	}

	cfg.MaxIdleConns = 0
	cfg.MaxOpenConns = -1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "DB_MAX_CONNS") {
		t.Fatalf("expected open pool validation error, got %v", err)
	}

	cfg.MaxOpenConns = 10
	cfg.MaxIdleConns = -1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "DB_MAX_IDLE_CONNS") {
		t.Fatalf("expected idle pool validation error, got %v", err)
	}
}
