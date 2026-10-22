package config

import (
	"fmt"
	"os"
	"strings"

	"kinstack/internal/pkg/assert"
)

// EmailConfig configures the outbound SMTP relay used by application emails.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	StartTLS bool
}

func (c *EmailConfig) Load() {
	c.Port = assert.OK(envInt("SMTP_PORT", 587))
	c.StartTLS = assert.OK(envBool("SMTP_STARTTLS", true))
	c.Host = strings.TrimSpace(os.Getenv("SMTP_HOST"))
	c.Username = os.Getenv("SMTP_USERNAME")
	c.Password = os.Getenv("SMTP_PASSWORD")
	c.From = os.Getenv("SMTP_FROM")
}

// Validate checks SMTP settings when email delivery is configured.
func (c EmailConfig) Validate() error {
	if c.Host != "" {
		if c.Port < 1 || c.Port > 65535 || c.From == "" {
			return fmt.Errorf(
				"SMTP host, port, and from address are required when SMTP is configured",
			)
		}
		if (c.Username == "") != (c.Password == "") {
			return fmt.Errorf("SMTP username and password must be provided together")
		}
	}
	if isProduction() {
		if c.Host == "" || c.Port < 1 || c.Port > 65535 || c.From == "" {
			return fmt.Errorf("SMTP host, port, and from address are required in production")
		}
		if !c.StartTLS {
			return fmt.Errorf("SMTP_STARTTLS must be enabled in production")
		}
		if (c.Username == "") != (c.Password == "") || placeholderSecret(c.Password) {
			return fmt.Errorf(
				"SMTP credentials must be provided together and must not be placeholders",
			)
		}
	}
	return nil
}
