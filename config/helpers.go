package config

import (
	"fmt"
	"net/url"
	"strings"
)

const environmentProduction = "production"

func isProduction() bool {
	return strings.EqualFold(
		strings.TrimSpace(env("APP_ENV", "development")),
		environmentProduction,
	)
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

// exampleSecrets are the literal values shipped in .env.example and
// deploy/compose.yml: fine locally, never in production.
var exampleSecrets = map[string]struct{}{
	"kinstack":                {},
	"kinstack-secret":         {},
	"kinstack-local-password": {},
	"kinstack-local":          {},
	"superadmin":              {},
	"rustfsadmin":             {},
	"admin":                   {},
	"password":                {},
	"replace-with-random-secret-at-least-32-characters":                {},
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
	"minimal-32",
	"local-",
	"123456",
}

// placeholderSecret catches an empty, example, template, or trivially weak secret.
// Callers gate this behind production checks.
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
	distinct, digitsOnly := secretShape(value)
	if distinct < 4 {
		return true
	}
	if digitsOnly && len(value) < 24 {
		return true
	}
	// A short block repeated to fill the value ("abcabcabcabc") is not a secret.
	return repeatedBlock(value)
}

// secretShape counts distinct bytes and reports whether the value is only digits.
func secretShape(value string) (distinct int, digitsOnly bool) {
	seen := make(map[byte]struct{}, len(value))
	digitsOnly = true
	for index := 0; index < len(value); index++ {
		seen[value[index]] = struct{}{}
		if value[index] < '0' || value[index] > '9' {
			digitsOnly = false
		}
	}
	return len(seen), digitsOnly
}

func repeatedBlock(value string) bool {
	for size := 1; size <= len(value)/2 && size <= 8; size++ {
		if len(value)%size != 0 {
			continue
		}
		if blockRepeats(value, size) {
			return true
		}
	}
	return false
}

func blockRepeats(value string, size int) bool {
	for index := size; index < len(value); index++ {
		if value[index] != value[index%size] {
			return false
		}
	}
	return true
}
