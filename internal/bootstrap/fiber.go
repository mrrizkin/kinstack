package bootstrap

import (
	"os"
	"strings"

	"kinstack/config"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/debug"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/httperror"
	"kinstack/internal/platform/observability"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type FiberAppDependency struct {
	Config    config.Config
	Inertia   *fibertia.Fibertia
	Log       *log.Logger
	Telemetry *observability.Telemetry
}

func newFiberApp(d FiberAppDependency) *fiber.App {
	assert.Assert(d.Config.App.Name != "", "app name is required")
	assert.Assert(d.Inertia != nil, "inertia is required")
	assert.Assert(d.Log != nil, "logger is required")

	fiberConfig := fiber.Config{
		AppName:      d.Config.App.Name,
		BodyLimit:    int(d.Config.App.HTTPBodyLimit),
		ReadTimeout:  d.Config.App.HTTPReadTimeout,
		WriteTimeout: d.Config.App.HTTPWriteTimeout,
		IdleTimeout:  d.Config.App.HTTPIdleTimeout,
		Views:        d.Inertia.Engine,
		ErrorHandler: d.Inertia.ErrorHandler(func(c fiber.Ctx, err error) error {
			problem := httperror.Describe(err)
			if problem.Status >= fiber.StatusInternalServerError {
				d.Telemetry.CaptureException(c, err)
			}
			recovererStackFrames := []debug.StackFrame{}

			if stack, ok := c.Locals("recoverer_stack_trace").([]debug.StackFrame); ok {
				recovererStackFrames = stack
			}

			if len(recovererStackFrames) > 0 {
				d.Log.Errorw(
					"unexpected request failed",
					"code",
					problem.Code,
					"status",
					problem.Status,
					"error",
					err,
					"trace",
					recovererStackFrames,
				)
			}

			// JSON clients receive one stable envelope regardless of which feature
			// produced the error. Inertia requests intentionally keep the page
			// contract so navigation and server-rendered fallbacks remain intact.
			if strings.Contains(c.Get("Accept"), "application/json") && c.Get("X-Inertia") == "" {
				return c.Status(problem.Status).JSON(fiber.Map{"error": problem})
			}

			component := "errors/server"
			if problem.Status == fiber.StatusNotFound {
				component = "errors/not-found"
			}
			return c.Status(problem.Status).Render(component, fiber.Map{
				"status":    problem.Status,
				"code":      problem.Code,
				"kind":      problem.Kind,
				"message":   problem.Message,
				"retryable": problem.Retryable,
				"trace":     recovererStackFrames,
			})
		}),
	}
	if d.Config.App.Env == "production" {
		// Trust forwarded scheme only from private/loopback reverse proxies so
		// Helmet can emit HSTS after TLS termination without trusting public clients.
		fiberConfig.TrustProxy = true
		fiberConfig.TrustProxyConfig = fiber.TrustProxyConfig{Loopback: true, Private: true}
	}
	app := fiber.New(fiberConfig)
	d.Log.Infow("Fiber configured")
	return app
}

// registerStaticAssets serves the Vite build output and the public root files.
// Inertia switches to dev-server URLs while public/hot exists, so the same
// binary keeps working with and without the Vite dev server.
func registerStaticAssets(app *fiber.App) {
	assert.Assert(app != nil, "fiber app is required")

	app.Get("/build/*", static.New(
		"public/build",
		static.Config{MaxAge: 31536000, CacheDuration: -1},
	))

	for _, file := range []string{"favicon.svg", "robots.txt", "manifest.webmanifest"} {
		path := "public/" + file
		if _, err := os.Stat(path); err != nil {
			continue
		}
		app.Get("/"+file, static.New(path, static.Config{MaxAge: 86400, CacheDuration: -1}))
	}
}
