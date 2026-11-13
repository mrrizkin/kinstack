package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	mediasqlc "kinstack/internal/features/media/sqlc"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
)

type SQLRepository struct {
	queries *mediasqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "media database is required")

	queries, err := mediasqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare media queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) List(
	ctx context.Context,
	ownerPublicID string,
	query pagination.Query,
) ([]Asset, error) {
	rows, err := r.queries.ListAssets(ctx, mediasqlc.ListAssetsParams{
		OwnerPublicID: ownerPublicID,
		RowLimit:      int32(query.Limit()),
		RowOffset:     int32(query.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	assets := make([]Asset, 0, len(rows))
	for _, row := range rows {
		assets = append(assets, assetFromListRow(row))
	}
	return assets, nil
}

func (r *SQLRepository) Create(ctx context.Context, input CreateAssetInput) (Asset, error) {
	row, err := r.queries.CreateAsset(ctx, mediasqlc.CreateAssetParams{
		PublicID:      input.PublicID,
		OwnerPublicID: input.OwnerPublicID,
		Filename:      input.Filename,
		ContentType:   input.ContentType,
		SizeBytes:     input.SizeBytes,
		ObjectKey:     input.ObjectKey,
	})
	if err != nil {
		return Asset{}, fmt.Errorf("create media: %w", err)
	}
	return assetFromCreateRow(row), nil
}

func (r *SQLRepository) Count(ctx context.Context, ownerPublicID string) (int, error) {
	count, err := r.queries.CountAssets(ctx, ownerPublicID)
	if err != nil {
		return 0, fmt.Errorf("count media: %w", err)
	}
	return int(count), nil
}

func (r *SQLRepository) Find(ctx context.Context, ownerPublicID, publicID string) (Asset, error) {
	row, err := r.queries.FindAsset(ctx, mediasqlc.FindAssetParams{
		OwnerPublicID: ownerPublicID,
		PublicID:      publicID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("find media: %w", err)
	}
	return assetFromFindRow(row), nil
}

func (r *SQLRepository) Delete(ctx context.Context, ownerPublicID, publicID string) (Asset, error) {
	row, err := r.queries.SoftDeleteAsset(ctx, mediasqlc.SoftDeleteAssetParams{
		OwnerPublicID: ownerPublicID,
		PublicID:      publicID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("delete media: %w", err)
	}
	return assetFromDeleteRow(row), nil
}

func assetFromListRow(row mediasqlc.ListAssetsRow) Asset {
	return Asset{
		PublicID:      row.PublicID,
		OwnerPublicID: row.OwnerPublicID,
		Filename:      row.Filename,
		ContentType:   row.ContentType,
		SizeBytes:     row.SizeBytes,
		ObjectKey:     row.ObjectKey,
		CreatedAt:     row.CreatedAt,
	}
}

func assetFromCreateRow(row mediasqlc.CreateAssetRow) Asset {
	return Asset{
		PublicID:      row.PublicID,
		OwnerPublicID: row.OwnerPublicID,
		Filename:      row.Filename,
		ContentType:   row.ContentType,
		SizeBytes:     row.SizeBytes,
		ObjectKey:     row.ObjectKey,
		CreatedAt:     row.CreatedAt,
	}
}

func assetFromFindRow(row mediasqlc.FindAssetRow) Asset {
	return Asset{
		PublicID:      row.PublicID,
		OwnerPublicID: row.OwnerPublicID,
		Filename:      row.Filename,
		ContentType:   row.ContentType,
		SizeBytes:     row.SizeBytes,
		ObjectKey:     row.ObjectKey,
		CreatedAt:     row.CreatedAt,
	}
}

func assetFromDeleteRow(row mediasqlc.SoftDeleteAssetRow) Asset {
	return Asset{
		PublicID:      row.PublicID,
		OwnerPublicID: row.OwnerPublicID,
		Filename:      row.Filename,
		ContentType:   row.ContentType,
		SizeBytes:     row.SizeBytes,
		ObjectKey:     row.ObjectKey,
		CreatedAt:     row.CreatedAt,
	}
}
