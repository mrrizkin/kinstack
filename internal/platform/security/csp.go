// Package security provides HTTP hardening middleware.
package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// NonceLocal is the Fiber locals key holding the per-request CSP nonce.
const NonceLocal = "csp_nonce"

// CSPConfig describes the policy sources. Origins are added verbatim, so
// callers should pass values such as "https://umami.example.com".
type CSPConfig struct {
	Enabled        bool
	ReportOnly     bool
	ReportURI      string
	ScriptSources  []string
	ConnectSources []string
	ImageSources   []string
}

// CSP returns middleware that generates a nonce, stores it in Fiber locals, and
// writes the Content-Security-Policy header. When disabled it is a no-op.
func CSP(cfg CSPConfig) fiber.Handler {
	if !cfg.Enabled {
		return func(c fiber.Ctx) error { return c.Next() }
	}

	return func(c fiber.Ctx) error {
		nonce, err := newNonce()
		if err != nil {
			return fmt.Errorf("generate CSP nonce: %w", err)
		}
		c.Locals(NonceLocal, nonce)

		header := "Content-Security-Policy"
		if cfg.ReportOnly {
			header += "-Report-Only"
		}
		c.Set(header, policy(nonce, cfg))
		return c.Next()
	}
}

// Nonce returns the nonce generated for the current request, if any.
func Nonce(c fiber.Ctx) string {
	nonce, _ := c.Locals(NonceLocal).(string)
	return nonce
}

// Origin extracts the scheme and host from an absolute URL. It returns an empty
// string for invalid or relative values.
func Origin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func policy(nonce string, cfg CSPConfig) string {
	script := dedupe(append([]string{"'self'", "'nonce-" + nonce + "'"}, cfg.ScriptSources...))
	connect := dedupe(append([]string{"'self'"}, cfg.ConnectSources...))
	image := dedupe(append([]string{"'self'", "data:", "blob:"}, cfg.ImageSources...))

	directives := []string{
		"default-src 'self'",
		"base-uri 'self'",
		"object-src 'none'",
		"frame-ancestors 'self'",
		"form-action 'self'",
		"script-src " + strings.Join(script, " "),
		"style-src 'self' 'unsafe-inline'",
		"img-src " + strings.Join(image, " "),
		"font-src 'self' data:",
		"connect-src " + strings.Join(connect, " "),
		"worker-src 'self' blob:",
		"manifest-src 'self'",
	}
	if uri := strings.TrimSpace(cfg.ReportURI); uri != "" {
		directives = append(directives, "report-uri "+uri)
	}
	return strings.Join(directives, "; ")
}

func newNonce() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(raw), nil
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
