package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"kinstack/internal/pkg/assert"
)

// MethodToggle is a boolean switch for one authentication method that needs no
// further configuration.
type MethodToggle struct {
	Enabled bool
}

func (t *MethodToggle) load(key string, fallback bool) {
	t.Enabled = assert.OK(envBool(key, fallback))
}

// OIDCProviderConfig configures the generic OpenID Connect provider used for
// modern SSO. Endpoints are supplied explicitly so any compliant IdP works.
type OIDCProviderConfig struct {
	Enabled      bool
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// SocialProviderConfig configures one OAuth 2.0 provider whose endpoints are
// bundled with the goth provider library (Google, Apple, Facebook, GitHub).
type SocialProviderConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
}

// SAMLProviderConfig configures the SAML 2.0 service provider.
type SAMLProviderConfig struct {
	Enabled        bool
	EntityID       string
	IDPMetadataURL string
	Certificate    string
	PrivateKey     string
}

// AuthConfig groups every authentication method and external identity provider.
// A method is active only when its Enabled flag is true; business logic never
// inspects the raw settings to decide availability.
type AuthConfig struct {
	StateSecret string
	// TwoFactorKey is the envelope key material for TOTP secrets. When empty the
	// application secret key is used instead.
	TwoFactorKey string

	MagicLink     MethodToggle
	PasswordReset MethodToggle
	Passkey       MethodToggle
	APIToken      MethodToggle

	OIDC     OIDCProviderConfig
	Google   SocialProviderConfig
	Apple    SocialProviderConfig
	Facebook SocialProviderConfig
	GitHub   SocialProviderConfig
	SAML     SAMLProviderConfig
}

func (c *AuthConfig) Load() {
	c.StateSecret = os.Getenv("AUTH_STATE_SECRET")
	c.TwoFactorKey = strings.TrimSpace(os.Getenv("AUTH_2FA_KEY"))

	c.MagicLink.load("AUTH_MAGIC_LINK_ENABLED", true)
	c.PasswordReset.load("AUTH_PASSWORD_RESET_ENABLED", true)
	c.Passkey.load("AUTH_PASSKEY_ENABLED", true)
	c.APIToken.load("AUTH_API_TOKEN_ENABLED", true)

	c.OIDC.Enabled = assert.OK(envBool("AUTH_OIDC_ENABLED", false))
	c.OIDC.Name = strings.TrimSpace(os.Getenv("AUTH_OIDC_NAME"))
	c.OIDC.ClientID = strings.TrimSpace(os.Getenv("AUTH_OIDC_CLIENT_ID"))
	c.OIDC.ClientSecret = os.Getenv("AUTH_OIDC_CLIENT_SECRET")
	c.OIDC.AuthURL = strings.TrimSpace(os.Getenv("AUTH_OIDC_AUTH_URL"))
	c.OIDC.TokenURL = strings.TrimSpace(os.Getenv("AUTH_OIDC_TOKEN_URL"))
	c.OIDC.UserInfoURL = strings.TrimSpace(os.Getenv("AUTH_OIDC_USERINFO_URL"))

	c.Google = loadSocial("AUTH_GOOGLE")
	c.Apple = loadSocial("AUTH_APPLE")
	c.Facebook = loadSocial("AUTH_FACEBOOK")
	c.GitHub = loadSocial("AUTH_GITHUB")

	c.SAML.Enabled = assert.OK(envBool("AUTH_SAML_ENABLED", false))
	c.SAML.EntityID = strings.TrimSpace(os.Getenv("AUTH_SAML_ENTITY_ID"))
	c.SAML.IDPMetadataURL = strings.TrimSpace(os.Getenv("AUTH_SAML_IDP_METADATA_URL"))
	c.SAML.Certificate = os.Getenv("AUTH_SAML_CERT")
	c.SAML.PrivateKey = os.Getenv("AUTH_SAML_KEY")
}

