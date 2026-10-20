package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
)

// minSecretKeyLength is the minimum accepted super admin secret length.
const minSecretKeyLength = 32

// AppConfig contains HTTP application settings.
type AppConfig struct {
	Name                string
	Env                 string
	URL                 string
	Port                int
	SessionIdleTimeout  time.Duration
	AuthRateLimitMax    int
	AuthRateLimitWindow time.Duration
	AuthLockoutMax      int
	AuthLockoutWindow   time.Duration
	APIRateLimitMax     int
	APIRateLimitWindow  time.Duration
	HTTPBodyLimit       int64
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	AccessKey           string
	SecretKey           string
}

// Load reads application settings from the environment.
func (c *AppConfig) Load() {
	c.Port = assert.OK(envInt("APP_PORT", 3000))
	c.SessionIdleTimeout = assert.OK(envDuration("SESSION_IDLE_TIMEOUT", 24*time.Hour))
	c.AuthRateLimitMax = assert.OK(envInt("AUTH_RATE_LIMIT_MAX", 10))
	c.AuthRateLimitWindow = assert.OK(envDuration("AUTH_RATE_LIMIT_WINDOW", time.Minute))
	c.AuthLockoutMax = assert.OK(envInt("AUTH_LOCKOUT_MAX", 5))
	c.AuthLockoutWindow = assert.OK(envDuration("AUTH_LOCKOUT_WINDOW", 15*time.Minute))
	c.APIRateLimitMax = assert.OK(envInt("API_RATE_LIMIT_MAX", 240))
	c.APIRateLimitWindow = assert.OK(envDuration("API_RATE_LIMIT_WINDOW", time.Minute))
	c.HTTPBodyLimit = int64(assert.OK(envInt("APP_BODY_LIMIT_MB", 12))) * 1024 * 1024
	c.HTTPReadTimeout = assert.OK(envDuration("HTTP_READ_TIMEOUT", 15*time.Second))
	c.HTTPWriteTimeout = assert.OK(envDuration("HTTP_WRITE_TIMEOUT", 30*time.Second))
	c.HTTPIdleTimeout = assert.OK(envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second))

	c.Name = env("APP_NAME", "KinStack")
	c.Env = strings.ToLower(strings.TrimSpace(env("APP_ENV", "development")))
	c.URL = env("APP_URL", "http://localhost:3000")
	c.AccessKey = strings.TrimSpace(os.Getenv("ACCESS_KEY"))
	c.SecretKey = os.Getenv("SECRET_KEY")
}

// Validate checks application settings that do not depend on another section.
func (c AppConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("APP_NAME is required")
	}
	switch c.Env {
	case "development", "test", "staging", "production":
	default:
		return fmt.Errorf("APP_ENV must be one of development, test, staging, or production")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("APP_PORT must be between 1 and 65535")
	}
	if c.SessionIdleTimeout <= 0 {
		return fmt.Errorf("SESSION_IDLE_TIMEOUT must be positive")
	}
	if c.AuthRateLimitMax < 1 || c.AuthRateLimitWindow <= 0 {
		return fmt.Errorf("auth rate limit settings must be positive")
	}
	if c.AuthLockoutMax < 0 || (c.AuthLockoutMax > 0 && c.AuthLockoutWindow <= 0) {
		return fmt.Errorf("AUTH_LOCKOUT_MAX dan AUTH_LOCKOUT_WINDOW tidak valid")
	}
	if c.APIRateLimitMax < 1 || c.APIRateLimitWindow <= 0 {
		return fmt.Errorf("API_RATE_LIMIT_MAX dan API_RATE_LIMIT_WINDOW harus positif")
	}
	if c.HTTPBodyLimit < 1 {
		return fmt.Errorf("APP_BODY_LIMIT_MB harus positif")
	}
	if c.HTTPReadTimeout <= 0 || c.HTTPWriteTimeout <= 0 || c.HTTPIdleTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_TIMEOUT, HTTP_WRITE_TIMEOUT, dan HTTP_IDLE_TIMEOUT harus positif")
	}
	if err := validateURL("APP_URL", c.URL, false); err != nil {
		return err
	}
	if (c.AccessKey == "") != (c.SecretKey == "") {
		return fmt.Errorf("ACCESS_KEY dan SECRET_KEY harus diisi bersamaan")
	}
	if c.SecretKey != "" && len([]byte(c.SecretKey)) < minSecretKeyLength {
		return fmt.Errorf("SECRET_KEY minimal %d karakter", minSecretKeyLength)
	}
	if c.Env == "production" {
		if err := validateURL("APP_URL", c.URL, true); err != nil {
			return err
		}
		if c.AccessKey == "" {
			return fmt.Errorf("ACCESS_KEY dan SECRET_KEY wajib diisi di production")
		}
		if placeholderSecret(c.SecretKey) {
			return fmt.Errorf("SECRET_KEY harus nilai rahasia yang tidak placeholder di production")
		}
	}
	return nil
}
