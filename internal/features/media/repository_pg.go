package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const assetProjection = `
public_id,
owner_public_id,
filename,
content_type,
size_bytes,
object_key,
created_at`

const (
	listAssetsQuery = `
SELECT ` + assetProjection + `
FROM media_assets
WHERE owner_public_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3`

	countAssetsQuery = `
SELECT COUNT(*)
FROM media_assets
WHERE owner_public_id = $1
  AND deleted_at IS NULL`

	createAssetQuery = `
INSERT INTO media_assets (public_id, owner_public_id, filename, content_type, size_bytes, object_key)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + assetProjection

	findAssetQuery = `
SELECT ` + assetProjection + `
FROM media_assets
WHERE owner_public_id = $1
  AND public_id = $2
  AND deleted_at IS NULL
LIMIT 1`

	deleteAssetQuery = `
UPDATE media_assets
SET deleted_at = now()
WHERE owner_public_id = $1
  AND public_id = $2
  AND deleted_at IS NULL
RETURNING ` + assetProjection
)

type SQLRepository struct {
	owner      *platformdatabase.StatementOwner
	listStmt   *sqlx.Stmt
	countStmt  *sqlx.Stmt
	createStmt *sqlx.Stmt
	findStmt   *sqlx.Stmt
	deleteStmt *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sqlx.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "media database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.listStmt, "media_list", listAssetsQuery},
		{&repository.countStmt, "media_count", countAssetsQuery},
		{&repository.createStmt, "media_create", createAssetQuery},
		{&repository.findStmt, "media_find", findAssetQuery},
		{&repository.deleteStmt, "media_delete", deleteAssetQuery},
	}
	for _, statement := range statements {
		stmt, err := owner.Prepare(statement.name, statement.query)
		if err != nil {
			return nil, err
		}
		*statement.target = stmt
	}
	return repository, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

func (r *SQLRepository) List(
	ctx context.Context,
	ownerPublicID string,
	query pagination.Query,
) ([]Asset, error) {
	assets := make([]Asset, 0)
	if err := r.listStmt.SelectContext(
		ctx,
		&assets,
		ownerPublicID,
		query.Limit(),
		query.Offset(),
	); err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	return assets, nil
}

func (r *SQLRepository) Create(ctx context.Context, input CreateAssetInput) (Asset, error) {
	var asset Asset
	if err := r.createStmt.GetContext(
		ctx,
		&asset,
		input.PublicID,
		input.OwnerPublicID,
		input.Filename,
		input.ContentType,
		input.SizeBytes,
		input.ObjectKey,
	); err != nil {
		return Asset{}, fmt.Errorf("create media: %w", err)
	}
	return asset, nil
}

// Count returns the number of active assets owned by the actor.
func (r *SQLRepository) Count(ctx context.Context, ownerPublicID string) (int, error) {
	var count int
	if err := r.countStmt.GetContext(ctx, &count, ownerPublicID); err != nil {
		return 0, fmt.Errorf("count media: %w", err)
	}
	return count, nil
}

func (r *SQLRepository) Find(ctx context.Context, ownerPublicID, publicID string) (Asset, error) {
	var asset Asset
	if err := r.findStmt.GetContext(ctx, &asset, ownerPublicID, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Asset{}, ErrNotFound
		}
		return Asset{}, fmt.Errorf("find media: %w", err)
	}
	return asset, nil
}

func (r *SQLRepository) Delete(ctx context.Context, ownerPublicID, publicID string) (Asset, error) {
	var asset Asset
	if err := r.deleteStmt.GetContext(ctx, &asset, ownerPublicID, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Asset{}, ErrNotFound
		}
		return Asset{}, fmt.Errorf("delete media: %w", err)
	}
	return asset, nil
}
