package config

import (
	"fmt"
	"strings"

	"kinstack/internal/pkg/assert"
)

// GlitchTipConfig configures the Sentry-compatible error tracker. GlitchTip is
// wire-compatible with the Sentry SDK, so any Sentry-compatible DSN works.
type GlitchTipConfig struct {
	Enabled          bool
	DSN              string
	Environment      string
	Release          string
	TracesSampleRate float64
	Debug            bool
}

func (c *GlitchTipConfig) Load() {
	c.DSN = strings.TrimSpace(env("GLITCHTIP_DSN", ""))
	c.Enabled = c.DSN != ""
	c.Environment = strings.TrimSpace(env("GLITCHTIP_ENVIRONMENT", env("APP_ENV", "development")))
	c.Release = strings.TrimSpace(env("GLITCHTIP_RELEASE", ""))
	c.TracesSampleRate = assert.OK(envFloat("GLITCHTIP_TRACES_SAMPLE_RATE", 0))
	c.Debug = assert.OK(envBool("GLITCHTIP_DEBUG", false))
}

// Validate checks error tracking settings. An empty DSN disables the feature.
func (c GlitchTipConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := validateURL("GLITCHTIP_DSN", c.DSN, isProduction()); err != nil {
		return err
	}
	if c.TracesSampleRate < 0 || c.TracesSampleRate > 1 {
		return fmt.Errorf("GLITCHTIP_TRACES_SAMPLE_RATE must be between 0 and 1")
	}
	return nil
}
