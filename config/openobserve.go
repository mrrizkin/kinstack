package config

import (
	"fmt"
	"strings"

	"kinstack/internal/pkg/assert"
)

// OpenObserveConfig exports OpenTelemetry traces and logs to an OpenObserve
// instance over OTLP/HTTP. The Authorization header is passed through verbatim
// so both Basic (email:password) and Bearer ingestion tokens work.
type OpenObserveConfig struct {
	Enabled          bool
	Endpoint         string
	Organization     string
	AuthHeader       string
	StreamName       string
	ServiceName      string
	Environment      string
	Release          string
	TracesSampleRate float64
	ExportLogs       bool
	ExportMetrics    bool
	Insecure         bool
}

// Load reads observability settings from the environment.
func (c *OpenObserveConfig) Load() {
	c.Enabled = assert.OK(envBool("OPENOBSERVE_ENABLED", false))
	c.Endpoint = strings.TrimRight(strings.TrimSpace(env("OPENOBSERVE_ENDPOINT", "")), "/")
	c.Organization = strings.TrimSpace(env("OPENOBSERVE_ORG", "default"))
	c.AuthHeader = strings.TrimSpace(env("OPENOBSERVE_AUTH_HEADER", ""))
	c.StreamName = strings.TrimSpace(env("OPENOBSERVE_STREAM", "kinstack"))
	c.ServiceName = strings.TrimSpace(env("OPENOBSERVE_SERVICE_NAME", "kinstack"))
	c.Environment = strings.TrimSpace(env("OPENOBSERVE_ENVIRONMENT", env("APP_ENV", "development")))
	c.Release = strings.TrimSpace(env("OPENOBSERVE_RELEASE", ""))
	c.TracesSampleRate = assert.OK(envFloat("OPENOBSERVE_TRACES_SAMPLE_RATE", 1))
	c.ExportLogs = assert.OK(envBool("OPENOBSERVE_EXPORT_LOGS", true))
	c.ExportMetrics = assert.OK(envBool("OPENOBSERVE_EXPORT_METRICS", true))
	c.Insecure = assert.OK(envBool("OPENOBSERVE_INSECURE", false))
}

// Validate checks observability settings. Disabled observability ignores every
// setting so local development needs no OpenObserve instance.
func (c OpenObserveConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := validateURL("OPENOBSERVE_ENDPOINT", c.Endpoint, false); err != nil {
		return err
	}
	if c.Organization == "" {
		return fmt.Errorf("OPENOBSERVE_ORG is required when OPENOBSERVE_ENABLED is true")
	}
	if c.StreamName == "" {
		return fmt.Errorf("OPENOBSERVE_STREAM is required when OPENOBSERVE_ENABLED is true")
	}
	if c.ServiceName == "" {
		return fmt.Errorf("OPENOBSERVE_SERVICE_NAME is required when OPENOBSERVE_ENABLED is true")
	}
	if c.TracesSampleRate < 0 || c.TracesSampleRate > 1 {
		return fmt.Errorf("OPENOBSERVE_TRACES_SAMPLE_RATE must be between 0 and 1")
	}
	if isProduction() && c.AuthHeader == "" {
		return fmt.Errorf("OPENOBSERVE_AUTH_HEADER is required in production")
	}
	return nil
}
