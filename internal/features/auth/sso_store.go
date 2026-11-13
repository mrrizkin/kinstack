package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

const ssoStateTTL = 10 * time.Minute

const (
	ssoOAuthPrefix    = "sso:oauth:"
	ssoVerifierPrefix = "sso:verifier:"
	ssoSAMLRequestID  = "sso:saml:"
)

// ssoStore keeps SSO transaction data in shared storage, not the browser session:
// cross-site POST callbacks (Apple form_post, SAML HTTP-POST) carry no SameSite=Lax
// cookie.
type ssoStore struct {
	storage fiber.Storage
	prefix  string
}

func newSSOStore(storage fiber.Storage, prefix string) ssoStore {
	return ssoStore{storage: storage, prefix: prefix}
}

func (s ssoStore) Save(key, value string) error {
	if s.storage == nil || key == "" {
		return errSSOStorageUnavailable
	}
	return s.storage.Set(s.prefix+key, []byte(value), ssoStateTTL)
}

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

func (s ssoStore) Consume(key string) (string, bool) {
	value, ok := s.Load(key)
	if ok {
		s.Delete(key)
	}
	return value, ok
}

func (s ssoStore) Delete(key string) {
	if s.storage == nil || key == "" {
		return
	}
	_ = s.storage.Delete(s.prefix + key)
}
