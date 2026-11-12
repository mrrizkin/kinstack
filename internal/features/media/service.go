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
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/pkg/publicid"
)

const (
	uploadSizeMax     int64 = 10 << 20
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

type ServiceDependency struct {
	Repository Repository
	Storage    MediaStorage
}

type Service struct {
	repository Repository
	storage    MediaStorage
}

func NewService(d ServiceDependency) *Service {
	assert.Assert(d.Repository != nil, "media repository is required")
	assert.Assert(d.Storage != nil, "media storage is required")
	return &Service{repository: d.Repository, storage: d.Storage}
}

// uploadInput is a validated upload: the owner, the name that will be stored,
// and the extension chosen from the detected type.
type uploadInput struct {
	ownerPublicID string
	filename      string
	contentType   string
	extension     string
	size          int64
	body          io.Reader
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
	input, err := validateUpload(ownerPublicID, filename, contentType, size, body)
	if err != nil {
		return AssetDTO{}, err
	}
	head, err := readHead(input.body)
	if err != nil {
		return AssetDTO{}, err
	}
	if !detectedContentTypeMatches(input.contentType, http.DetectContentType(head)) {
		return AssetDTO{}, ErrInvalidInput
	}
	prepared, preparedSize, err := prepareBody(input.body, head, input.contentType, input.size)
	if err != nil {
		return AssetDTO{}, err
	}
	return s.storeUpload(ctx, input, prepared, preparedSize)
}

func validateUpload(
	ownerPublicID, filename, contentType string,
	size int64,
	body io.Reader,
) (uploadInput, error) {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	filename = sanitizeFilename(filename)
	if ownerPublicID == "" || body == nil || filename == "" ||
		size < 1 || size > uploadSizeMax {
		return uploadInput{}, ErrInvalidInput
	}
	extension, ok := allowedContentTypes[contentType]
	if !ok {
		return uploadInput{}, ErrInvalidInput
	}
	return uploadInput{
		ownerPublicID: ownerPublicID,
		filename:      filename,
		contentType:   contentType,
		extension:     extension,
		size:          size,
		body:          body,
	}, nil
}

// readHead reads the magic bytes the type check needs. A short body ends the
// read without failing: the size and type checks decide what happens next.
func readHead(body io.Reader) ([]byte, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(body, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, ErrInvalidInput
	}
	return head[:n], nil
}

// prepareBody makes the body safe for the object store. Images are decoded for
// their dimensions without decoding the pixel payload, which rejects
// decompression bombs. The S3 SDK needs a seekable body for checksums, so the
// reader is rewound when it can be and buffered when it cannot.
func prepareBody(
	body io.Reader,
	head []byte,
	contentType string,
	size int64,
) (io.Reader, int64, error) {
	if strings.HasPrefix(contentType, "image/") && contentType != "image/webp" {
		data, err := readBounded(body, head)
		if err != nil {
			return nil, 0, err
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width < 1 || config.Height < 1 ||
			config.Width > maxImageDimension || config.Height > maxImageDimension {
			return nil, 0, ErrInvalidInput
		}
		return bytes.NewReader(data), int64(len(data)), nil
	}
	if seeker, ok := body.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, 0, ErrInvalidInput
		}
		return body, size, nil
	}
	data, err := readBounded(body, head)
	if err != nil {
		return nil, 0, err
	}
	return bytes.NewReader(data), int64(len(data)), nil
}

// readBounded buffers the head plus the body and refuses anything larger than a
// permitted upload, so a lying Content-Length cannot fill memory.
func readBounded(body io.Reader, head []byte) ([]byte, error) {
	data, err := io.ReadAll(
		io.LimitReader(io.MultiReader(bytes.NewReader(head), body), uploadSizeMax+1),
	)
	if err != nil || int64(len(data)) > uploadSizeMax {
		return nil, ErrInvalidInput
	}
	return data, nil
}

func (s *Service) storeUpload(
	ctx context.Context,
	input uploadInput,
	body io.Reader,
	size int64,
) (AssetDTO, error) {
	publicID := publicid.MustNew()
	objectKey := "media/" + input.ownerPublicID + "/" + publicID + input.extension
	if err := s.storage.Put(ctx, objectKey, body, size, input.contentType); err != nil {
		return AssetDTO{}, fmt.Errorf("%w: %v", ErrStorage, err)
	}
	asset, err := s.repository.Create(ctx, CreateAssetInput{
		PublicID:      publicID,
		OwnerPublicID: input.ownerPublicID,
		Filename:      input.filename,
		ContentType:   input.contentType,
		SizeBytes:     size,
		ObjectKey:     objectKey,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return AssetDTO{}, err
	}
	return s.toDTO(asset), nil
}

// Count returns the number of assets owned by the actor.
func (s *Service) Count(ctx context.Context, ownerPublicID string) (int, error) {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	if ownerPublicID == "" {
		return 0, ErrInvalidInput
	}
	return s.repository.Count(ctx, ownerPublicID)
}

// List returns one page of the actor's active assets, newest first.
func (s *Service) List(
	ctx context.Context,
	ownerPublicID string,
	query pagination.Query,
) (MediaPage, error) {
	ownerPublicID = strings.TrimSpace(ownerPublicID)
	if ownerPublicID == "" {
		return MediaPage{}, ErrInvalidInput
	}
	total, err := s.repository.Count(ctx, ownerPublicID)
	if err != nil {
		return MediaPage{}, err
	}
	query = pagination.ClampPage(query, total)
	assets, err := s.repository.List(ctx, ownerPublicID, query)
	if err != nil {
		return MediaPage{}, err
	}
	items := make([]AssetDTO, 0, len(assets))
	for _, asset := range assets {
		items = append(items, s.toDTO(asset))
	}
	return MediaPage{Assets: items, Page: pagination.NewPage(query, total)}, nil
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
