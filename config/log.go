package config

import (
	"fmt"

	"kinstack/internal/pkg/assert"
)

// LogConfig controls structured logger output and rolling-file retention.
type LogConfig struct {
	Level     string
	Console   bool
	File      bool
	Dir       string
	Name      string
	MaxBackup int
	MaxSize   int
	MaxAge    int
}

// Load reads logging settings from the environment.
func (c *LogConfig) Load() {
	c.Console = assert.OK(envBool("LOG_CONSOLE", true))
	c.File = assert.OK(envBool("LOG_FILE", true))
	c.MaxBackup = assert.OK(envInt("LOG_MAX_BACKUP", 20))
	c.MaxSize = assert.OK(envInt("LOG_MAX_SIZE", 50))
	c.MaxAge = assert.OK(envInt("LOG_MAX_AGE", 7))
	c.Level = env("LOG_LEVEL", "debug")
	c.Dir = env("LOG_DIR", "storage/logs")
	c.Name = env("LOG_NAME", "kinstack")
}

// Validate checks logging settings. A zero LogConfig is accepted for callers
// that construct a partial Config directly in tests.
func (c LogConfig) Validate() error {
	if c.Level == "" && c.Dir == "" && c.Name == "" && c.MaxBackup == 0 && c.MaxSize == 0 &&
		c.MaxAge == 0 {
		return nil
	}
	if c.Level == "" {
		return fmt.Errorf("LOG_LEVEL is required")
	}
	if c.MaxBackup < 0 || c.MaxSize < 0 || c.MaxAge < 0 {
		return fmt.Errorf("log retention settings must be non-negative")
	}
	if c.File && (c.Dir == "" || c.Name == "") {
		return fmt.Errorf("LOG_DIR and LOG_NAME are required when file logging is enabled")
	}
	return nil
}
