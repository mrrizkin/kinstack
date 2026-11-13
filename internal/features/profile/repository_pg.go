package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	profilesqlc "kinstack/internal/features/profile/sqlc"
	"kinstack/internal/pkg/assert"
)

type SQLRepository struct {
	queries *profilesqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "profile database is required")

	queries, err := profilesqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare profile queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) FindProfile(ctx context.Context, publicID string) (ProfileResult, error) {
	row, err := r.queries.FindProfile(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return ProfileResult{}, ErrNotFound
	}
	if err != nil {
		return ProfileResult{}, fmt.Errorf("find profile: %w", err)
	}
	return profileFromFindRow(row), nil
}

func (r *SQLRepository) FindPasswordHash(ctx context.Context, publicID string) (string, error) {
	hash, err := r.queries.FindPasswordHash(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find password hash: %w", err)
	}
	return hash, nil
}

func (r *SQLRepository) UpdateDisplayName(
	ctx context.Context,
	publicID string,
	displayName string,
) (ProfileResult, error) {
	row, err := r.queries.UpdateDisplayName(ctx, profilesqlc.UpdateDisplayNameParams{
		PublicID:    publicID,
		DisplayName: displayName,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ProfileResult{}, ErrNotFound
	}
	if err != nil {
		return ProfileResult{}, fmt.Errorf("update display name: %w", err)
	}
	return profileFromUpdateRow(row), nil
}

func (r *SQLRepository) FindPageSize(ctx context.Context, publicID string) (int, error) {
	size, err := r.queries.FindPageSize(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("find page size: %w", err)
	}
	return int(size), nil
}

func (r *SQLRepository) UpdatePreferences(
	ctx context.Context,
	publicID string,
	update PreferencesUpdate,
) (PreferencesResult, error) {
	row, err := r.queries.UpdatePreferences(ctx, profilesqlc.UpdatePreferencesParams{
		PublicID:        publicID,
		ThemePreference: update.ThemePreference,
		SoundEnabled:    update.SoundEnabled,
		PageSize:        int32(update.PageSize),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return PreferencesResult{}, ErrNotFound
	}
	if err != nil {
		return PreferencesResult{}, fmt.Errorf("update preferences: %w", err)
	}
	return PreferencesResult{
		ThemePreference: row.ThemePreference,
		SoundEnabled:    row.SoundEnabled,
		PageSize:        int(row.PageSize),
	}, nil
}

// MarkPasswordSet stores a new password hash and clears the invited marker so
// an invited account becomes a regular account once it owns its password.
func (r *SQLRepository) MarkPasswordSet(
	ctx context.Context,
	publicID string,
	hash string,
) error {
	affected, err := r.queries.MarkPasswordSet(ctx, profilesqlc.MarkPasswordSetParams{
		PublicID:     publicID,
		PasswordHash: hash,
	})
	if err != nil {
		return fmt.Errorf("mark password set: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func profileFromFindRow(row profilesqlc.FindProfileRow) ProfileResult {
	return ProfileResult{
		PublicID:        row.PublicID,
		Username:        row.Username,
		DisplayName:     row.DisplayName,
		Role:            row.Role,
		Invited:         row.Invited,
		ThemePreference: row.ThemePreference,
		SoundEnabled:    row.SoundEnabled,
		PageSize:        int(row.PageSize),
		LastLoginAt:     nullableTime(row.LastLoginAt),
		CreatedAt:       row.CreatedAt,
	}
}

func profileFromUpdateRow(row profilesqlc.UpdateDisplayNameRow) ProfileResult {
	return ProfileResult{
		PublicID:        row.PublicID,
		Username:        row.Username,
		DisplayName:     row.DisplayName,
		Role:            row.Role,
		Invited:         row.Invited,
		ThemePreference: row.ThemePreference,
		SoundEnabled:    row.SoundEnabled,
		PageSize:        int(row.PageSize),
		LastLoginAt:     nullableTime(row.LastLoginAt),
		CreatedAt:       row.CreatedAt,
	}
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