// OAuthEnabled reports whether at least one OAuth 2.0 / OIDC provider is on.
func (c AuthConfig) OAuthEnabled() bool {
	return c.OIDC.Enabled || c.Google.Enabled || c.Apple.Enabled ||
		c.Facebook.Enabled || c.GitHub.Enabled
}

// OIDCProviderName returns a URL-safe slug for the generic OIDC provider.
func (c OIDCProviderConfig) ProviderName() string {
	return slug(c.Name, "oidc")
}

// Validate enforces the fail-fast rule: an enabled method or provider must be
// fully and validly configured. A missing setting is a developer error, so the
// process panics at startup instead of degrading at runtime.
func (c AuthConfig) Validate() error {
	if err := c.OIDC.validate(); err != nil {
		return err
	}
	for name, provider := range map[string]SocialProviderConfig{
		"Google":   c.Google,
		"Apple":    c.Apple,
		"Facebook": c.Facebook,
		"GitHub":   c.GitHub,
	} {
		if err := provider.validate(name); err != nil {
			return err
		}
	}
	if err := c.SAML.validate(); err != nil {
		return err
	}
	if c.OAuthEnabled() && len([]byte(c.StateSecret)) < 32 {
		return fmt.Errorf(
			"AUTH_STATE_SECRET must be at least 32 bytes when an OAuth provider is enabled",
		)
	}
	if c.TwoFactorKey != "" && (len([]byte(c.TwoFactorKey)) < 32 ||
		placeholderSecret(c.TwoFactorKey)) {
		return fmt.Errorf(
			"AUTH_2FA_KEY must be at least 32 bytes and not a placeholder value when set",
		)
	}
	return nil
}

func (c OIDCProviderConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	if c.ClientID == "" || c.ClientSecret == "" || c.AuthURL == "" ||
		c.TokenURL == "" || c.UserInfoURL == "" {
		return fmt.Errorf(
			"AUTH_OIDC_ENABLED is on but incomplete: set AUTH_OIDC_CLIENT_ID, " +
				"AUTH_OIDC_CLIENT_SECRET, AUTH_OIDC_AUTH_URL, AUTH_OIDC_TOKEN_URL, " +
				"AUTH_OIDC_USERINFO_URL",
		)
	}
	for _, raw := range []string{c.AuthURL, c.TokenURL, c.UserInfoURL} {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("AUTH_OIDC_* URLs must be valid https URLs")
		}
	}
	return nil
}

func (c SocialProviderConfig) validate(name string) error {
	if !c.Enabled {
		return nil
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return fmt.Errorf(
			"AUTH_%s_ENABLED is on but AUTH_%s_CLIENT_ID / AUTH_%s_CLIENT_SECRET are empty",
			strings.ToUpper(name), strings.ToUpper(name), strings.ToUpper(name),
		)
	}
	return nil
}

func (c SAMLProviderConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	switch {
	case c.EntityID == "", c.IDPMetadataURL == "", c.Certificate == "", c.PrivateKey == "":
		return fmt.Errorf(
			"AUTH_SAML_ENABLED is on but incomplete: set AUTH_SAML_ENTITY_ID, " +
				"AUTH_SAML_IDP_METADATA_URL, AUTH_SAML_CERT, AUTH_SAML_KEY",
		)
	}
	parsed, err := url.Parse(c.IDPMetadataURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("AUTH_SAML_IDP_METADATA_URL must be a valid https URL")
	}
	return nil
}

func loadSocial(prefix string) SocialProviderConfig {
	return SocialProviderConfig{
		Enabled:      assert.OK(envBool(prefix+"_ENABLED", false)),
		ClientID:     strings.TrimSpace(os.Getenv(prefix + "_CLIENT_ID")),
		ClientSecret: os.Getenv(prefix + "_CLIENT_SECRET"),
	}
}

func slug(value, fallback string) string {
	filtered := strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z',
			character >= '0' && character <= '9',
			character == '-', character == '_':
			return character
		case character >= 'A' && character <= 'Z':
			return character + ('a' - 'A')
		}
		return -1
	}, strings.TrimSpace(value))
	if filtered == "" {
		return fallback
	}
	return filtered
}
