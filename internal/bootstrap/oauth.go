package bootstrap

import (
	"kinstack/config"
	platformoauth "kinstack/internal/platform/oauth"

	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

// oauthProviders builds the enabled SSO providers. Config validation has already
// guaranteed that every enabled provider is complete, so composition only reads
// the flags.
func oauthProviders(cfg config.Config, redirectURL string) []platformoauth.Provider {
	providers := make([]platformoauth.Provider, 0, 5)

	if cfg.Auth.OIDC.Enabled {
		label := cfg.Auth.OIDC.Name
		if label == "" {
			label = "SSO"
		}
		providers = append(providers, &platformoauth.HTTPProvider{
			ProviderName: cfg.Auth.OIDC.ProviderName(),
			DisplayLabel: label,
			ClientID:     cfg.Auth.OIDC.ClientID,
			ClientSecret: cfg.Auth.OIDC.ClientSecret,
			AuthURL:      cfg.Auth.OIDC.AuthURL,
			TokenURL:     cfg.Auth.OIDC.TokenURL,
			UserInfoURL:  cfg.Auth.OIDC.UserInfoURL,
		})
	}
	if cfg.Auth.Google.Enabled {
		providers = append(providers, platformoauth.NewGothProvider(
			google.New(cfg.Auth.Google.ClientID, cfg.Auth.Google.ClientSecret, redirectURL),
			"Google",
		))
	}
	if cfg.Auth.Apple.Enabled {
		providers = append(providers, platformoauth.NewGothProvider(
			apple.New(cfg.Auth.Apple.ClientID, cfg.Auth.Apple.ClientSecret, redirectURL, nil),
			"Apple",
		))
	}
	if cfg.Auth.Facebook.Enabled {
		providers = append(providers, platformoauth.NewGothProvider(
			facebook.New(cfg.Auth.Facebook.ClientID, cfg.Auth.Facebook.ClientSecret, redirectURL),
			"Facebook",
		))
	}
	if cfg.Auth.GitHub.Enabled {
		providers = append(providers, platformoauth.NewGothProvider(
			github.New(
				cfg.Auth.GitHub.ClientID,
				cfg.Auth.GitHub.ClientSecret,
				redirectURL,
				"user:email",
			),
			"GitHub",
		))
	}
	return providers
}
