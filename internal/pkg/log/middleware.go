package log

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func HTTPMiddleware(l *Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		// The access log measures latency against the server clock; this
		// middleware is the clock's only consumer.
		//kinstack:allow-nondeterminism
		start := time.Now()

		requestID := requestid.FromContext(c)
		scoped := ForRequest(l, requestID)
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		scoped.Infow("Request started",
			"method", method,
			"path", path,
			"ip", ip,
		)

		err := c.Next()

		// Latency is wall-clock by definition.
		//kinstack:allow-nondeterminism
		latency := time.Since(start)

		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			}
		}

		// One access line per request. The failure itself is logged, with its
		// stack, by the error handler; logging it here as well duplicated it.
		scoped.Infow("Request completed",
			"method", method,
			"path", path,
			"ip", ip,
			"status", status,
			"latency", latency.String(),
		)

		return err
	}
}
