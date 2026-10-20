package observability

import (
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/propagation"
)

// headerCarrier adapts Fiber request headers to the OpenTelemetry propagator.
type headerCarrier struct {
	c fiber.Ctx
}

var _ propagation.TextMapCarrier = headerCarrier{}

func (h headerCarrier) Get(key string) string {
	return h.c.Get(key)
}

func (h headerCarrier) Set(key, value string) {
	h.c.Set(key, value)
}

func (h headerCarrier) Keys() []string {
	keys := make([]string, 0, 16)
	for key := range h.c.Request().Header.All() {
		keys = append(keys, string(key))
	}
	return keys
}
