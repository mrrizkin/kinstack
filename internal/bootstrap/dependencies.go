package bootstrap

import (
	"context"
	"io"
	"strings"

	"kinstack/config"
	"kinstack/internal/features/apitoken"
	auditfeature "kinstack/internal/features/audit"
	"kinstack/internal/features/auth"
	mediafeature "kinstack/internal/features/media"
	notificationsfeature "kinstack/internal/features/notifications"
	"kinstack/internal/features/passkeys"
	"kinstack/internal/features/profile"
	"kinstack/internal/features/system"
	"kinstack/internal/features/users"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/objectstorage"
	platformsaml "kinstack/internal/platform/saml"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
)

// appDependencies holds the repositories, platform services, and feature slices
// composed by NewApp. Repositories register their closers so cleanup is owned in
// one place (see cleanup.go).
type appDependencies struct {
	Inertia         *fibertia.Fibertia
	Authorization   *authz.Service
	SessionRegistry *sessionregistry.Service
	TwoFactor       *twofactor.Service

	Auth          *auth.Feature
	Users         *users.Feature
	Profile       *profile.Feature
	APITokens     *apitoken.Feature
	Passkeys      *passkeys.Feature
	Audit         *auditfeature.Feature
	Media         *mediafeature.Feature
	Notifications *notificationsfeature.Feature
	System        *system.Feature

	// BearerAuth resolves a personal API token to a request-scoped actor. It is
	// fail-closed when API tokens are disabled.
	BearerAuth func(context.Context, string) (*actor.SessionActor, error)

	closers []io.Closer
}

