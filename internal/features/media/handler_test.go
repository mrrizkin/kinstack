package media

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"

	"github.com/gofiber/fiber/v3"
)

type failingStorage struct{ err error }

func (s failingStorage) Put(context.Context, string, io.Reader, int64, string) error {
	return s.err
}

func (s failingStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, s.err
}

func (s failingStorage) Delete(context.Context, string) error { return s.err }

func (s failingStorage) PublicURL(key string) string { return "https://cdn.test/" + key }

func newMediaTestApp(t *testing.T, service *Service) (*fiber.App, *actor.Actor) {
	t.Helper()
	actorBoundary := actor.New(actor.Dependency{Repository: actor.NewMemoryRepository("user-1")})
	// The failure paths under test never render; only the actor boundary and the
	// logger are exercised, so the Inertia adapter stays nil.
	handler := &Handler{service: service, actor: actorBoundary, logger: applog.NoopLogger{}}
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		actorBoundary.SetRequestActor(c, actor.SessionActor{PublicID: "user-1"})
		return c.Next()
	})
	handler.Mount(app)
	return app, actorBoundary
}

func TestFileResponseDoesNotLeakStorageDetails(t *testing.T) {
	repository := NewMemoryRepository()
	repository.Seed(Asset{
		PublicID:      "asset-1",
		OwnerPublicID: "user-1",
		Filename:      "photo.png",
		ContentType:   "image/png",
		SizeBytes:     4,
		ObjectKey:     "media/user-1/asset-1.png",
	})
	service := NewService(ServiceDependency{
		Repository: repository,
		Storage: failingStorage{
			err: errors.New("bucket=private endpoint=s3.internal access denied"),
		},
	})
	app, _ := newMediaTestApp(t, service)

	resp, err := app.Test(httptest.NewRequest("GET", "/media/asset-1/file", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}
	text := string(body)
	if strings.Contains(text, "s3.internal") || strings.Contains(text, "bucket=") {
		t.Fatalf("storage details leaked: %s", text)
	}
	if !strings.Contains(text, ErrStorage.Error()) {
		t.Fatalf("expected the shared storage message, got %s", text)
	}
}

func TestDeleteResponseDoesNotLeakStorageDetails(t *testing.T) {
	repository := NewMemoryRepository()
	repository.Seed(Asset{
		PublicID:      "asset-1",
		OwnerPublicID: "user-1",
		Filename:      "photo.png",
		ContentType:   "image/png",
		SizeBytes:     4,
		ObjectKey:     "media/user-1/asset-1.png",
	})
	service := NewService(ServiceDependency{
		Repository: repository,
		Storage: failingStorage{
			err: errors.New("bucket=private endpoint=s3.internal access denied"),
		},
	})
	app, _ := newMediaTestApp(t, service)

	req := httptest.NewRequest("POST", "/media/asset-1/delete", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}
	if text := string(body); strings.Contains(text, "s3.internal") {
		t.Fatalf("storage details leaked: %s", text)
	}
}

func TestHeaderSafeFilenameStripsHeaderBreakout(t *testing.T) {
	cases := map[string]string{
		"photo.png":            "photo.png",
		"bad\"name\nX-Evil: 1": "bad_name_X-Evil: 1",
		"":                     "download",
		"a\\b.png":             "a_b.png",
	}
	for input, want := range cases {
		if got := headerSafeFilename(input); got != want {
			t.Errorf("headerSafeFilename(%q)=%q, want %q", input, got, want)
		}
	}
	if got := headerSafeFilename("a\r\nb"); strings.ContainsAny(got, "\r\n") {
		t.Fatalf("control characters remain: %q", got)
	}
}
