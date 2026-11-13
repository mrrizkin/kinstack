package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
)

type DatabaseConfig struct {
	URL              string
	StatementTimeout time.Duration
	OpenConnsMax     int
	IdleConnsMax     int
}

func (c *DatabaseConfig) Load() {
	c.URL = os.Getenv("DATABASE_URL")
	c.StatementTimeout = assert.OK(envDuration("DB_STATEMENT_TIMEOUT", 15*time.Second))
	c.OpenConnsMax = assert.OK(envInt("DB_MAX_CONNS", 10))
	c.IdleConnsMax = assert.OK(envInt("DB_MAX_IDLE_CONNS", 5))
}

// LoadDatabase reads and validates only the database configuration for migration commands.
func LoadDatabase() DatabaseConfig {
	var cfg DatabaseConfig
	cfg.Load()
	assert.NoError(cfg.validateDatabase())
	return cfg
}

func (c DatabaseConfig) Validate() error {
	dsn, err := parseDatabaseURL(c.URL)
	if err != nil {
		return err
	}
	if err := c.validatePool(); err != nil {
		return err
	}
	if !isProduction() {
		return nil
	}
	return requireProductionDSN(dsn)
}

// validatePool rejects pool sizes that cannot be satisfied. The idle ceiling is
// checked against open here because the driver would only find out at runtime.
func (c DatabaseConfig) validatePool() error {
	if c.StatementTimeout < 0 {
		return fmt.Errorf("DB_STATEMENT_TIMEOUT must not be negative")
	}
	if c.OpenConnsMax < 0 {
		return fmt.Errorf("DB_MAX_CONNS must not be negative")
	}
	if c.IdleConnsMax < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must not be negative")
	}
	if c.OpenConnsMax > 0 && c.IdleConnsMax > c.OpenConnsMax {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must not exceed DB_MAX_CONNS")
	}
	return nil
}

// requireProductionDSN states the three things a production connection string
// may not omit: a user, a real password, and encryption in transit.
func requireProductionDSN(dsn *url.URL) error {
	if dsn.User == nil || dsn.User.Username() == "" {
		return fmt.Errorf("DATABASE_URL must include a database username")
	}
	password, hasPassword := dsn.User.Password()
	if !hasPassword || placeholderSecret(password) {
		return fmt.Errorf("DATABASE_URL must include a non-placeholder database password")
	}
	sslMode := strings.ToLower(dsn.Query().Get("sslmode"))
	if sslMode == "" || sslMode == "disable" {
		return fmt.Errorf("DATABASE_URL must set sslmode to an encrypted mode in production")
	}
	return nil
}

func (c DatabaseConfig) validateDatabase() error {
	_, err := parseDatabaseURL(c.URL)
	return err
}

// parseDatabaseURL is the single definition of an acceptable DATABASE_URL, so
// the startup assertion and Validate cannot disagree about it.
func parseDatabaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	dsn, err := url.Parse(raw)
	if err != nil || dsn.Host == "" ||
		(dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") {
		return nil, fmt.Errorf("DATABASE_URL must be a postgres:// or postgresql:// URL")
	}
	return dsn, nil
}
