package authz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const roleColumns = `code, name, description, rank, is_system, created_at, updated_at`

const (
	listRolesQuery = `
SELECT ` + roleColumns + `
FROM roles
ORDER BY rank DESC, code ASC
LIMIT 100`

	countRolesQuery = `
SELECT COUNT(*) FROM roles`

	findRoleQuery = `
SELECT ` + roleColumns + `
FROM roles
WHERE code = $1
LIMIT 1`

	createRoleQuery = `
INSERT INTO roles (code, name, description, rank)
VALUES ($1, $2, $3, $4)
RETURNING ` + roleColumns

	updateRoleQuery = `
UPDATE roles
SET name = $2, description = $3, rank = $4, updated_at = now()
WHERE code = $1
  AND is_system = false
RETURNING ` + roleColumns

	deleteRoleQuery = `DELETE FROM roles WHERE code = $1 AND is_system = false`

	rolePermissionsQuery = `
SELECT permission_code
FROM role_permissions
WHERE role_code = $1
ORDER BY permission_code ASC`

	rolePermissionsManyQuery = `
SELECT role_code, permission_code
FROM role_permissions
WHERE role_code = ANY($1)
ORDER BY role_code ASC, permission_code ASC`

	replaceRolePermissionsQuery = `
WITH target AS (
    SELECT code FROM roles WHERE code = $1 AND is_system = false
),
deleted AS (
    DELETE FROM role_permissions
    WHERE role_code IN (SELECT code FROM target)
)
INSERT INTO role_permissions (role_code, permission_code)
SELECT target.code, unnest($2::text[])
FROM target`

	userOverridesQuery = `
SELECT permission_code, effect, reason, expires_at
FROM user_permissions
WHERE user_public_id = $1
ORDER BY permission_code ASC`

	userOverridesManyQuery = `
SELECT user_public_id, permission_code, effect, reason, expires_at
FROM user_permissions
WHERE user_public_id = ANY($1)
ORDER BY user_public_id ASC, permission_code ASC`

	upsertUserOverrideQuery = `
INSERT INTO user_permissions (user_public_id, permission_code, effect, reason, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_public_id, permission_code)
DO UPDATE SET effect = EXCLUDED.effect,
              reason = EXCLUDED.reason,
              expires_at = EXCLUDED.expires_at,
              updated_at = now()
RETURNING permission_code, effect, reason, expires_at`

	deleteUserOverrideQuery = `
DELETE FROM user_permissions
WHERE user_public_id = $1 AND permission_code = $2`

	assignUserRoleQuery = `
WITH deleted AS (
    DELETE FROM user_roles WHERE user_public_id = $1
)
INSERT INTO user_roles (user_public_id, role_code)
SELECT $1, $2
WHERE EXISTS (
    SELECT 1 FROM users WHERE public_id = $1 AND deleted_at IS NULL
)
RETURNING user_public_id`

	userRoleQuery = `
SELECT role_code
FROM user_roles
WHERE user_public_id = $1
LIMIT 1`

	countRoleAssignmentsQuery = `
SELECT count(DISTINCT ur.user_public_id)
FROM user_roles ur
JOIN users u ON u.public_id = ur.user_public_id
WHERE ur.role_code = $1
  AND u.deleted_at IS NULL`
)

