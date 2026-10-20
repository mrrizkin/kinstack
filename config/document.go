package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
)

// defaultDocumentTimeout bounds one PDF rendering request.
const defaultDocumentTimeout = 30 * time.Second

// DocumentConfig contains the report document rendering settings. Rendering is
// delegated to a Gotenberg service; when no URL is configured the report PDF
// export stays disabled and the application runs without it.
type DocumentConfig struct {
	GotenbergURL string
	Timeout      time.Duration
}

// Load reads document rendering settings from the environment.
func (c *DocumentConfig) Load() {
	c.Timeout = assert.OK(envDuration("DOCUMENT_TIMEOUT", defaultDocumentTimeout))
	c.GotenbergURL = strings.TrimRight(strings.TrimSpace(os.Getenv("GOTENBERG_URL")), "/")
}

// Validate checks document rendering settings that do not depend on another
// section.
func (c DocumentConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("DOCUMENT_TIMEOUT harus bernilai positif")
	}
	if c.GotenbergURL == "" {
		return nil
	}
	return validateURL("GOTENBERG_URL", c.GotenbergURL, false)
}

// PDFEnabled reports whether report PDF rendering is configured.
func (c DocumentConfig) PDFEnabled() bool {
	return c.GotenbergURL != ""
}
