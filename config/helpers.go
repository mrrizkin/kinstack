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

func placeholderSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "",
		"kinstack",
		"kinstack-secret",
		"change-me",
		"change-me-in-production",
		"replace-me",
		"replace-with-a-secret",
		"replace-with-local-password",
		"your-secret",
		"ci-only-password":
		return true
	default:
		return strings.Contains(normalized, "replace_with") ||
			strings.Contains(normalized, "change-me") ||
			strings.Contains(normalized, "your-")
	}
}
