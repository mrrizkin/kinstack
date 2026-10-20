// Package config loads and validates the application configuration.
package config

import "kinstack/internal/pkg/assert"

// Configuration is a single application configuration section.
//
// Load reads the section from the environment and Validate checks the loaded
// values. Config uses this contract to orchestrate all sections.
type Configuration interface {
	Load()
	Validate() error
}

// Config is the complete runtime configuration.
type Config struct {
	App         AppConfig
	DB          DatabaseConfig
	Valkey      ValkeyConfig
	Log         LogConfig
	Document    DocumentConfig
	Email       EmailConfig
	Auth        AuthConfig
	RustFS      RustFSConfig
	GlitchTip   GlitchTipConfig
	Umami       UmamiConfig
	OpenObserve OpenObserveConfig
	Security    SecurityConfig
}

// Load reads and validates every configuration section. Invalid configuration
// causes an immediate panic before the application starts.
func Load() Config {
	var cfg Config
	sections := []Configuration{
		&cfg.App,
		&cfg.DB,
		&cfg.Valkey,
		&cfg.Log,
		&cfg.Document,
		&cfg.Email,
		&cfg.Auth,
		&cfg.RustFS,
		&cfg.GlitchTip,
		&cfg.Umami,
		&cfg.OpenObserve,
		&cfg.Security,
	}
	for _, section := range sections {
		section.Load()
		assert.NoError(section.Validate())
	}
	return cfg
}
