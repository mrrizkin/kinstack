package actor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const findActiveActorQuery = `
SELECT 1
FROM users
WHERE public_id = $1
  AND deleted_at IS NULL
  AND is_active
LIMIT 1`

// SQLRepository resolves active actors directly from the users table.
type SQLRepository struct {
	owner    *platformdatabase.StatementOwner
	findStmt *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// NewSQLRepository prepares the actor lookup for the repository lifetime.
type RepositoryDependency struct {
	DB *sqlx.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "actor database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	findStmt, err := owner.Prepare("actor_find_by_public_id", findActiveActorQuery)
	if err != nil {
		return nil, err
	}
	return &SQLRepository{owner: owner, findStmt: findStmt}, nil
}

// Close releases the prepared statement owned by the repository.
func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

func (r *SQLRepository) FindByPublicID(ctx context.Context, publicID string) error {
	if r == nil || r.findStmt == nil {
		return ErrRepositoryUnavailable
	}
	var found int
	err := r.findStmt.GetContext(ctx, &found, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find actor by public ID: %w", err)
	}
	return nil
}
