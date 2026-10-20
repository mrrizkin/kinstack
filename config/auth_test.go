package config

import (
	"strings"
	"testing"
)

func TestAuthConfigDefaultsEnableBuiltInMethods(t *testing.T) {
	cfg := AuthConfig{}
	cfg.Load()
	if !cfg.MagicLink.Enabled || !cfg.Passkey.Enabled || !cfg.APIToken.Enabled {
		t.Fatalf("built-in methods should default to enabled: %+v", cfg)
	}
	if cfg.OAuthEnabled() {
		t.Fatalf("providers should default to disabled")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default auth config should be valid: %v", err)
	}
}

func TestAuthConfigEnabledOIDCRequiresSettings(t *testing.T) {
	cfg := AuthConfig{OIDC: OIDCProviderConfig{Enabled: true}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_OIDC") {
		t.Fatalf("expected OIDC configuration error, got %v", err)
	}
}

func TestAuthConfigEnabledSocialRequiresCredentials(t *testing.T) {
	cfg := AuthConfig{Google: SocialProviderConfig{Enabled: true}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_GOOGLE") {
		t.Fatalf("expected Google provider error, got %v", err)
	}
}

func TestAuthConfigOAuthRequiresStateSecret(t *testing.T) {
	cfg := AuthConfig{
		OIDC: OIDCProviderConfig{
			Enabled:      true,
			ClientID:     "client",
			ClientSecret: "secret",
			AuthURL:      "https://idp.example.test/auth",
			TokenURL:     "https://idp.example.test/token",
			UserInfoURL:  "https://idp.example.test/userinfo",
		},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_STATE_SECRET") {
		t.Fatalf("expected state secret error, got %v", err)
	}
}

func TestAuthConfigAcceptsCompleteOIDC(t *testing.T) {
	cfg := AuthConfig{
		StateSecret: strings.Repeat("x", 32),
		OIDC: OIDCProviderConfig{
			Enabled:      true,
			Name:         "Acme",
			ClientID:     "client",
			ClientSecret: "secret",
			AuthURL:      "https://idp.example.test/auth",
			TokenURL:     "https://idp.example.test/token",
			UserInfoURL:  "https://idp.example.test/userinfo",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("complete OIDC config should be valid: %v", err)
	}
	if cfg.OIDC.ProviderName() != "acme" {
		t.Fatalf("unexpected provider name %q", cfg.OIDC.ProviderName())
	}
}

func TestAuthConfigSAMLRequiresSettings(t *testing.T) {
	cfg := AuthConfig{SAML: SAMLProviderConfig{Enabled: true}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_SAML") {
		t.Fatalf("expected SAML configuration error, got %v", err)
	}
}

func TestAuthConfigLoadReadsFlags(t *testing.T) {
	t.Setenv("AUTH_MAGIC_LINK_ENABLED", "false")
	t.Setenv("AUTH_API_TOKEN_ENABLED", "false")
	t.Setenv("AUTH_OIDC_ENABLED", "true")
	t.Setenv("AUTH_OIDC_CLIENT_ID", "client")
	t.Setenv("AUTH_OIDC_CLIENT_SECRET", "secret")
	t.Setenv("AUTH_OIDC_AUTH_URL", "https://idp.example.test/auth")
	t.Setenv("AUTH_OIDC_TOKEN_URL", "https://idp.example.test/token")
	t.Setenv("AUTH_OIDC_USERINFO_URL", "https://idp.example.test/userinfo")
	t.Setenv("AUTH_STATE_SECRET", strings.Repeat("y", 32))

	cfg := AuthConfig{}
	cfg.Load()
	if cfg.MagicLink.Enabled || cfg.APIToken.Enabled {
		t.Fatalf("flags should disable methods: %+v", cfg)
	}
	if !cfg.OAuthEnabled() {
		t.Fatalf("OIDC should be enabled")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("loaded auth config should be valid: %v", err)
	}
}

func TestAuthConfigTwoFactorKey(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "empty falls back to app secret", key: ""},
		{name: "strong key accepted", key: "K7p2Qz9mX4vL8sN3wR6tY1uB5cD0eF7h"},
		{name: "short key rejected", key: "too-short", wantErr: true},
		{name: "example key rejected", key: "change-me-in-production-please-32b", wantErr: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := AuthConfig{TwoFactorKey: testCase.key}
			err := cfg.Validate()
			if testCase.wantErr && err == nil {
				t.Fatal("expected a validation error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("expected a valid config, got %v", err)
			}
		})
	}
}
