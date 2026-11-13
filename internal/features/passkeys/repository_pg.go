package passkeys

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	passkeyssqlc "kinstack/internal/features/passkeys/sqlc"
	"kinstack/internal/pkg/assert"

	"github.com/lib/pq"
)

type SQLRepository struct {
	queries *passkeyssqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "passkey database is required")

	queries, err := passkeyssqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare passkey queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) FindByPublicID(
	ctx context.Context,
	id string,
) (FindPasskeyUserResult, error) {
	row, err := r.queries.FindPasskeyUserByPublicID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return FindPasskeyUserResult{}, ErrUserNotFound
	}
	if err != nil {
		return FindPasskeyUserResult{}, fmt.Errorf("find passkey user: %w", err)
	}
	return findPasskeyUserResult(row), nil
}

func (r *SQLRepository) ListPasskeys(ctx context.Context, id string) ([]ListPasskeysResult, error) {
	if _, err := r.FindByPublicID(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.queries.ListPasskeys(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	return listPasskeysResults(rows), nil
}

func (r *SQLRepository) SavePasskey(ctx context.Context, id string, record PasskeyRecord) error {
	record.Name = strings.TrimSpace(record.Name)
	if len(record.ID) == 0 || !bytes.Equal(record.ID, record.Credential.ID) ||
		record.Name == "" || len(record.Name) > passkeyNameLengthMax {
		return ErrPasskeyInvalid
	}
	credential, err := json.Marshal(record.Credential)
	if err != nil {
		return fmt.Errorf("encode passkey credential: %w", err)
	}
	rows, err := r.queries.SavePasskey(ctx, passkeyssqlc.SavePasskeyParams{
		PublicID:     id,
		CredentialID: record.ID,
		Name:         record.Name,
		Credential:   credential,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrPasskeyConflict
		}
		return fmt.Errorf("save passkey: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *SQLRepository) UpdatePasskey(ctx context.Context, id string, record PasskeyRecord) error {
	credential, err := json.Marshal(record.Credential)
	if err != nil {
		return fmt.Errorf("encode passkey credential: %w", err)
	}
	rows, err := r.queries.UpdatePasskey(ctx, passkeyssqlc.UpdatePasskeyParams{
		PublicID:     id,
		CredentialID: record.ID,
		Credential:   credential,
		LastUsedAt:   nullableTimeArg(record.LastUsedAt),
	})
	if err != nil {
		return fmt.Errorf("update passkey: %w", err)
	}
	if rows == 0 {
		active, err := r.hasActiveUser(ctx, id)
		if err != nil {
			return err
		}
		if !active {
			return ErrUserNotFound
		}
		return ErrPasskeyNotFound
	}
	return nil
}

func (r *SQLRepository) DeletePasskey(ctx context.Context, id string, credentialID []byte) error {
	rows, err := r.queries.DeletePasskey(ctx, passkeyssqlc.DeletePasskeyParams{
		PublicID:     id,
		CredentialID: credentialID,
	})
	if err != nil {
		return fmt.Errorf("delete passkey: %w", err)
	}
	if rows == 0 {
		active, err := r.hasActiveUser(ctx, id)
		if err != nil {
			return err
		}
		if !active {
			return ErrUserNotFound
		}
		return ErrPasskeyNotFound
	}
	return nil
}

func (r *SQLRepository) hasActiveUser(ctx context.Context, id string) (bool, error) {
	active, err := r.queries.HasActivePasskeyUser(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check passkey user: %w", err)
	}
	return active, nil
}

func (r *SQLRepository) FindPasskeyUser(
	ctx context.Context,
	credentialID, userHandle []byte,
) (FindPasskeyUserResult, []ListPasskeysResult, error) {
	row, err := r.queries.FindPasskeyLoginUser(ctx, credentialID)
	if errors.Is(err, sql.ErrNoRows) {
		return FindPasskeyUserResult{}, nil, ErrPasskeyNotFound
	}
	if err != nil {
		return FindPasskeyUserResult{}, nil, fmt.Errorf("find passkey login user: %w", err)
	}
	user := findPasskeyLoginUserResult(row)
	if !bytes.Equal(passkeyUserHandle(user.PublicID), userHandle) {
		return FindPasskeyUserResult{}, nil, ErrPasskeyNotFound
	}
	results, err := r.listPasskeys(ctx, user.PublicID)
	if err != nil {
		return FindPasskeyUserResult{}, nil, err
	}
	return user, results, nil
}

func (r *SQLRepository) listPasskeys(ctx context.Context, id string) ([]ListPasskeysResult, error) {
	rows, err := r.queries.ListPasskeys(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list passkeys for login: %w", err)
	}
	return listPasskeysResults(rows), nil
}

func findPasskeyUserResult(row passkeyssqlc.FindPasskeyUserByPublicIDRow) FindPasskeyUserResult {
	return FindPasskeyUserResult{
		PublicID:    row.PublicID,
		Email:       row.Email,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		IsActive:    row.IsActive,
		DeletedAt:   nullableTime(row.DeletedAt),
	}
}

func findPasskeyLoginUserResult(row passkeyssqlc.FindPasskeyLoginUserRow) FindPasskeyUserResult {
	return FindPasskeyUserResult{
		PublicID:    row.PublicID,
		Email:       row.Email,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		IsActive:    row.IsActive,
		DeletedAt:   nullableTime(row.DeletedAt),
	}
}

func listPasskeysResults(rows []passkeyssqlc.ListPasskeysRow) []ListPasskeysResult {
	results := make([]ListPasskeysResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, ListPasskeysResult{
			ID:         row.CredentialID,
			Name:       row.Name,
			Credential: row.Credential,
			CreatedAt:  row.CreatedAt,
			LastUsedAt: nullableTime(row.LastUsedAt),
		})
	}
	return results
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullableTimeArg(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}
