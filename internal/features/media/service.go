package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/publicid"
)

const (
	maxUploadSize     int64 = 10 << 20
	maxFilenameLength       = 200
	maxImageDimension       = 12000
)

// allowedContentTypes is the upload allowlist. Object storage is not a file
// server: only well-known browser-safe formats are accepted.
var allowedContentTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"application/pdf": ".pdf",
	"text/plain":      ".txt",
	"text/csv":        ".csv",
}

// MediaStorage owns immutable object bytes and their delivery URLs.
type MediaStorage interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

// ServiceDependency contains the persistence and object storage boundaries used by the service.
type ServiceDependency struct {
	Repository Repository
	Storage    MediaStorage
}

// Service owns media business rules.
type Service struct {
	repository Repository
	storage    MediaStorage
}

// NewService constructs a media service around its boundaries.
func NewService(d ServiceDependency) *Service {
	assert.Assert(d.Repository != nil, "media repository is required")
	assert.Assert(d.Storage != nil, "media storage is required")
	return &Service{repository: d.Repository, storage: d.Storage}
}

// Upload validates and stores one uploaded file for the owner.
func (s *Service) Upload(
	ctx context.Context,
	ownerPublicID string,
	filename string,
	contentType string,
	size int64,
	body io.Reader,
) (AssetDTO, error) {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	filename = sanitizeFilename(filename)
	if ownerPublicID == "" || body == nil || filename == "" ||
		size < 1 || size > maxUploadSize {
		return AssetDTO{}, ErrInvalidInput
	}
	extension, ok := allowedContentTypes[contentType]
	if !ok {
		return AssetDTO{}, ErrInvalidInput
	}

	// Trust the bytes, not the browser-supplied MIME type. The S3 SDK also
	// needs a seekable body for checksum calculation on plain HTTP, so restore
	// the reader position when possible and fall back to an in-memory buffer.
	head := make([]byte, 512)
	n, readErr := io.ReadFull(body, head)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return AssetDTO{}, ErrInvalidInput
	}
	head = head[:n]
	if !detectedContentTypeMatches(contentType, http.DetectContentType(head)) {
		return AssetDTO{}, ErrInvalidInput
	}

	// Validate image dimensions without decoding the full pixel payload. This
	// rejects decompression bombs while keeping metadata handling out of scope.
	if strings.HasPrefix(contentType, "image/") && contentType != "image/webp" {
		data, err := io.ReadAll(io.LimitReader(io.MultiReader(bytes.NewReader(head), body), maxUploadSize+1))
		if err != nil || int64(len(data)) > maxUploadSize {
			return AssetDTO{}, ErrInvalidInput
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width < 1 || config.Height < 1 ||
			config.Width > maxImageDimension || config.Height > maxImageDimension {
			return AssetDTO{}, ErrInvalidInput
		}
		body = bytes.NewReader(data)
		size = int64(len(data))
	} else if seeker, ok := body.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return AssetDTO{}, ErrInvalidInput
		}
	} else {
		data, err := io.ReadAll(io.LimitReader(io.MultiReader(bytes.NewReader(head), body), maxUploadSize+1))
		if err != nil || int64(len(data)) > maxUploadSize {
			return AssetDTO{}, ErrInvalidInput
		}
		body = bytes.NewReader(data)
	}

	publicID := publicid.MustNew()
	objectKey := "media/" + ownerPublicID + "/" + publicID + extension
	if err := s.storage.Put(ctx, objectKey, body, size, contentType); err != nil {
		return AssetDTO{}, fmt.Errorf("%w: %v", ErrStorage, err)
	}
	asset, err := s.repository.Create(ctx, CreateAssetInput{
		PublicID:      publicID,
		OwnerPublicID: ownerPublicID,
		Filename:      filename,
		ContentType:   contentType,
		SizeBytes:     size,
		ObjectKey:     objectKey,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return AssetDTO{}, err
	}
	return s.toDTO(asset), nil
}

// List returns every active asset owned by the actor, newest first.
func (s *Service) List(ctx context.Context, ownerPublicID string) ([]AssetDTO, error) {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	if ownerPublicID == "" {
		return nil, ErrInvalidInput
	}
	assets, err := s.repository.List(ctx, ownerPublicID)
	if err != nil {
		return nil, err
	}
	items := make([]AssetDTO, 0, len(assets))
	for _, asset := range assets {
		items = append(items, s.toDTO(asset))
	}
	return items, nil
}

// Delete removes one owned asset from storage and marks it deleted.
func (s *Service) Delete(ctx context.Context, ownerPublicID, publicID string) error {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	publicID = strings.TrimSpace(publicID)
	if ownerPublicID == "" || publicID == "" {
		return ErrInvalidInput
	}
	asset, err := s.repository.Find(ctx, ownerPublicID, publicID)
	if err != nil {
		return err
	}
	if err := s.storage.Delete(ctx, asset.ObjectKey); err != nil {
		return fmt.Errorf("%w: %v", ErrStorage, err)
	}
	if _, err := s.repository.Delete(ctx, ownerPublicID, publicID); err != nil {
		return err
	}
	return nil
}

// Open returns one owned asset together with its object body for streaming.
func (s *Service) Open(
	ctx context.Context,
	ownerPublicID, publicID string,
) (Asset, io.ReadCloser, error) {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	publicID = strings.TrimSpace(publicID)
	if ownerPublicID == "" || publicID == "" {
		return Asset{}, nil, ErrInvalidInput
	}
	asset, err := s.repository.Find(ctx, ownerPublicID, publicID)
	if err != nil {
		return Asset{}, nil, err
	}
	body, err := s.storage.Get(ctx, asset.ObjectKey)
	if err != nil {
		return Asset{}, nil, fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return asset, body, nil
}

func (s *Service) toDTO(asset Asset) AssetDTO {
	return AssetDTO{
		PublicID:    asset.PublicID,
		Filename:    asset.Filename,
		ContentType: asset.ContentType,
		Size:        asset.SizeBytes,
		URL:         "/media/" + asset.PublicID + "/file",
		CreatedAt:   asset.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func detectedContentTypeMatches(declared, detected string) bool {
	declared = strings.TrimSpace(strings.SplitN(declared, ";", 2)[0])
	detected = strings.TrimSpace(strings.SplitN(detected, ";", 2)[0])
	if declared == detected {
		return true
	}
	// text/csv is commonly sniffed as text/plain by net/http.
	return (declared == "text/csv" || declared == "text/plain") && detected == "text/plain"
}

func sanitizeFilename(filename string) string {
	filename = strings.TrimSpace(filepath.Base(strings.ReplaceAll(filename, "\\", "/")))
	filename = strings.ReplaceAll(filename, "\x00", "")
	if len(filename) > maxFilenameLength {
		filename = filename[:maxFilenameLength]
	}
	if filename == "." || filename == ".." {
		return ""
	}
	return filename
}
