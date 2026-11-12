package objectstorage

import (
	"context"
	"strings"
	"testing"
	"time"

	"kinstack/config"
	applog "kinstack/internal/pkg/log"
)

func testConfig() config.RustFSConfig {
	return config.RustFSConfig{
		Enabled:   true,
		Endpoint:  "http://localhost:9000",
		PublicURL: "http://localhost:9000/",
		AccessKey: "access",
		SecretKey: "secret",
		Bucket:    "kinstack",
		Region:    "us-east-1",
		Timeout:   5 * time.Second,
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	return New(Dependency{Config: testConfig(), Logger: applog.NewNoopLogger()})
}

func TestPublicURLJoinsBucketAndEscapesKey(t *testing.T) {
	client := newTestClient(t)
	got := client.PublicURL("media/a b.png")
	want := "http://localhost:9000/kinstack/media/a%20b.png"
	if got != want {
		t.Fatalf("PublicURL = %q, want %q", got, want)
	}
}

func TestPresignedGetURLCarriesTheLifetime(t *testing.T) {
	client := newTestClient(t)
	url, err := client.PresignedGetURL(context.Background(), "media/asset.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if !strings.Contains(url.RawQuery, "X-Amz-Expires=300") {
		t.Fatalf("presigned URL must carry the lifetime: %s", url)
	}
	if !strings.Contains(url.RawQuery, "X-Amz-Signature=") {
		t.Fatalf("presigned URL must be signed: %s", url)
	}
}

func TestPresignedGetURLRejectsNonPositiveLifetime(t *testing.T) {
	client := newTestClient(t)
	defer func() {
		if recover() == nil {
			t.Fatal("a non-positive lifetime must be a programmer error")
		}
	}()
	_, _ = client.PresignedGetURL(context.Background(), "media/asset.png", 0)
}

func TestNewRequiresATimeout(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 0
	defer func() {
		if recover() == nil {
			t.Fatal("a missing object storage timeout must fail at startup")
		}
	}()
	New(Dependency{Config: cfg, Logger: applog.NewNoopLogger()})
}
