package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// newStaticApp mounts assets from the repository root, as the server does.
func newStaticApp(t *testing.T) *fiber.App {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	t.Chdir(root)

	app := fiber.New()
	registerStaticAssets(app)

	return app
}

func fetch(t *testing.T, app *fiber.App, path string) *http.Response {
	t.Helper()

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("%s: status = %d, want %d", path, response.StatusCode, fiber.StatusOK)
	}

	return response
}

// TestPWAInstallAssetsAreServedFromRoot: installability needs the manifest and
// the worker at root, and an uncached worker so a deployment can replace it.
func TestPWAInstallAssetsAreServedFromRoot(t *testing.T) {
	app := newStaticApp(t)

	for _, file := range []string{"manifest.webmanifest", "sw.js"} {
		if _, err := os.Stat(filepath.Join("public/build", file)); err != nil {
			t.Skipf("PWA artifact %s missing; run `bun x vite build` first", file)
		}
	}

	manifest := fetch(t, app, "/manifest.webmanifest")
	if got := manifest.Header.Get(fiber.HeaderContentType); got != "application/manifest+json" {
		t.Fatalf("manifest content type = %q", got)
	}

	worker := fetch(t, app, "/sw.js")
	if got := worker.Header.Get(fiber.HeaderCacheControl); !strings.Contains(got, "no-cache") {
		t.Fatalf("sw.js cache-control = %q, want no-cache", got)
	}

	fetch(t, app, "/icons/icon-192.png")

	// sw.js imports its workbox runtime from the same scope under a hashed name.
	chunks, err := filepath.Glob("public/build/workbox-*.js")
	if err != nil || len(chunks) == 0 {
		t.Fatalf("workbox chunk missing from public/build: %v", err)
	}
	fetch(t, app, "/"+filepath.Base(chunks[0]))
}
