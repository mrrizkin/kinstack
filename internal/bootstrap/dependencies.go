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
	"kinstack/internal/platform/inertia"
	"kinstack/internal/platform/objectstorage"
	platformsaml "kinstack/internal/platform/saml"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
)

// appDependencies holds the repositories, platform services, and feature slices
// composed by NewApp. Repositories register their closers so cleanup is owned in
// one place (see cleanup.go).
type appDependencies struct {
	Inertia         *inertia.Adapter
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

func newAppDependencies(d AppDependency, inertiaAdapter *inertia.Adapter) *appDependencies {
	deps := &appDependencies{Inertia: inertiaAdapter}
	repositories := newAppRepositories(deps, d)

	deps.Authorization = authz.New(authz.ServiceDependency{
		Repository: repositories.authz,
		CacheTTL:   authz.DefaultCacheTTL,
	})

	revoker := &credentialRevoker{log: d.Log}
	deps.SessionRegistry = newSessionRegistry(deps, d)
	revoker.sessions = deps.SessionRegistry
	deps.TwoFactor = newTwoFactor(deps, d)
	// Profile owns the account page-size preference, so it is built before every
	// list feature: only the composition root may connect two slices.
	deps.Profile = newProfileFeature(d, deps, repositories.profile, inertiaAdapter)
	deps.Media = newMediaFeature(d, deps, inertiaAdapter)
	deps.Notifications = newNotificationsFeature(d, deps, inertiaAdapter)

	deps.Auth = newAuthFeature(d, authFeatureDependency{
		repositories: repositories,
		deps:         deps,
		revoker:      revoker,
		saml:         newSAMLProvider(d),
	})
	deps.Users = newUsersFeature(d, deps, repositories.users, revoker, inertiaAdapter)
	deps.System = system.New(system.Dependency{
		Repository: system.NewStaticRepository(),
		Logger:     d.Log,
	})
	deps.Audit = auditfeature.New(auditfeature.Dependency{
		Repository: repositories.audit,
		Actor:      d.Actor,
		PageSize:   deps.Profile.Service(),
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
	})
	deps.APITokens = apitoken.New(apitoken.Dependency{
		Repository:    repositories.apiToken,
		Actor:         d.Actor,
		Authorization: deps.Authorization,
		PageSize:      deps.Profile.Service(),
		Logger:        d.Log,
		Inertia:       inertiaAdapter,
	})
	deps.Passkeys = passkeys.New(passkeys.Dependency{
		Repository: repositories.passkey,
		AppURL:     d.Config.App.URL,
		Auth:       deps.Auth,
		Actor:      d.Actor,
		PageSize:   deps.Profile.Service(),
		Logger:     d.Log,
	})

	revoker.tokens = deps.APITokens.Service()
	deps.BearerAuth = bearerAuthenticator(d.Config, deps.APITokens, deps.Auth)
	return deps
}

// appRepositories groups the SQL repositories owned by the composition root.
type appRepositories struct {
	auth     *auth.SQLRepository
	users    *users.SQLRepository
	authz    *authz.SQLRepository
	profile  *profile.SQLRepository
	passkey  *passkeys.SQLRepository
	apiToken *apitoken.SQLRepository
	audit    *auditfeature.SQLRepository
}

func newAppRepositories(deps *appDependencies, d AppDependency) *appRepositories {
	repositories := &appRepositories{}
	repositories.auth = assert.OK(
		auth.NewSQLRepository(auth.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.auth)
	repositories.users = assert.OK(
		users.NewSQLRepository(users.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.users)
	repositories.authz = assert.OK(
		authz.NewSQLRepository(authz.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.authz)
	repositories.profile = assert.OK(
		profile.NewSQLRepository(profile.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.profile)
	repositories.passkey = assert.OK(
		passkeys.NewSQLRepository(passkeys.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.passkey)
	repositories.apiToken = assert.OK(
		apitoken.NewSQLRepository(apitoken.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.apiToken)
	repositories.audit = assert.OK(
		auditfeature.NewSQLRepository(auditfeature.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repositories.audit)
	return repositories
}

func newMediaFeature(
	d AppDependency,
	deps *appDependencies,
	inertiaAdapter *inertia.Adapter,
) *mediafeature.Feature {
	if !d.Config.RustFS.Enabled {
		return nil
	}
	storage := objectstorage.New(objectstorage.Dependency{Config: d.Config.RustFS, Logger: d.Log})
	assert.NoError(storage.EnsureBucket(context.Background()))
	repository := assert.OK(
		mediafeature.NewSQLRepository(mediafeature.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repository)
	return mediafeature.New(mediafeature.Dependency{
		Repository: repository,
		Storage:    storage,
		Actor:      d.Actor,
		PageSize:   deps.Profile.Service(),
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
	})
}

func newNotificationsFeature(
	d AppDependency,
	deps *appDependencies,
	inertiaAdapter *inertia.Adapter,
) *notificationsfeature.Feature {
	repository := assert.OK(
		notificationsfeature.NewSQLRepository(
			notificationsfeature.RepositoryDependency{DB: d.Database.DB},
		),
	)
	deps.track(repository)
	return notificationsfeature.New(notificationsfeature.Dependency{
		Repository: repository,
		Actor:      d.Actor,
		PageSize:   deps.Profile.Service(),
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
	})
}

func newSessionRegistry(deps *appDependencies, d AppDependency) *sessionregistry.Service {
	repository := assert.OK(
		sessionregistry.NewSQLRepository(sessionregistry.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repository)
	return sessionregistry.New(sessionregistry.Dependency{
		Repository: repository,
		Store:      d.SessionStore,
	})
}

func newTwoFactor(deps *appDependencies, d AppDependency) *twofactor.Service {
	repository := assert.OK(
		twofactor.NewSQLRepository(twofactor.RepositoryDependency{DB: d.Database.DB}),
	)
	deps.track(repository)
	return twofactor.New(twofactor.Dependency{
		Repository: repository,
		Issuer:     d.Config.App.Name,
		Box:        twoFactorBox(d.Config),
	})
}

// newSAMLProvider builds the SAML service provider when the feature is enabled.
func newSAMLProvider(d AppDependency) *platformsaml.Provider {
	if !d.Config.Auth.SAML.Enabled {
		return nil
	}
	return assert.OK(platformsaml.New(context.Background(), platformsaml.Config{
		EntityID:       d.Config.Auth.SAML.EntityID,
		RootURL:        d.Config.App.URL,
		IDPMetadataURL: d.Config.Auth.SAML.IDPMetadataURL,
		Certificate:    d.Config.Auth.SAML.Certificate,
		PrivateKey:     d.Config.Auth.SAML.PrivateKey,
	}))
}

type authFeatureDependency struct {
	repositories *appRepositories
	deps         *appDependencies
	revoker      *credentialRevoker
	saml         *platformsaml.Provider
}

func newAuthFeature(d AppDependency, dependency authFeatureDependency) *auth.Feature {
	deps := dependency.deps
	oauthRedirectURL := strings.TrimRight(d.Config.App.URL, "/") + "/login/oauth/callback"
	return auth.New(auth.Dependency{
		Repository:    dependency.repositories.auth,
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
		Revoker:       dependency.revoker,

		MagicLinkEnabled: d.Config.Auth.MagicLink.Enabled,
		PasskeyEnabled:   d.Config.Auth.Passkey.Enabled,
		PasswordReset:    d.Config.Auth.PasswordReset.Enabled,

		OAuthProviders:   oauthProviders(d.Config, oauthRedirectURL),
		OAuthStateSecret: d.Config.Auth.StateSecret,
		OAuthRedirectURL: oauthRedirectURL,

		SAMLProvider: dependency.saml,
		SAMLACSURL:   strings.TrimRight(d.Config.App.URL, "/") + "/login/saml/acs",
	})
}

func newUsersFeature(
	d AppDependency,
	deps *appDependencies,
	repository *users.SQLRepository,
	revoker *credentialRevoker,
	inertiaAdapter *inertia.Adapter,
) *users.Feature {
	return users.New(users.Dependency{
		Repository:    repository,
		Authorization: deps.Authorization,
		Inviter:       deps.Auth.Service(),
		Revoker:       revoker,
		Actor:         d.Actor,
		PageSize:      deps.Profile.Service(),
		Logger:        d.Log,
		Inertia:       inertiaAdapter,
	})
}

func newProfileFeature(
	d AppDependency,
	deps *appDependencies,
	repository *profile.SQLRepository,
	inertiaAdapter *inertia.Adapter,
) *profile.Feature {
	return profile.New(profile.Dependency{
		Repository: repository,
		Actor:      d.Actor,
		Logger:     d.Log,
		Inertia:    inertiaAdapter,
		Sessions:   deps.SessionRegistry,
		TwoFactor:  deps.TwoFactor,
	})
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
