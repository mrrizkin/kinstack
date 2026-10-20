package bootstrap

import (
	"context"
	"strings"
	"time"

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
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/database"
	"kinstack/internal/platform/email"
	"kinstack/internal/platform/inertia"
	"kinstack/internal/platform/objectstorage"
	"kinstack/internal/platform/observability"
	"kinstack/internal/platform/pdf"
	platformsaml "kinstack/internal/platform/saml"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

// AppDependency contains every dependency required to construct the HTTP application.
type AppDependency struct {
	Config        config.Config
	Database      *database.DB
	Log           *log.Logger
	Session       fiber.Handler
	SessionStore  *fibersession.Store
	SharedStorage fiber.Storage
	Actor         *actor.Actor
	Audit         audit.Recorder
	Email         email.Sender
	Readiness     ReadinessChecks
	Telemetry     *observability.Telemetry
}

// NewApp composes the platform boundaries and feature slices into an HTTP app.
// The returned cleanup releases repositories the application owns.
func NewApp(d AppDependency) (*fiber.App, func()) {
	assert.Assert(d.Database != nil, "database is required")
	assert.Assert(d.Log != nil, "logger is required")
	assert.Assert(d.Session != nil, "session middleware is required")
	assert.Assert(d.SessionStore != nil, "session store is required")
	assert.Assert(d.Actor != nil, "actor boundary is required")
	assert.Assert(d.Audit != nil, "audit recorder is required")
	assert.Assert(d.Email != nil, "email sender is required")

	inertiaAdapter := inertia.New(inertia.Dependency{
		Environment:  d.Config.App.Env,
		SessionStore: d.SessionStore,
	})
	documentRenderer := pdf.New(pdf.Config{
		URL:     d.Config.Document.GotenbergURL,
		Timeout: d.Config.Document.Timeout,
	})
	app := newFiberApp(FiberAppDependency{
		Config:    d.Config,
		Inertia:   inertiaAdapter,
		Log:       d.Log,
		Telemetry: d.Telemetry,
	})

	authRepository := assert.OK(auth.NewSQLRepository(auth.RepositoryDependency{DB: d.Database.DB}))
	usersRepository := assert.OK(
		users.NewSQLRepository(users.RepositoryDependency{DB: d.Database.DB}),
	)
	authzRepository := assert.OK(
		authz.NewSQLRepository(authz.RepositoryDependency{DB: d.Database.DB}),
	)
	authorization := authz.New(authz.ServiceDependency{
		Repository: authzRepository,
		CacheTTL:   30 * time.Second,
	})
	profileRepository := assert.OK(
		profile.NewSQLRepository(profile.RepositoryDependency{DB: d.Database.DB}),
	)
	passkeyRepository := assert.OK(
		passkeys.NewSQLRepository(passkeys.RepositoryDependency{DB: d.Database.DB}),
	)
	apiTokenRepository := assert.OK(
		apitoken.NewSQLRepository(apitoken.RepositoryDependency{DB: d.Database.DB}),
	)
	auditRepository := assert.OK(
		auditfeature.NewSQLRepository(auditfeature.RepositoryDependency{DB: d.Database.DB}),
	)
	var mediaRepository *mediafeature.SQLRepository
	var mediaFeature *mediafeature.Feature
	if d.Config.RustFS.Enabled {
		storage := objectstorage.New(objectstorage.Dependency{
			Config: d.Config.RustFS,
			Logger: d.Log,
		})
		assert.NoError(storage.EnsureBucket(context.Background()))
		mediaRepository = assert.OK(
			mediafeature.NewSQLRepository(mediafeature.RepositoryDependency{DB: d.Database.DB}),
		)
		mediaFeature = mediafeature.New(mediafeature.Dependency{
			Repository: mediaRepository,
			Storage:    storage,
			Actor:      d.Actor,
			Logger:     d.Log,
			Inertia:    inertiaAdapter,
		})
	}
	notificationsRepository := assert.OK(
		notificationsfeature.NewSQLRepository(
			notificationsfeature.RepositoryDependency{DB: d.Database.DB},
		),
	)
	notificationsFeature := notificationsfeature.New(notificationsfeature.Dependency{
		Repository: notificationsRepository,
		Actor:      d.Actor,
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
	})
	sessionRegistryRepository := assert.OK(
		sessionregistry.NewSQLRepository(sessionregistry.RepositoryDependency{DB: d.Database.DB}),
	)
	sessionRegistry := sessionregistry.New(sessionregistry.Dependency{
		Repository: sessionRegistryRepository,
		Store:      d.SessionStore,
	})
	twoFactorRepository := assert.OK(
		twofactor.NewSQLRepository(twofactor.RepositoryDependency{DB: d.Database.DB}),
	)
	twoFactor := twofactor.New(twofactor.Dependency{
		Repository: twoFactorRepository,
		Issuer:     d.Config.App.Name,
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

	authFeature := auth.New(auth.Dependency{
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
		Sessions:      sessionRegistry,
		TwoFactor:     twoFactor,

		MagicLinkEnabled: d.Config.Auth.MagicLink.Enabled,
		PasskeyEnabled:   d.Config.Auth.Passkey.Enabled,
		PasswordReset:    d.Config.Auth.PasswordReset.Enabled,

		OAuthProviders:   oauthProviders(d.Config, oauthRedirectURL),
		OAuthStateSecret: d.Config.Auth.StateSecret,
		OAuthRedirectURL: oauthRedirectURL,

		SAMLProvider: samlProvider,
		SAMLACSURL:   samlACSURL,
	})
	usersFeature := users.New(users.Dependency{
		Repository:    usersRepository,
		Authorization: authorization,
		Inviter:       authFeature.Service(),
		Actor:         d.Actor,
		Logger:        d.Log,
		Inertia:       inertiaAdapter,
	})
	profileFeature := profile.New(profile.Dependency{
		Repository: profileRepository,
		Actor:      d.Actor,
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
		Sessions:   sessionRegistry,
		TwoFactor:  twoFactor,
	})
	systemFeature := system.New(system.Dependency{
		Repository: system.NewStaticRepository(),
		Logger:     d.Log,
	})
	auditFeature := auditfeature.New(auditfeature.Dependency{
		Repository: auditRepository,
		Logger:     d.Log,
	})
	apiTokenFeature := apitoken.New(apitoken.Dependency{
		Repository:    apiTokenRepository,
		Actor:         d.Actor,
		Authorization: authorization,
		Logger:        d.Log,
		Inertia:       inertiaAdapter,
	})
	passkeysFeature := passkeys.New(passkeys.Dependency{
		Repository: passkeyRepository,
		AppURL:     d.Config.App.URL,
		Auth:       authFeature,
		Actor:      d.Actor,
		Logger:     d.Log,
	})

	// bearerAuth resolves an API token to a request-scoped actor without a
	// browser session, reusing the auth service for the account projection. It
	// is fail-closed when personal API tokens are disabled.
	var bearerAuth func(context.Context, string) (*actor.SessionActor, error)
	if d.Config.Auth.APIToken.Enabled {
		bearerAuth = func(ctx context.Context, raw string) (*actor.SessionActor, error) {
			userPublicID, scopes, err := apiTokenFeature.Service().Authenticate(ctx, raw)
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
	} else {
		bearerAuth = func(context.Context, string) (*actor.SessionActor, error) {
			return nil, fiber.ErrUnauthorized
		}
	}

	RegisterGlobalMiddleware(GlobalMiddlewareDependency{
		App:           app,
		Config:        d.Config,
		Inertia:       inertiaAdapter,
		Session:       d.Session,
		SessionStore:  d.SessionStore,
		SharedStorage: d.SharedStorage,
		Actor:         d.Actor,
		Authz:         authorization,
		Audit:         d.Audit,
		Log:           d.Log,
		Documents:     documentRenderer,
		Sessions:      sessionRegistry,
		BearerAuth:    bearerAuth,
		Telemetry:     d.Telemetry,
	})

	RegisterWebRoutes(WebRoutesDependency{
		App:           app,
		Config:        d.Config,
		Inertia:       inertiaAdapter,
		Actor:         d.Actor,
		Authz:         authorization,
		Auth:          authFeature,
		Users:         usersFeature,
		Profile:       profileFeature,
		APITokens:     apiTokenFeature,
		Passkeys:      passkeysFeature,
		Audit:         auditFeature,
		Media:         mediaFeature,
		Notifications: notificationsFeature,
		System:        systemFeature,
		Readiness:     d.Readiness,
	})

	cleanup := func() {
		assert.NoError(authRepository.Close())
		assert.NoError(usersRepository.Close())
		assert.NoError(authzRepository.Close())
		assert.NoError(profileRepository.Close())
		assert.NoError(apiTokenRepository.Close())
		assert.NoError(auditRepository.Close())
		assert.NoError(notificationsRepository.Close())
		if mediaRepository != nil {
			assert.NoError(mediaRepository.Close())
		}
		assert.NoError(passkeyRepository.Close())
		assert.NoError(sessionRegistryRepository.Close())
		assert.NoError(twoFactorRepository.Close())
	}
	return app, cleanup
}
