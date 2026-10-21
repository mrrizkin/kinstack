// Package inertia configures the Fibertia server adapter and document shell.
package inertia

import (
	"context"
	"io"
	"runtime/debug"
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

// documentTimeout bounds one shell render. Rendering only writes buffered
// strings, so a healthy render finishes in milliseconds.
const documentTimeout = 10 * time.Second

// Adapter is the application's Inertia boundary: Fibertia plus shared response helpers.
type Adapter struct {
	*fibertia.Fibertia
}

// Dependency contains the environment and session boundaries used to construct the adapter.
type Dependency struct {
	Environment  string
	SessionStore *fibersession.Store
}

func New(d Dependency) *Adapter {
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
		Version:        BuildVersion(),
	})
	adapter.SetSession(platformsession.NewInertiaSessionStore(d.SessionStore))
	adapter.SetRendererWithData(
		func(w io.Writer, head fibertia.InertiaHead, body fibertia.InertiaBody, data map[string]any) error {
			// The document renderer is adapter scoped and receives no request
			// context, so it owns a short deadline instead of an unbounded one.
			ctx, cancel := context.WithTimeout(context.Background(), documentTimeout)
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
	return &Adapter{Fibertia: adapter}
}

// buildVersion is injected at build time, for example with
// -ldflags "-X kinstack/internal/platform/inertia.buildVersion=$GIT_SHA".
// Container builds strip .git, so this is the only reliable version source
// there.
var buildVersion string

// BuildVersion resolves the Inertia asset version. Build-time injection wins;
// otherwise VCS metadata embedded by the Go toolchain is used. A new commit or
// build therefore invalidates cached pages, and builds without either source
// fall back to a stable development marker.
func BuildVersion() string {
	if version := strings.TrimSpace(buildVersion); version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	revision := ""
	dirty := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if revision == "" {
		return "dev"
	}
	if dirty {
		return revision + "-dirty"
	}
	return revision
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

func Middleware(adapter *Adapter) fiber.Handler {
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
func (a *Adapter) ValidationErrors(c fiber.Ctx, back string, fields map[string]string) error {
	bag := make(map[string][]string, len(fields))
	for field, message := range fields {
		bag[field] = []string{message}
	}
	if err := a.WithErrors(c, "", bag); err != nil {
		return err
	}
	if back == "" {
		back = "/"
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(back)
}

// FlashSuccess stores a success flash message for the next page load.
func (a *Adapter) FlashSuccess(c fiber.Ctx, message string) error {
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return a.Flash(c, map[string]any{"success": message})
}

// RedirectWithFlash flashes a success message and redirects (303) to target.
func (a *Adapter) RedirectWithFlash(c fiber.Ctx, target string, message string) error {
	if err := a.FlashSuccess(c, message); err != nil {
		return err
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(target)
}

// WantsJSON reports whether the caller expects JSON instead of an Inertia page.
// Inertia visits always send X-Inertia, so Accept alone is not enough.
func WantsJSON(c fiber.Ctx) bool {
	return c.Get("X-Inertia") == "" &&
		strings.Contains(c.Get(fiber.HeaderAccept), "application/json")
}

// FlashError stores an error flash for the next page load.
func (a *Adapter) FlashError(c fiber.Ctx, message string) error {
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return a.Flash(c, map[string]any{"error": message})
}

// RedirectWithFlashError flashes an error and redirects (303) to target.
func (a *Adapter) RedirectWithFlashError(c fiber.Ctx, target string, message string) error {
	if err := a.FlashError(c, message); err != nil {
		return err
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(target)
}

// FailedPayload answers an unreadable body: JSON clients get a 400 envelope,
// Inertia visits a form error.
func (a *Adapter) FailedPayload(c fiber.Ctx, back string, message string) error {
	if WantsJSON(c) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": message})
	}
	return a.ValidationErrors(c, back, map[string]string{"form": message})
}

// FailedAction answers a rejected mutation: JSON clients get the status
// envelope, Inertia visits an error flash and a redirect.
func (a *Adapter) FailedAction(c fiber.Ctx, back string, status int, message string) error {
	if WantsJSON(c) {
		return c.Status(status).JSON(fiber.Map{"error": message})
	}
	return a.RedirectWithFlashError(c, back, message)
}
