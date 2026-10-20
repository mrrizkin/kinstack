// Package totp implements RFC 6238 time-based one-time passwords with the
// SHA-1 / 6-digit / 30-second profile used by authenticator apps.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// Period is the time step in seconds.
	Period = 30
	// Digits is the number of decimal digits in a generated code.
	Digits = 6
	// secretBytes is the raw entropy stored per secret.
	secretBytes = 20
)

var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecret returns a new random base32 secret suitable for storage.
func GenerateSecret() (string, error) {
	buffer := make([]byte, secretBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read random secret: %w", err)
	}
	return encoding.EncodeToString(buffer), nil
}

// ProvisioningURI builds the otpauth:// URI an authenticator app can scan. The
// issuer is prefixed to the label as recommended by the Key URI format.
func ProvisioningURI(secret, account, issuer string) string {
	account = strings.TrimSpace(account)
	issuer = strings.TrimSpace(issuer)
	label := account
	if issuer != "" {
		label = issuer + ":" + account
	}
	query := url.Values{}
	query.Set("secret", strings.ToUpper(strings.TrimSpace(secret)))
	query.Set("algorithm", "SHA1")
	query.Set("digits", fmt.Sprintf("%d", Digits))
	query.Set("period", fmt.Sprintf("%d", Period))
	if issuer != "" {
		query.Set("issuer", issuer)
	}
	return "otpauth://totp/" + url.PathEscape(label) + "?" + query.Encode()
}

// GenerateCode returns the code valid for the step containing at.
func GenerateCode(secret string, at time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	counter := uint64(at.UTC().Unix() / Period)
	return codeForCounter(key, counter), nil
}

// Validate reports whether code matches the current step or one step on either
// side, absorbing small clock differences between server and phone.
func Validate(secret, code string, at time.Time) bool {
	_, ok := ValidateStep(secret, code, at)
	return ok
}

// ValidateStep returns the time step a code matched, so callers can reject a
// code that was already used for that step.
func ValidateStep(secret, code string, at time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != Digits {
		return 0, false
	}
	key, err := decodeSecret(secret)
	if err != nil {
		return 0, false
	}
	counter := uint64(at.UTC().Unix() / Period)
	for _, delta := range []int64{-1, 0, 1} {
		adjusted := int64(counter) + delta
		if adjusted < 0 {
			continue
		}
		expected := codeForCounter(key, uint64(adjusted))
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return adjusted, true
		}
	}
	return 0, false
}

func decodeSecret(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	key, err := encoding.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("decode secret: %w", err)
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("decode secret: empty secret")
	}
	return key, nil
}

func codeForCounter(key []byte, counter uint64) string {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buffer)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%0*d", Digits, value%1_000_000)
}
