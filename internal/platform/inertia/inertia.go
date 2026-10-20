// Package inertia configures the Fibertia server adapter and document shell.
package inertia

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/security"
	platformsession "kinstack/internal/platform/session"
	"kinstack/resources/views"

	"codeberg.org/mrrizkin/fibertia"
	vitego "codeberg.org/mrrizkin/vite-go"
	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

// Dependency contains the environment and session boundaries used to construct the adapter.
type Dependency struct {
	Environment  string
	SessionStore *fibersession.Store
}

func New(d Dependency) *fibertia.Fibertia {
	vite := newVite()
	baseTags, err := vite.ToHTML()
	viteAvailable := err == nil
	if d.Environment == "test" && err != nil {
		baseTags = ""
		err = nil
	}
	assert.NoError(err)

	var viteMu sync.Mutex
	// vite-go bakes the nonce into the generated tags. The Vite instance caches
	// its manifest, so regenerating tags per document only rebuilds the strings.
	// The mutex serialises UseCspNonce/ToHTML because Vite holds request state.
	tags := func(nonce string) (string, error) {
		if nonce == "" || !viteAvailable {
			return baseTags, nil
		}
		viteMu.Lock()
		defer viteMu.Unlock()
		vite.UseCspNonce(nonce)
		return vite.ToHTML()
	}

	adapter := fibertia.New(fibertia.Config{
		EncryptHistory: true,
		RootView:       "app",
		Version:        "v3",
	})
	adapter.SetSession(platformsession.NewInertiaSessionStore(d.SessionStore))
	adapter.SetRendererWithData(
		func(w io.Writer, head fibertia.InertiaHead, body fibertia.InertiaBody, data map[string]any) error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			nonce, _ := data["nonce"].(string)
			viteTags, err := tags(nonce)
			if err != nil {
				return err
			}
			return views.InertiaEntry(
				viteTags,
				views.ScriptNonce(string(head), nonce),
				string(body),
				nonce,
			).Render(ctx, w)
		},
	)
	return adapter
}

func newVite() *vitego.Vite {
	vite := vitego.NewVite()
	vite.WithEntryPoints([]string{"resources/js/app.ts"})
	vite.UseBuildDirectory("public/build")
	vite.UseHotFile("public/hot")
	vite.UseWaterfallPrefetching(nil)
	vite.CreateAssetPathsUsing(func(path string, _ bool) string {
		return strings.TrimPrefix(path, "public")
	})
	return vite
}

func Middleware(adapter *fibertia.Fibertia) fiber.Handler {
	assert.Assert(adapter != nil, "inertia adapter is required")

	handler := adapter.Middleware
	return func(c fiber.Ctx) error {
		if nonce := security.Nonce(c); nonce != "" {
			adapter.WithViewData(c, "nonce", nonce)
		}
		return handler(c)
	}
}

// ValidationErrors stores field errors in the session and redirects back so the
// Inertia client renders them on the originating form. Use the "form" key for
// errors that do not belong to a single field.
func ValidationErrors(
	adapter *fibertia.Fibertia,
	c fiber.Ctx,
	back string,
	fields map[string]string,
) error {
	assert.Assert(adapter != nil, "inertia adapter is required")

	bag := make(map[string][]string, len(fields))
	for field, message := range fields {
		bag[field] = []string{message}
	}
	if err := adapter.WithErrors(c, "", bag); err != nil {
		return err
	}
	if back == "" {
		back = "/"
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(back)
}

// FlashSuccess stores a success flash message for the next page load.
func FlashSuccess(adapter *fibertia.Fibertia, c fiber.Ctx, message string) error {
	assert.Assert(adapter != nil, "inertia adapter is required")
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return adapter.Flash(c, map[string]any{"success": message})
}

// RedirectWithFlash flashes a success message and redirects (303) to target.
func RedirectWithFlash(
	adapter *fibertia.Fibertia,
	c fiber.Ctx,
	target string,
	message string,
) error {
	if err := FlashSuccess(adapter, c, message); err != nil {
		return err
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(target)
}
