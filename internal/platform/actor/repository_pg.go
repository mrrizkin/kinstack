package actor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	actorsqlc "kinstack/internal/platform/actor/sqlc"
)

type SQLRepository struct {
	queries *actorsqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "actor database is required")

	queries, err := actorsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare actor queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) FindByPublicID(ctx context.Context, publicID string) error {
	if r == nil || r.queries == nil {
		return ErrRepositoryUnavailable
	}
	if _, err := r.queries.FindActiveActor(ctx, publicID); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("find actor by public ID: %w", err)
	}
	return nil
}
