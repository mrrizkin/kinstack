package config

import (
	"fmt"
	"net/url"
	"strings"

	"kinstack/internal/pkg/assert"
)

// SecurityConfig controls the Content-Security-Policy and optional source
// extensions. CSP is enabled by default in production because Vite's dev
// server needs a looser policy.
type SecurityConfig struct {
	CSPEnabled          bool
	CSPReportOnly       bool
	CSPReportURI        string
	ExtraScriptSources  []string
	ExtraConnectSources []string
	ExtraImageSources   []string
}

// Load reads security settings from the environment.
func (c *SecurityConfig) Load() {
	c.CSPEnabled = assert.OK(envBool("CSP_ENABLED", isProduction()))
	c.CSPReportOnly = assert.OK(envBool("CSP_REPORT_ONLY", false))
	c.CSPReportURI = strings.TrimSpace(env("CSP_REPORT_URI", "/csp-report"))
	c.ExtraScriptSources = envList("CSP_EXTRA_SCRIPT_SRC")
	c.ExtraConnectSources = envList("CSP_EXTRA_CONNECT_SRC")
	c.ExtraImageSources = envList("CSP_EXTRA_IMG_SRC")
}

// Validate rejects values that would break the policy or inject headers.
func (c SecurityConfig) Validate() error {
	if c.CSPReportURI != "" && !strings.HasPrefix(c.CSPReportURI, "/") {
		if parsed, err := url.Parse(c.CSPReportURI); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("CSP_REPORT_URI must be a relative path or an https URL")
		}
	}
	if strings.ContainsAny(c.CSPReportURI, ";\r\n") {
		return fmt.Errorf("CSP_REPORT_URI contains an invalid value")
	}
	for name, sources := range map[string][]string{
		"CSP_EXTRA_SCRIPT_SRC":  c.ExtraScriptSources,
		"CSP_EXTRA_CONNECT_SRC": c.ExtraConnectSources,
		"CSP_EXTRA_IMG_SRC":     c.ExtraImageSources,
	} {
		for _, source := range sources {
			if strings.ContainsAny(source, ";,\r\n") {
				return fmt.Errorf("%s contains an invalid source: %q", name, source)
			}
		}
	}
	return nil
}
