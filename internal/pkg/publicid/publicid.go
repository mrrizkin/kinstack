// Package publicid provides the single application boundary for generated public IDs.
package publicid

import (
	"kinstack/internal/pkg/assert"

	nanoid "github.com/matoous/go-nanoid/v2"
)

const defaultLength = 21

// New returns a cryptographically secure, pure 21-character Nano ID.
// Public IDs carry no entity-specific semantic prefix.
func New() (string, error) {
	return nanoid.New(defaultLength)
}

// MustNew returns a cryptographically secure public ID or panics if generation fails.
// It is intended for trusted setup code and tests where generation failure is unrecoverable.
func MustNew() string {
	return assert.OK(New())
}

// IsValid reports whether value is a standard 21-character Nano ID using the
// alphabet emitted by this package.
func IsValid(value string) bool {
	if len(value) != defaultLength {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}