func newAppDependencies(d AppDependency, inertiaAdapter *fibertia.Fibertia) *appDependencies {
	deps := &appDependencies{Inertia: inertiaAdapter}

	authRepository := assert.OK(auth.NewSQLRepository(auth.RepositoryDependency{DB: d.Database.DB}))
	deps.track(authRepository)
	usersRepository := assert.OK(users.NewSQLRepository(users.RepositoryDependency{DB: d.Database.DB}))
	deps.track(usersRepository)
	authzRepository := assert.OK(authz.NewSQLRepository(authz.RepositoryDependency{DB: d.Database.DB}))
	deps.track(authzRepository)
	profileRepository := assert.OK(profile.NewSQLRepository(profile.RepositoryDependency{DB: d.Database.DB}))
	deps.track(profileRepository)
	passkeyRepository := assert.OK(passkeys.NewSQLRepository(passkeys.RepositoryDependency{DB: d.Database.DB}))
	deps.track(passkeyRepository)
	apiTokenRepository := assert.OK(apitoken.NewSQLRepository(apitoken.RepositoryDependency{DB: d.Database.DB}))
	deps.track(apiTokenRepository)
	auditRepository := assert.OK(
		auditfeature.NewSQLRepository(auditfeature.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(auditRepository)

	deps.Authorization = authz.New(authz.ServiceDependency{
		Repository: authzRepository,
		CacheTTL:   authz.DefaultCacheTTL,
	})

	if d.Config.RustFS.Enabled {
		storage := objectstorage.New(objectstorage.Dependency{
			Config: d.Config.RustFS,
			Logger: d.Log,
		})
		assert.NoError(storage.EnsureBucket(context.Background()))
		mediaRepository := assert.OK(
			mediafeature.NewSQLRepository(mediafeature.RepositoryDependency{DB: d.Database.DB}),
		)
		deps.track(mediaRepository)
		deps.Media = mediafeature.New(mediafeature.Dependency{
			Repository: mediaRepository,
			Storage:    storage,
			Actor:      d.Actor,
			Logger:     d.Log,
			Inertia:    inertiaAdapter,
		})
	}

	notificationsRepository := assert.OK(
		notificationsfeature.NewSQLRepository(notificationsfeature.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(notificationsRepository)
	deps.Notifications = notificationsfeature.New(notificationsfeature.Dependency{
		Repository: notificationsRepository,
		Actor:      d.Actor,
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
	})

	revoker := &credentialRevoker{log: d.Log}

	sessionRegistryRepository := assert.OK(
		sessionregistry.NewSQLRepository(sessionregistry.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(sessionRegistryRepository)
	deps.SessionRegistry = sessionregistry.New(sessionregistry.Dependency{
		Repository: sessionRegistryRepository,
		Store:      d.SessionStore,
	})
	revoker.sessions = deps.SessionRegistry

	twoFactorRepository := assert.OK(
		twofactor.NewSQLRepository(twofactor.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(twoFactorRepository)
	deps.TwoFactor = twofactor.New(twofactor.Dependency{
		Repository: twoFactorRepository,
		Issuer:     d.Config.App.Name,
		Box:        twoFactorBox(d.Config),
	})

	// Generic OIDC stays disabled unless explicitly enabled; otherwise the auth
	// feature keeps its unavailable placeholder.
	oauthRedirectURL := strings.TrimRight(d.Config.App.URL, "/") + "/login/oauth/callback"
	samlACSURL := strings.TrimRight(d.Config.App.URL, "/") + "/login/saml/acs"
	var samlProvider *platformsaml.Provider
	if d.Config.Auth.SAML.Enabled {
		samlProvider = assert.OK(platformsaml.New(context.Background(), platformsaml.Config{
			EntityID:       d.Config.Auth.SAML.EntityID,
			RootURL:        d.Config.App.URL,
			IDPMetadataURL: d.Config.Auth.SAML.IDPMetadataURL,
			Certificate:    d.Config.Auth.SAML.Certificate,
			PrivateKey:     d.Config.Auth.SAML.PrivateKey,
		}))
	}

	deps.Auth = auth.New(auth.Dependency{
		Repository:    authRepository,
		Actor:         d.Actor,
		Logger:        d.Log,
		AccessKey:     d.Config.App.AccessKey,
		SecretKey:     d.Config.App.SecretKey,
		Email:         d.Email,
		EmailFrom:     d.Config.Email.From,
		BaseURL:       d.Config.App.URL,
		Storage:       d.SharedStorage,
		LockoutMax:    d.Config.App.AuthLockoutMax,
		LockoutWindow: d.Config.App.AuthLockoutWindow,
		Sessions:      deps.SessionRegistry,
		TwoFactor:     deps.TwoFactor,
		Revoker:       revoker,

		MagicLinkEnabled: d.Config.Auth.MagicLink.Enabled,
		PasskeyEnabled:   d.Config.Auth.Passkey.Enabled,
		PasswordReset:    d.Config.Auth.PasswordReset.Enabled,

		OAuthProviders:   oauthProviders(d.Config, oauthRedirectURL),
		OAuthStateSecret: d.Config.Auth.StateSecret,
		OAuthRedirectURL: oauthRedirectURL,

		SAMLProvider: samlProvider,
		SAMLACSURL:   samlACSURL,
	})
	deps.Users = users.New(users.Dependency{
		Repository:    usersRepository,
		Authorization: deps.Authorization,
		Inviter:       deps.Auth.Service(),
		Revoker:       revoker,
		Actor:         d.Actor,
		Logger:        d.Log,
		Inertia:       inertiaAdapter,
	})
	deps.Profile = profile.New(profile.Dependency{
		Repository: profileRepository,
		Actor:      d.Actor,
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
		Sessions:   deps.SessionRegistry,
		TwoFactor:  deps.TwoFactor,
	})
	deps.System = system.New(system.Dependency{
		Repository: system.NewStaticRepository(),
		Logger:     d.Log,
	})
	deps.Audit = auditfeature.New(auditfeature.Dependency{
		Repository: auditRepository,
		Logger:     d.Log,
	})
	deps.APITokens = apitoken.New(apitoken.Dependency{
		Repository:    apiTokenRepository,
		Actor:         d.Actor,
		Authorization: deps.Authorization,
		Logger:        d.Log,
		Inertia:       inertiaAdapter,
	})
	deps.Passkeys = passkeys.New(passkeys.Dependency{
		Repository: passkeyRepository,
		AppURL:     d.Config.App.URL,
		Auth:       deps.Auth,
		Actor:      d.Actor,
		Logger:     d.Log,
	})

	revoker.tokens = deps.APITokens.Service()
	deps.BearerAuth = bearerAuthenticator(d.Config, deps.APITokens, deps.Auth)

	return deps
}

// bearerAuthenticator validates a bearer token and projects the resolved account
// into a request-scoped actor. Disabled API tokens resolve to unauthorized.
func bearerAuthenticator(
	cfg config.Config,
	tokens *apitoken.Feature,
	authFeature *auth.Feature,
) func(context.Context, string) (*actor.SessionActor, error) {
	if !cfg.Auth.APIToken.Enabled {
		return func(context.Context, string) (*actor.SessionActor, error) {
			return nil, fiber.ErrUnauthorized
		}
	}

	return func(ctx context.Context, raw string) (*actor.SessionActor, error) {
		userPublicID, scopes, err := tokens.Service().Authenticate(ctx, raw)
		if err != nil {
			return nil, err
		}
		session, err := authFeature.Service().Session(ctx, userPublicID)
		if err != nil {
			return nil, err
		}
		return &actor.SessionActor{
			PublicID:    session.PublicID,
			Username:    session.Username,
			DisplayName: session.DisplayName,
			Role:        session.Role,
			Scopes:      scopes,
		}, nil
	}
}

func (d *appDependencies) track(closer io.Closer) {
	d.closers = append(d.closers, closer)
}
