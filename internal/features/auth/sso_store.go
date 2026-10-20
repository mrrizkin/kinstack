package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

// ssoStateTTL bounds how long an SSO transaction may sit between the
// authorization redirect and the callback. It matches the sealed state TTL.
const ssoStateTTL = 10 * time.Minute

// sso prefixes keep the three kinds of SSO transaction data apart.
const (
	ssoOAuthPrefix    = "sso:oauth:"
	ssoVerifierPrefix = "sso:verifier:"
	ssoSAMLRequestID  = "sso:saml:"
)

// ssoStore keeps short-lived SSO transaction data in the shared server-side
// storage instead of the browser session. This is required for providers that
// post the authorization response back cross-site (Apple form_post, SAML
// HTTP-POST): the browser does not send the SameSite=Lax session cookie with
// such a request, so session-backed state would always be missing.
type ssoStore struct {
	storage fiber.Storage
	prefix  string
}

// newSSOStore builds a namespaced view over the shared storage.
func newSSOStore(storage fiber.Storage, prefix string) ssoStore {
	return ssoStore{storage: storage, prefix: prefix}
}

// Save stores one value for the SSO state TTL.
func (s ssoStore) Save(key, value string) error {
	if s.storage == nil || key == "" {
		return errSSOStorageUnavailable
	}
	return s.storage.Set(s.prefix+key, []byte(value), ssoStateTTL)
}

// Load reads a value without removing it.
func (s ssoStore) Load(key string) (string, bool) {
	if s.storage == nil || key == "" {
		return "", false
	}
	raw, err := s.storage.Get(s.prefix + key)
	if err != nil || len(raw) == 0 {
		return "", false
	}
	return string(raw), true
}

// Consume reads a value and removes it, making the transaction single use.
func (s ssoStore) Consume(key string) (string, bool) {
	value, ok := s.Load(key)
	if ok {
		s.Delete(key)
	}
	return value, ok
}

// Delete removes a value.
func (s ssoStore) Delete(key string) {
	if s.storage == nil || key == "" {
		return
	}
	_ = s.storage.Delete(s.prefix + key)
}
