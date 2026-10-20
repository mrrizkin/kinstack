package twofactor

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// recoveryCodeShape matches the "XXXX-XXXX" codes handed to users, with the
// separator optional so pasted codes still verify.
var recoveryCodeShape = regexp.MustCompile(`^[A-Za-z0-9]{4}-?[A-Za-z0-9]{4}$`)

// normalizeRecoveryCode removes separators and upper-cases a code so storage and
// verification always agree on the canonical form.
func normalizeRecoveryCode(code string) string {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	return strings.NewReplacer("-", "", " ", "").Replace(normalized)
}

// legacyRecoveryDigest reproduces the unsalted SHA-256 hash used before
// recovery codes moved to Argon2id, so existing codes keep working once.
func legacyRecoveryDigest(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// isLegacyRecoveryHash reports whether a stored hash uses the old format.
func isLegacyRecoveryHash(hash string) bool {
	if len(hash) != legacyHashLength {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}
