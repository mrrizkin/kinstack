package passkeys

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SQLRepository is the PostgreSQL implementation of the passkey repository.
type SQLRepository struct {
	owner               *platformdatabase.StatementOwner
	findByPublicIDStmt  *sqlx.Stmt
	listPasskeysStmt    *sqlx.Stmt
	savePasskeyStmt     *sqlx.Stmt
	updatePasskeyStmt   *sqlx.Stmt
	deletePasskeyStmt   *sqlx.Stmt
	findPasskeyUserStmt *sqlx.Stmt
	hasActiveUserStmt   *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to build the repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository prepares regular passkey queries once for the repository lifetime.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "passkey database is required")

	repository := &SQLRepository{owner: platformdatabase.NewStatementOwner(dependency.DB)}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.findByPublicIDStmt, "passkey_find_user", findPasskeyUserByPublicIDQuery},
		{&repository.listPasskeysStmt, "passkey_list", listPasskeysQuery},
		{&repository.savePasskeyStmt, "passkey_save", savePasskeyQuery},
		{&repository.updatePasskeyStmt, "passkey_update", updatePasskeyQuery},
		{&repository.deletePasskeyStmt, "passkey_delete", deletePasskeyQuery},
		{&repository.findPasskeyUserStmt, "passkey_find_login", findPasskeyUserQuery},
		{&repository.hasActiveUserStmt, "passkey_has_active_user", hasActivePasskeyUserQuery},
	}
	for _, statement := range statements {
		stmt, err := repository.owner.Prepare(statement.name, statement.query)
		if err != nil {
			return nil, err
		}
		*statement.target = stmt
	}
	return repository, nil
}

// Close releases the prepared statements owned by the repository.
func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

// FindByPublicID resolves one active account.
func (r *SQLRepository) FindByPublicID(
	ctx context.Context,
	id string,
) (FindPasskeyUserResult, error) {
	var result FindPasskeyUserResult
	if err := r.findByPublicIDStmt.GetContext(ctx, &result, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FindPasskeyUserResult{}, ErrUserNotFound
		}
		return FindPasskeyUserResult{}, fmt.Errorf("find passkey user: %w", err)
	}
	return result, nil
}

// ListPasskeys returns the credentials of one active account.
func (r *SQLRepository) ListPasskeys(ctx context.Context, id string) ([]ListPasskeysResult, error) {
	if _, err := r.FindByPublicID(ctx, id); err != nil {
		return nil, err
	}
	results := make([]ListPasskeysResult, 0)
	if err := r.listPasskeysStmt.SelectContext(ctx, &results, id); err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	return results, nil
}

// SavePasskey stores a new credential.
func (r *SQLRepository) SavePasskey(ctx context.Context, id string, record PasskeyRecord) error {
	record.Name = strings.TrimSpace(record.Name)
	if len(record.ID) == 0 || !bytes.Equal(record.ID, record.Credential.ID) ||
		record.Name == "" || len(record.Name) > maxPasskeyNameLength {
		return ErrPasskeyInvalid
	}
	credential, err := json.Marshal(record.Credential)
	if err != nil {
		return fmt.Errorf("encode passkey credential: %w", err)
	}
	result, err := r.savePasskeyStmt.ExecContext(ctx, id, record.ID, record.Name, credential)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrPasskeyConflict
		}
		return fmt.Errorf("save passkey: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("save passkey result: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdatePasskey stores authenticator state after a successful assertion.
func (r *SQLRepository) UpdatePasskey(ctx context.Context, id string, record PasskeyRecord) error {
	credential, err := json.Marshal(record.Credential)
	if err != nil {
		return fmt.Errorf("encode passkey credential: %w", err)
	}
	result, err := r.updatePasskeyStmt.ExecContext(
		ctx,
		id,
		record.ID,
		credential,
		record.LastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("update passkey: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update passkey result: %w", err)
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

// DeletePasskey removes one credential.
func (r *SQLRepository) DeletePasskey(ctx context.Context, id string, credentialID []byte) error {
	result, err := r.deletePasskeyStmt.ExecContext(ctx, id, credentialID)
	if err != nil {
		return fmt.Errorf("delete passkey: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete passkey result: %w", err)
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
	var active bool
	if err := r.hasActiveUserStmt.GetContext(ctx, &active, id); err != nil {
		return false, fmt.Errorf("check passkey user: %w", err)
	}
	return active, nil
}

// FindPasskeyUser resolves the account selected by a discoverable login.
func (r *SQLRepository) FindPasskeyUser(
	ctx context.Context,
	credentialID, userHandle []byte,
) (FindPasskeyUserResult, []ListPasskeysResult, error) {
	var user FindPasskeyUserResult
	if err := r.findPasskeyUserStmt.GetContext(ctx, &user, credentialID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FindPasskeyUserResult{}, nil, ErrPasskeyNotFound
		}
		return FindPasskeyUserResult{}, nil, fmt.Errorf("find passkey login user: %w", err)
	}
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
	results := make([]ListPasskeysResult, 0)
	if err := r.listPasskeysStmt.SelectContext(ctx, &results, id); err != nil {
		return nil, fmt.Errorf("list passkeys for login: %w", err)
	}
	return results, nil
}

const findPasskeyUserByPublicIDQuery = `
SELECT public_id,email,username,display_name,is_active,deleted_at
FROM users
WHERE public_id=$1 AND deleted_at IS NULL AND is_active
LIMIT 1`

const listPasskeysQuery = `
SELECT p.credential_id,p.name,p.credential,p.created_at,p.last_used_at
FROM auth_passkeys p
JOIN users u ON u.id=p.user_id
WHERE u.public_id=$1 AND u.deleted_at IS NULL AND u.is_active
ORDER BY p.created_at ASC,p.id ASC`

const savePasskeyQuery = `
INSERT INTO auth_passkeys(user_id,credential_id,name,credential)
SELECT u.id,$2,$3,$4::jsonb
FROM users u
WHERE u.public_id=$1 AND u.deleted_at IS NULL AND u.is_active`

const updatePasskeyQuery = `
UPDATE auth_passkeys p
SET credential=$3::jsonb,last_used_at=COALESCE($4,p.last_used_at)
FROM users u
WHERE p.user_id=u.id AND u.public_id=$1 AND p.credential_id=$2
	AND u.deleted_at IS NULL AND u.is_active`

const deletePasskeyQuery = `
DELETE FROM auth_passkeys p
USING users u
WHERE p.user_id=u.id AND u.public_id=$1 AND p.credential_id=$2
	AND u.deleted_at IS NULL AND u.is_active`

const findPasskeyUserQuery = `
SELECT u.public_id,u.email,u.username,u.display_name,u.is_active,u.deleted_at
FROM auth_passkeys p
JOIN users u ON u.id=p.user_id
WHERE p.credential_id=$1 AND u.deleted_at IS NULL AND u.is_active
LIMIT 1`

const hasActivePasskeyUserQuery = `
SELECT EXISTS(
	SELECT 1 FROM users
	WHERE public_id=$1 AND deleted_at IS NULL AND is_active
)`
