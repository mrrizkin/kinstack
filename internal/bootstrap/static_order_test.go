package bootstrap

import (
	"os"
	"strings"
	"testing"
)

// TestStaticAssetsAreMountedBetweenTransportAndAccessMiddleware guards the
// composition contract: assets must run after the header/logging layer (so they
// carry CSP and HSTS) but before the session layer (so cached files never touch
// the session store).
func TestStaticAssetsAreMountedBetweenTransportAndAccessMiddleware(t *testing.T) {
	source, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	body := string(source)

	transport := strings.Index(body, "registerTransportMiddleware(middleware)")
	assets := strings.Index(body, "registerStaticAssets(app)")
	access := strings.Index(body, "RegisterGlobalMiddleware(middleware)")

	if transport < 0 || assets < 0 || access < 0 {
		t.Fatalf(
			"composition calls missing: transport=%d assets=%d access=%d",
			transport, assets, access,
		)
	}
	if !(transport < assets && assets < access) {
		t.Fatalf(
			"assets must be mounted between transport and access middleware: transport=%d assets=%d access=%d",
			transport, assets, access,
		)
	}
}
