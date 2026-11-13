// Package i18n is the response-message catalog. It is keyed, so more locales can
// be added without changing call sites.
package i18n

import (
	"fmt"
	"strings"
)

type Locale string

const (
	// English is the default and currently the only shipped locale.
	English Locale = "en"
)

func Default() Locale {
	return English
}

// Resolve normalizes an Accept-Language header. Only English is available, so
// every request resolves to English while the API remains future-proof.
func Resolve(raw string) Locale {
	for _, part := range strings.Split(raw, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		if strings.HasPrefix(tag, "en") {
			return English
		}
	}
	return Default()
}

func T(key string, args ...any) string {
	return TLocale(Default(), key, args...)
}

// TLocale translates a catalog key for an explicit locale. Unknown locales and
// unknown keys fall back to the key so a missing translation is obvious in
// development without breaking a response.
func TLocale(locale Locale, key string, args ...any) string {
	messages, ok := catalogs[locale]
	if !ok {
		messages = catalogs[Default()]
	}
	message, ok := messages[key]
	if !ok {
		message = key
	}
	if len(args) == 0 {
		return message
	}
	return fmt.Sprintf(message, args...)
}