// SQLRepository is the PostgreSQL authorization repository.
type SQLRepository struct {
	owner *platformdatabase.StatementOwner

	listRolesStmt        *sqlx.Stmt
	countRolesStmt       *sqlx.Stmt
	findRoleStmt         *sqlx.Stmt
	createRoleStmt       *sqlx.Stmt
	updateRoleStmt       *sqlx.Stmt
	deleteRoleStmt       *sqlx.Stmt
	rolePermissionsStmt  *sqlx.Stmt
	rolePermsManyStmt    *sqlx.Stmt
	replacePermsStmt     *sqlx.Stmt
	overridesStmt        *sqlx.Stmt
	overridesManyStmt    *sqlx.Stmt
	upsertOverrideStmt   *sqlx.Stmt
	deleteOverrideStmt   *sqlx.Stmt
	assignRoleStmt       *sqlx.Stmt
	userRoleStmt         *sqlx.Stmt
	countAssignmentsStmt *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to construct the SQL repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL authorization repository.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "authz database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.listRolesStmt, "authz_list_roles", listRolesQuery},
		{&repository.countRolesStmt, "authz_count_roles", countRolesQuery},
		{&repository.findRoleStmt, "authz_find_role", findRoleQuery},
		{&repository.createRoleStmt, "authz_create_role", createRoleQuery},
		{&repository.updateRoleStmt, "authz_update_role", updateRoleQuery},
		{&repository.deleteRoleStmt, "authz_delete_role", deleteRoleQuery},
		{&repository.rolePermissionsStmt, "authz_role_permissions", rolePermissionsQuery},
		{&repository.rolePermsManyStmt, "authz_role_permissions_many", rolePermissionsManyQuery},
		{&repository.replacePermsStmt, "authz_replace_role_permissions", replaceRolePermissionsQuery},
		{&repository.overridesStmt, "authz_user_overrides", userOverridesQuery},
		{&repository.overridesManyStmt, "authz_user_overrides_many", userOverridesManyQuery},
		{&repository.upsertOverrideStmt, "authz_upsert_user_override", upsertUserOverrideQuery},
		{&repository.deleteOverrideStmt, "authz_delete_user_override", deleteUserOverrideQuery},
		{&repository.assignRoleStmt, "authz_assign_user_role", assignUserRoleQuery},
		{&repository.userRoleStmt, "authz_user_role", userRoleQuery},
		{&repository.countAssignmentsStmt, "authz_count_role_assignments", countRoleAssignmentsQuery},
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

// Close releases the prepared statements owned by the repository.
func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

// ListRoles returns every role.
func (r *SQLRepository) ListRoles(ctx context.Context) ([]Role, error) {
	roles := make([]Role, 0)
	if err := r.listRolesStmt.SelectContext(ctx, &roles); err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	return roles, nil
}

// CountRoles returns the number of roles.
func (r *SQLRepository) CountRoles(ctx context.Context) (int, error) {
	var count int
	if err := r.countRolesStmt.GetContext(ctx, &count); err != nil {
		return 0, fmt.Errorf("count roles: %w", err)
	}
	return count, nil
}

// FindRole resolves a single role.
func (r *SQLRepository) FindRole(ctx context.Context, code string) (Role, error) {
	var role Role
	err := r.findRoleStmt.GetContext(ctx, &role, code)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, fmt.Errorf("find role: %w", err)
	}
	return role, nil
}

// CreateRole inserts a new role.
func (r *SQLRepository) CreateRole(ctx context.Context, input CreateRoleInput) (Role, error) {
	var role Role
	err := r.createRoleStmt.GetContext(ctx, &role, input.Code, input.Name, input.Description, input.Rank)
	if isUniqueViolation(err) {
		return Role{}, ErrRoleCodeTaken
	}
	if err != nil {
		return Role{}, fmt.Errorf("create role: %w", err)
	}
	return role, nil
}

// UpdateRole edits a role's metadata.
func (r *SQLRepository) UpdateRole(ctx context.Context, input UpdateRoleInput) (Role, error) {
	var role Role
	err := r.updateRoleStmt.GetContext(ctx, &role, input.Code, input.Name, input.Description, input.Rank)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, fmt.Errorf("update role: %w", err)
	}
	return role, nil
}

// DeleteRole removes a role, mapping foreign-key conflicts to ErrRoleInUse.
func (r *SQLRepository) DeleteRole(ctx context.Context, code string) error {
	result, err := r.deleteRoleStmt.ExecContext(ctx, code)
	if isForeignKeyViolation(err) {
		return ErrRoleInUse
	}
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete role rows affected: %w", err)
	}
	if affected == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// RolePermissions returns the permission codes granted to a role.
func (r *SQLRepository) RolePermissions(ctx context.Context, code string) ([]Permission, error) {
	permissions := make([]Permission, 0)
	if err := r.rolePermissionsStmt.SelectContext(ctx, &permissions, code); err != nil {
		return nil, fmt.Errorf("role permissions: %w", err)
	}
	return permissions, nil
}

// rolePermissionRow is the flat projection used by the batched permission query.
type rolePermissionRow struct {
	RoleCode   string     `db:"role_code"`
	Permission Permission `db:"permission_code"`
}

