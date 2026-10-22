package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
)

// DatabaseConfig selects the event repository and its connection string.
type DatabaseConfig struct {
	URL              string
	StatementTimeout time.Duration
	MaxOpenConns     int
	MaxIdleConns     int
}

func (c *DatabaseConfig) Load() {
	c.URL = os.Getenv("DATABASE_URL")
	c.StatementTimeout = assert.OK(envDuration("DB_STATEMENT_TIMEOUT", 15*time.Second))
	c.MaxOpenConns = assert.OK(envInt("DB_MAX_CONNS", 10))
	c.MaxIdleConns = assert.OK(envInt("DB_MAX_IDLE_CONNS", 5))
}

// LoadDatabase reads and validates only the database configuration for migration commands.
func LoadDatabase() DatabaseConfig {
	var cfg DatabaseConfig
	cfg.Load()
	assert.NoError(cfg.validateDatabase())
	return cfg
}

// Validate checks the database settings required by the application.
func (c DatabaseConfig) Validate() error {
	if err := c.validateDatabase(); err != nil {
		return err
	}
	if c.StatementTimeout < 0 {
		return fmt.Errorf("DB_STATEMENT_TIMEOUT must not be negative")
	}
	if c.MaxOpenConns < 0 {
		return fmt.Errorf("DB_MAX_CONNS must not be negative")
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must not be negative")
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must not exceed DB_MAX_CONNS")
	}
	dsn, err := url.Parse(c.URL)
	if err != nil || dsn.Host == "" || (dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") {
		return fmt.Errorf("DATABASE_URL must be a postgres:// or postgresql:// URL")
	}
	if isProduction() {
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
	}
	return nil
}

func (c DatabaseConfig) validateDatabase() error {
	if c.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	dsn, err := url.Parse(c.URL)
	if err != nil || dsn.Host == "" || (dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") {
		return fmt.Errorf("DATABASE_URL must be a postgres:// or postgresql:// URL")
	}
	return nil
}
