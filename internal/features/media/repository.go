package media

import (
	"context"
	"time"

	"kinstack/internal/pkg/pagination"
)

type Asset struct {
	PublicID      string    `db:"public_id"`
	OwnerPublicID string    `db:"owner_public_id"`
	Filename      string    `db:"filename"`
	ContentType   string    `db:"content_type"`
	SizeBytes     int64     `db:"size_bytes"`
	ObjectKey     string    `db:"object_key"`
	CreatedAt     time.Time `db:"created_at"`
}

type CreateAssetInput struct {
	PublicID      string
	OwnerPublicID string
	Filename      string
	ContentType   string
	SizeBytes     int64
	ObjectKey     string
}

type Repository interface {
	List(context.Context, string, pagination.Query) ([]Asset, error)
	Count(context.Context, string) (int, error)
	Create(context.Context, CreateAssetInput) (Asset, error)
	Find(context.Context, string, string) (Asset, error)
	Delete(context.Context, string, string) (Asset, error)
}
