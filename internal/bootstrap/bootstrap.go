// Package bootstrap wires application startup and runtime composition.
package bootstrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kinstack/config"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/database"
	"kinstack/internal/platform/email"
	"kinstack/internal/platform/observability"
	"kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
	fiberlog "github.com/gofiber/fiber/v3/log"
)

// Run loads configuration, wires dependencies, and starts the HTTP server.
func Run() error {
	cfg := config.Load()

	telemetry, err := observability.New(observability.Dependency{Config: cfg})
	if err != nil {
		return fmt.Errorf("configure observability: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = telemetry.Shutdown(shutdownCtx)
	}()

	var logWriters []io.Writer
	if writer := telemetry.LogWriter(); writer != nil {
		logWriters = append(logWriters, writer)
	}
	logger := log.NewLogger(log.LoggerDependency{Config: cfg, ExtraWriters: logWriters})
	fiberlog.SetLogger(logger)

	valkeyStorage := session.NewValkeyStorage(cfg.Valkey)
	sessionStore := session.NewStore(session.StoreDependency{
		Config:       cfg.Valkey,
		Storage:      valkeyStorage,
		IdleTimeout:  cfg.App.SessionIdleTimeout,
		CookieSecure: cfg.App.Env == "production",
		Log:          logger,
	})
	runtimeSession := session.NewMiddleware(sessionStore, cfg.App.SessionIdleTimeout)

	db, err := database.OpenWithOptions(cfg.DB.URL, database.Options{
		StatementTimeout: cfg.DB.StatementTimeout,
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	actorRepository, err := actor.NewSQLRepository(actor.RepositoryDependency{DB: db.DB})
	if err != nil {
		return fmt.Errorf("configure actor repository: %w", err)
	}
	defer actorRepository.Close()
	actorBoundary := actor.New(actor.Dependency{Repository: actorRepository})

	auditRepository, err := audit.NewSQLRepository(audit.RepositoryDependency{DB: db.DB})
	if err != nil {
		return fmt.Errorf("configure audit repository: %w", err)
	}
	defer auditRepository.Close()

	emailSender, err := email.New(cfg.Email)
	if err != nil {
		return fmt.Errorf("configure email sender: %w", err)
	}

	readinessChecks := ReadinessChecks{
		Database: func(ctx context.Context) error { return db.DB.PingContext(ctx) },
		Session: func(context.Context) error {
			_, err := valkeyStorage.Get("kinstack:readiness")
			return err
		},
	}

	application, cleanup := NewApp(AppDependency{
		Config:        cfg,
		Database:      db,
		Log:           logger,
		Session:       runtimeSession,
		SessionStore:  sessionStore,
		SharedStorage: valkeyStorage,
		Actor:         actorBoundary,
		Audit:         audit.New(auditRepository),
		Email:         emailSender,
		Readiness:     readinessChecks,
		Telemetry:     telemetry,
	})
	defer cleanup()

	return listen(application, cfg.App.Port, logger)
}

// listen starts the HTTP server and drains it on SIGINT/SIGTERM. Container
// orchestrators send SIGTERM, so graceful shutdown keeps in-flight requests
// from being cut off.
func listen(application *fiber.App, port int, logger *log.Logger) error {
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- application.Listen(fmt.Sprintf(":%d", port))
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Infow("shutting down HTTP server")
	if err := application.ShutdownWithContext(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}
