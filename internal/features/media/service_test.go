package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"
)

type recordingStorage struct {
	objects map[string]int64
	deleted map[string]bool
	failPut bool
}

func newRecordingStorage() *recordingStorage {
	return &recordingStorage{objects: map[string]int64{}, deleted: map[string]bool{}}
}

func (s *recordingStorage) Put(_ context.Context, key string, body io.Reader, size int64, _ string) error {
	if s.failPut {
		return errors.New("storage down")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.objects[key] = int64(len(data))
	if int64(len(data)) != size {
		return errors.New("size mismatch")
	}
	return nil
}

func (s *recordingStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if _, ok := s.objects[key]; !ok {
		return nil, errors.New("object missing")
	}
	return io.NopCloser(strings.NewReader("data")), nil
}

func (s *recordingStorage) Delete(_ context.Context, key string) error {
	s.deleted[key] = true
	delete(s.objects, key)
	return nil
}

func (s *recordingStorage) PublicURL(key string) string {
	return "https://cdn.test/" + key
}

func testPNG(t *testing.T, width int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func newTestService() (*Service, *MemoryRepository, *recordingStorage) {
	repository := NewMemoryRepository()
	storage := newRecordingStorage()
	return NewService(ServiceDependency{Repository: repository, Storage: storage}), repository, storage
}

func TestUploadStoresAsset(t *testing.T) {
	service, _, storage := newTestService()
	data := testPNG(t, 2)
	body := bytes.NewReader(data)
	asset, err := service.Upload(context.Background(), "user-1", "foto.png", "image/png", int64(len(data)), body)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if asset.PublicID == "" || asset.URL != "/media/"+asset.PublicID+"/file" {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	if len(storage.objects) != 1 {
		t.Fatalf("expected one stored object, got %d", len(storage.objects))
	}
}

func TestUploadRejectsUnsupportedType(t *testing.T) {
	service, _, _ := newTestService()
	if _, err := service.Upload(
		context.Background(),
		"user-1",
		"evil.exe",
		"application/x-msdownload",
		4,
		bytes.NewReader([]byte("evil")),
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	service, _, _ := newTestService()
	if _, err := service.Upload(
		context.Background(),
		"user-1",
		"big.pdf",
		"application/pdf",
		maxUploadSize+1,
		bytes.NewReader([]byte("x")),
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestUploadSurfacesStorageFailure(t *testing.T) {
	service, _, storage := newTestService()
	storage.failPut = true
	data := testPNG(t, 2)
	if _, err := service.Upload(
		context.Background(),
		"user-1",
		"foto.png",
		"image/png",
		int64(len(data)),
		bytes.NewReader(data),
	); !errors.Is(err, ErrStorage) {
		t.Fatalf("expected storage error, got %v", err)
	}
}

func TestUploadRejectsImageOverDimensionLimit(t *testing.T) {
	service, _, _ := newTestService()
	data := testPNG(t, maxImageDimension+1)
	if _, err := service.Upload(
		context.Background(),
		"user-1",
		"huge.png",
		"image/png",
		int64(len(data)),
		bytes.NewReader(data),
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestUploadRejectsContentTypeMismatch(t *testing.T) {
	service, _, _ := newTestService()
	body := strings.NewReader("plain text pretending to be png")
	if _, err := service.Upload(
		context.Background(),
		"user-1",
		"fake.png",
		"image/png",
		int64(body.Len()),
		body,
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListScopesToOwner(t *testing.T) {
	service, repository, _ := newTestService()
	SeedDefault(repository, "user-1", "asset-1", "a.png")
	SeedDefault(repository, "user-2", "asset-2", "b.png")

	items, err := service.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].PublicID != "asset-1" {
		t.Fatalf("unexpected list: %#v", items)
	}
}

func TestDeleteRemovesOwnedAsset(t *testing.T) {
	service, repository, storage := newTestService()
	SeedDefault(repository, "user-1", "asset-1", "a.png")

	if err := service.Delete(context.Background(), "user-1", "asset-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repository.Find(context.Background(), "user-1", "asset-1"); !errors.Is(
		err,
		ErrNotFound,
	) {
		t.Fatalf("expected asset to be gone, got %v", err)
	}
	if !storage.deleted["media/user-1/asset-1.png"] {
		t.Fatalf("expected object to be deleted from storage")
	}
}

func TestDeleteRejectsForeignKey(t *testing.T) {
	service, repository, _ := newTestService()
	SeedDefault(repository, "user-1", "asset-1", "a.png")

	if err := service.Delete(context.Background(), "user-2", "asset-1"); !errors.Is(
		err,
		ErrNotFound,
	) {
		t.Fatalf("expected not found for other owner, got %v", err)
	}
}
