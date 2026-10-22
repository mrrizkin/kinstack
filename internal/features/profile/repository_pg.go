package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const profileProjection = `
u.public_id,
u.username,
u.display_name,
COALESCE((
    SELECT ur.role_code
    FROM user_roles ur
    JOIN roles r ON r.code = ur.role_code
    WHERE ur.user_public_id = u.public_id
    ORDER BY r.rank DESC, ur.role_code ASC
    LIMIT 1
), '') AS role,
COALESCE(p.theme_preference, 'system') AS theme_preference,
COALESCE(p.sound_enabled, true) AS sound_enabled,
COALESCE(p.page_size, 5) AS page_size,
(u.invited_at IS NOT NULL) AS invited,
u.last_login_at,
u.created_at`

const (
	findProfileQuery = `
SELECT ` + profileProjection + `
FROM users u
LEFT JOIN user_preferences p ON p.user_id = u.id
WHERE u.public_id = $1
  AND u.deleted_at IS NULL
LIMIT 1`

	findPasswordHashQuery = `
SELECT password_hash
FROM users
WHERE public_id = $1
  AND deleted_at IS NULL
LIMIT 1`

	findPageSizeQuery = `
SELECT COALESCE(p.page_size, 5) AS page_size
FROM users u
LEFT JOIN user_preferences p ON p.user_id = u.id
WHERE u.public_id = $1
  AND u.deleted_at IS NULL
LIMIT 1`

	updateDisplayNameQuery = `
WITH updated AS (
    UPDATE users
    SET display_name = $2,
        updated_at = now()
    WHERE public_id = $1
      AND deleted_at IS NULL
    RETURNING *
)
SELECT ` + profileProjection + `
FROM updated u
LEFT JOIN user_preferences p ON p.user_id = u.id`

	updatePreferencesQuery = `
WITH target AS (
    SELECT id
    FROM users
    WHERE public_id = $1
      AND deleted_at IS NULL
)
INSERT INTO user_preferences (user_id, theme_preference, sound_enabled, page_size, updated_at)
SELECT id, $2, $3, $4, now()
FROM target
ON CONFLICT (user_id) DO UPDATE
SET theme_preference = EXCLUDED.theme_preference,
    sound_enabled = EXCLUDED.sound_enabled,
    page_size = EXCLUDED.page_size,
    updated_at = now()
RETURNING theme_preference, sound_enabled, page_size`

	markPasswordSetQuery = `
UPDATE users
SET password_hash = $2,
    invited_at = NULL,
    updated_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`
)

type SQLRepository struct {
	owner        *platformdatabase.StatementOwner
	findProfile  *sqlx.Stmt
	findPassword *sqlx.Stmt
	findPageSize *sqlx.Stmt
	updateName   *sqlx.Stmt
	updatePrefs  *sqlx.Stmt
	markPassword *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sqlx.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "profile database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.findProfile, "profile_find", findProfileQuery},
		{&repository.findPassword, "profile_find_password", findPasswordHashQuery},
		{&repository.findPageSize, "profile_find_page_size", findPageSizeQuery},
		{&repository.updateName, "profile_update_name", updateDisplayNameQuery},
		{&repository.updatePrefs, "profile_update_preferences", updatePreferencesQuery},
		{&repository.markPassword, "profile_mark_password_set", markPasswordSetQuery},
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

func (r *SQLRepository) FindProfile(ctx context.Context, publicID string) (ProfileResult, error) {
	var result ProfileResult
	if err := r.findProfile.GetContext(ctx, &result, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileResult{}, ErrNotFound
		}
		return ProfileResult{}, fmt.Errorf("find profile: %w", err)
	}
	return result, nil
}

func (r *SQLRepository) FindPasswordHash(ctx context.Context, publicID string) (string, error) {
	var hash string
	if err := r.findPassword.GetContext(ctx, &hash, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("find password hash: %w", err)
	}
	return hash, nil
}

func (r *SQLRepository) UpdateDisplayName(
	ctx context.Context,
	publicID string,
	displayName string,
) (ProfileResult, error) {
	var result ProfileResult
	if err := r.updateName.GetContext(ctx, &result, publicID, displayName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileResult{}, ErrNotFound
		}
		return ProfileResult{}, fmt.Errorf("update display name: %w", err)
	}
	return result, nil
}

func (r *SQLRepository) FindPageSize(ctx context.Context, publicID string) (int, error) {
	var size int
	if err := r.findPageSize.GetContext(ctx, &size, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("find page size: %w", err)
	}
	return size, nil
}

func (r *SQLRepository) UpdatePreferences(
	ctx context.Context,
	publicID string,
	update PreferencesUpdate,
) (PreferencesResult, error) {
	var result PreferencesResult
	if err := r.updatePrefs.GetContext(
		ctx,
		&result,
		publicID,
		update.ThemePreference,
		update.SoundEnabled,
		update.PageSize,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreferencesResult{}, ErrNotFound
		}
		return PreferencesResult{}, fmt.Errorf("update preferences: %w", err)
	}
	return result, nil
}

// MarkPasswordSet stores a new password hash and clears the invited marker so
// an invited account becomes a regular account once it owns its password.
func (r *SQLRepository) MarkPasswordSet(
	ctx context.Context,
	publicID string,
	hash string,
) error {
	result, err := r.markPassword.ExecContext(ctx, publicID, hash)
	if err != nil {
		return fmt.Errorf("mark password set: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark password set rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
