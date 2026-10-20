package log

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// HTTPMiddleware logs each HTTP request with structured fields: request ID,
// method, path, client IP, status code, and latency. Errors are logged at
// Error level; successful requests at Info level.
func HTTPMiddleware(l *Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		requestID := requestid.FromContext(c)
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		l.Infow("Request started",
			"request_id", requestID,
			"method", method,
			"path", path,
			"ip", ip,
		)

		err := c.Next()

		latency := time.Since(start)

		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			}

			l.Errorw("Request completed",
				"request_id", requestID,
				"method", method,
				"path", path,
				"ip", ip,
				"status", status,
				"latency", latency.String(),
				"error", err,
			)
		} else {
			l.Infow("Request completed",
				"request_id", requestID,
				"method", method,
				"path", path,
				"ip", ip,
				"status", status,
				"latency", latency.String(),
			)
		}

		return err
	}
}
