package config

import (
	"fmt"
	"os"

	"kinstack/internal/pkg/assert"
)

// ValkeyConfig configures the shared session/cache backend.
type ValkeyConfig struct {
	Host     string
	Port     int
	Password string
	Database int
}

func (c *ValkeyConfig) Load() {
	c.Port = assert.OK(envInt("VALKEY_PORT", 6379))
	c.Database = assert.OK(envInt("VALKEY_DATABASE", 0))
	c.Host = env("VALKEY_HOST", "localhost")
	c.Password = os.Getenv("VALKEY_PASSWORD")
}

// Validate checks Valkey connection settings.
func (c ValkeyConfig) Validate() error {
	if c.Host == "" || c.Port < 1 || c.Port > 65535 || c.Database < 0 {
		return fmt.Errorf("valkey host, port, and database are invalid")
	}
	if isProduction() && placeholderSecret(c.Password) {
		return fmt.Errorf("VALKEY_PASSWORD must be a non-placeholder production secret")
	}
	return nil
}