// RolePermissionsMany returns the permission sets of several roles in one query.
func (r *SQLRepository) RolePermissionsMany(
	ctx context.Context,
	codes []string,
) (map[string][]Permission, error) {
	result := make(map[string][]Permission, len(codes))
	if len(codes) == 0 {
		return result, nil
	}
	rows := make([]rolePermissionRow, 0)
	if err := r.rolePermsManyStmt.SelectContext(ctx, &rows, pq.Array(codes)); err != nil {
		return nil, fmt.Errorf("role permissions batch: %w", err)
	}
	for _, row := range rows {
		result[row.RoleCode] = append(result[row.RoleCode], row.Permission)
	}
	return result, nil
}

// ReplaceRolePermissions atomically replaces a role's permission set.
func (r *SQLRepository) ReplaceRolePermissions(
	ctx context.Context,
	code string,
	codes []Permission,
) error {
	values := make([]string, 0, len(codes))
	for _, code := range codes {
		values = append(values, string(code))
	}
	if _, err := r.replacePermsStmt.ExecContext(ctx, code, pq.Array(values)); err != nil {
		return fmt.Errorf("replace role permissions: %w", err)
	}
	return nil
}

// UserOverrides returns the overrides recorded for a user.
func (r *SQLRepository) UserOverrides(ctx context.Context, publicID string) ([]Override, error) {
	overrides := make([]Override, 0)
	if err := r.overridesStmt.SelectContext(ctx, &overrides, publicID); err != nil {
		return nil, fmt.Errorf("user overrides: %w", err)
	}
	return overrides, nil
}

// userOverrideRow is the flat projection used by the batched override query.
type userOverrideRow struct {
	UserPublicID string     `db:"user_public_id"`
	Permission   Permission `db:"permission_code"`
	Effect       Effect     `db:"effect"`
	Reason       string     `db:"reason"`
	ExpiresAt    *time.Time `db:"expires_at"`
}

// UserOverridesMany returns overrides for every requested user in one query.
// Users without overrides are simply absent from the result map.
func (r *SQLRepository) UserOverridesMany(
	ctx context.Context,
	publicIDs []string,
) (map[string][]Override, error) {
	result := make(map[string][]Override, len(publicIDs))
	if len(publicIDs) == 0 {
		return result, nil
	}
	rows := make([]userOverrideRow, 0)
	if err := r.overridesManyStmt.SelectContext(ctx, &rows, pq.Array(publicIDs)); err != nil {
		return nil, fmt.Errorf("user overrides batch: %w", err)
	}
	for _, row := range rows {
		result[row.UserPublicID] = append(result[row.UserPublicID], Override{
			Permission: row.Permission,
			Effect:     row.Effect,
			Reason:     row.Reason,
			ExpiresAt:  row.ExpiresAt,
		})
	}
	return result, nil
}

// SetUserOverride upserts a per-user permission override.
func (r *SQLRepository) SetUserOverride(ctx context.Context, input SetOverrideInput) (Override, error) {
	var override Override
	err := r.upsertOverrideStmt.GetContext(
		ctx,
		&override,
		input.UserPublicID,
		input.Permission,
		input.Effect,
		input.Reason,
		input.ExpiresAt,
	)
	if isForeignKeyViolation(err) {
		return Override{}, ErrUserNotFound
	}
	if err != nil {
		return Override{}, fmt.Errorf("set user override: %w", err)
	}
	return override, nil
}

// DeleteUserOverride removes a per-user permission override.
func (r *SQLRepository) DeleteUserOverride(ctx context.Context, publicID string, permission Permission) error {
	if _, err := r.deleteOverrideStmt.ExecContext(ctx, publicID, permission); err != nil {
		return fmt.Errorf("delete user override: %w", err)
	}
	return nil
}

// UserRole returns the role assigned to one user, or "" when unassigned.
func (r *SQLRepository) UserRole(ctx context.Context, publicID string) (string, error) {
	var role string
	if err := r.userRoleStmt.GetContext(ctx, &role, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("find user role: %w", err)
	}
	return role, nil
}

// AssignUserRole changes a user's role.
func (r *SQLRepository) AssignUserRole(ctx context.Context, publicID string, role string) error {
	result, err := r.assignRoleStmt.ExecContext(ctx, publicID, role)
	if err != nil {
		return fmt.Errorf("assign user role: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("assign user role rows affected: %w", err)
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CountRoleAssignments counts non-deleted users holding a role.
func (r *SQLRepository) CountRoleAssignments(ctx context.Context, code string) (int, error) {
	var count int
	if err := r.countAssignmentsStmt.GetContext(ctx, &count, code); err != nil {
		return 0, fmt.Errorf("count role assignments: %w", err)
	}
	return count, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}
