package config

import (
	"fmt"
	"strings"

	"kinstack/internal/pkg/assert"
)

// UmamiConfig configures the privacy-friendly Umami analytics tracker. The
// tracker script is injected into the document shell only when enabled.
type UmamiConfig struct {
	Enabled   bool
	ScriptURL string
	WebsiteID string
	Domains   string
	AutoTrack bool
}

// Load reads analytics settings from the environment.
func (c *UmamiConfig) Load() {
	c.Enabled = assert.OK(envBool("UMAMI_ENABLED", false))
	c.ScriptURL = strings.TrimSpace(env("UMAMI_SCRIPT_URL", ""))
	c.WebsiteID = strings.TrimSpace(env("UMAMI_WEBSITE_ID", ""))
	c.Domains = strings.TrimSpace(env("UMAMI_DOMAINS", ""))
	c.AutoTrack = assert.OK(envBool("UMAMI_AUTO_TRACK", true))
}

// Validate checks analytics settings. Disabled analytics ignore every setting.
func (c UmamiConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := validateURL("UMAMI_SCRIPT_URL", c.ScriptURL, isProduction()); err != nil {
		return err
	}
	if c.WebsiteID == "" {
		return fmt.Errorf("UMAMI_WEBSITE_ID is required when UMAMI_ENABLED is true")
	}
	return nil
}
