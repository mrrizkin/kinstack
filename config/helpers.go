package config

import (
	"fmt"
	"net/url"
	"strings"
)

func isProduction() bool {
	return strings.EqualFold(strings.TrimSpace(env("APP_ENV", "development")), "production")
}

func validateURL(name, raw string, requireHTTPS bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", name)
	}
	if requireHTTPS && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use HTTPS in production", name)
	}
	return nil
}

// exampleSecrets are the literal credential values shipped in .env.example and
// deploy/compose.yml. They are fine for local development but must never reach
// production: a deployment that copies the example environment file would
// otherwise run with published credentials.
var exampleSecrets = map[string]struct{}{
	"kinstack":                {},
	"kinstack-secret":         {},
	"kinstack-local-password": {},
	"kinstack-local":          {},
	"superadmin":              {},
	"rustfsadmin":             {},
	"admin":                   {},
	"password":                {},
	"ganti-dengan-secret-acak-minimal-32-karakter":                     {},
	"local-glitchtip-secret-change-me":                                 {},
	"local-umami-secret-change-me":                                     {},
	"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef": {},
	"kinstack!2026": {},
}

// placeholderMarkers catch values that are obviously derived from the example
// file or from a template, even when they are not exact matches.
var placeholderMarkers = []string{
	"change-me",
	"changeme",
	"replace-me",
	"replace_with",
	"replace-with",
	"your-secret",
	"your-password",
	"example",
	"sample",
	"placeholder",
	"ganti-dengan",
	"minimal-32",
	"local-",
	"123456",
}

// placeholderSecret reports whether value is an empty, example, template, or
// trivially weak secret. Callers gate this behind production checks so local
// development can keep using the documented defaults.
func placeholderSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return true
	}
	if _, ok := exampleSecrets[normalized]; ok {
		return true
	}
	for _, marker := range placeholderMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return lowEntropySecret(normalized)
}

// lowEntropySecret rejects secrets that are repetitions or short numeric
// sequences. It intentionally avoids character-class rules so long passphrases
// and generated base64 values are accepted.
func lowEntropySecret(value string) bool {
	if len(value) < 8 {
		return true
	}
	distinct := make(map[byte]struct{}, len(value))
	digitsOnly := true
	for i := 0; i < len(value); i++ {
		distinct[value[i]] = struct{}{}
		if value[i] < '0' || value[i] > '9' {
			digitsOnly = false
		}
	}
	if len(distinct) < 4 {
		return true
	}
	if digitsOnly && len(value) < 24 {
		return true
	}
	// Detect a short block repeated to fill the value ("abcabcabcabc").
	for size := 1; size <= len(value)/2 && size <= 8; size++ {
		if len(value)%size != 0 {
			continue
		}
		repeated := true
		for i := size; i < len(value); i++ {
			if value[i] != value[i%size] {
				repeated = false
				break
			}
		}
		if repeated {
			return true
		}
	}
	return false
}
