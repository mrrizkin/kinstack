package authz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
	authzsqlc "kinstack/internal/platform/authz/sqlc"

	"github.com/lib/pq"
)

type SQLRepository struct {
	queries *authzsqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "authz database is required")

	queries, err := authzsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare authz queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) ListRoles(ctx context.Context, query pagination.Query) ([]Role, error) {
	rows, err := r.queries.ListRoles(ctx, authzsqlc.ListRolesParams{
		RowLimit:  int32(query.Limit()),
		RowOffset: int32(query.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, roleFromRow(row))
	}
	return roles, nil
}

func (r *SQLRepository) CountRoles(ctx context.Context) (int, error) {
	count, err := r.queries.CountRoles(ctx)
	if err != nil {
		return 0, fmt.Errorf("count roles: %w", err)
	}
	return int(count), nil
}

func (r *SQLRepository) FindRole(ctx context.Context, code string) (Role, error) {
	row, err := r.queries.FindRole(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, fmt.Errorf("find role: %w", err)
	}
	return roleFromRow(row), nil
}

func (r *SQLRepository) CreateRole(ctx context.Context, input CreateRoleInput) (Role, error) {
	row, err := r.queries.CreateRole(ctx, authzsqlc.CreateRoleParams{
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Rank:        int32(input.Rank),
	})
	if isUniqueViolation(err) {
		return Role{}, ErrRoleCodeTaken
	}
	if err != nil {
		return Role{}, fmt.Errorf("create role: %w", err)
	}
	return roleFromRow(row), nil
}

func (r *SQLRepository) UpdateRole(ctx context.Context, input UpdateRoleInput) (Role, error) {
	row, err := r.queries.UpdateRole(ctx, authzsqlc.UpdateRoleParams{
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Rank:        int32(input.Rank),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, fmt.Errorf("update role: %w", err)
	}
	return roleFromRow(row), nil
}

func (r *SQLRepository) DeleteRole(ctx context.Context, code string) error {
	affected, err := r.queries.DeleteRole(ctx, code)
	if isForeignKeyViolation(err) {
		return ErrRoleInUse
	}
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if affected == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func (r *SQLRepository) RolePermissions(ctx context.Context, code string) ([]Permission, error) {
	codes, err := r.queries.RolePermissions(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("role permissions: %w", err)
	}
	return permissions(codes), nil
}

func (r *SQLRepository) RolePermissionsMany(
	ctx context.Context,
	codes []string,
) (map[string][]Permission, error) {
	result := make(map[string][]Permission, len(codes))
	if len(codes) == 0 {
		return result, nil
	}
	rows, err := r.queries.RolePermissionsMany(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("role permissions batch: %w", err)
	}
	for _, row := range rows {
		result[row.RoleCode] = append(result[row.RoleCode], Permission(row.PermissionCode))
	}
	return result, nil
}

func (r *SQLRepository) ReplaceRolePermissions(
	ctx context.Context,
	code string,
	codes []Permission,
) error {
	values := make([]string, 0, len(codes))
	for _, code := range codes {
		values = append(values, string(code))
	}
	if err := r.queries.ReplaceRolePermissions(ctx, authzsqlc.ReplaceRolePermissionsParams{
		Code:            code,
		PermissionCodes: values,
	}); err != nil {
		return fmt.Errorf("replace role permissions: %w", err)
	}
	return nil
}

func (r *SQLRepository) UserOverrides(ctx context.Context, publicID string) ([]Override, error) {
	rows, err := r.queries.UserOverrides(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("user overrides: %w", err)
	}
	overrides := make([]Override, 0, len(rows))
	for _, row := range rows {
		overrides = append(overrides, Override{
			Permission: Permission(row.PermissionCode),
			Effect:     Effect(row.Effect),
			Reason:     row.Reason,
			ExpiresAt:  nullableTime(row.ExpiresAt),
		})
	}
	return overrides, nil
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
	rows, err := r.queries.UserOverridesMany(ctx, publicIDs)
	if err != nil {
		return nil, fmt.Errorf("user overrides batch: %w", err)
	}
	for _, row := range rows {
		result[row.UserPublicID] = append(result[row.UserPublicID], Override{
			Permission: Permission(row.PermissionCode),
			Effect:     Effect(row.Effect),
			Reason:     row.Reason,
			ExpiresAt:  nullableTime(row.ExpiresAt),
		})
	}
	return result, nil
}

func (r *SQLRepository) SetUserOverride(
	ctx context.Context,
	input SetOverrideInput,
) (Override, error) {
	row, err := r.queries.UpsertUserOverride(ctx, authzsqlc.UpsertUserOverrideParams{
		UserPublicID:   input.UserPublicID,
		PermissionCode: string(input.Permission),
		Effect:         string(input.Effect),
		Reason:         input.Reason,
		ExpiresAt:      nullableTimeArg(input.ExpiresAt),
	})
	if isForeignKeyViolation(err) {
		return Override{}, ErrUserNotFound
	}
	if err != nil {
		return Override{}, fmt.Errorf("set user override: %w", err)
	}
	return Override{
		Permission: Permission(row.PermissionCode),
		Effect:     Effect(row.Effect),
		Reason:     row.Reason,
		ExpiresAt:  nullableTime(row.ExpiresAt),
	}, nil
}

func (r *SQLRepository) DeleteUserOverride(
	ctx context.Context,
	publicID string,
	permission Permission,
) error {
	if err := r.queries.DeleteUserOverride(ctx, authzsqlc.DeleteUserOverrideParams{
		UserPublicID:   publicID,
		PermissionCode: string(permission),
	}); err != nil {
		return fmt.Errorf("delete user override: %w", err)
	}
	return nil
}

func (r *SQLRepository) UserRole(ctx context.Context, publicID string) (string, error) {
	role, err := r.queries.UserRole(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find user role: %w", err)
	}
	return role, nil
}

func (r *SQLRepository) AssignUserRole(ctx context.Context, publicID string, role string) error {
	affected, err := r.queries.AssignUserRole(ctx, authzsqlc.AssignUserRoleParams{
		UserPublicID: publicID,
		RoleCode:     role,
	})
	if err != nil {
		return fmt.Errorf("assign user role: %w", err)
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *SQLRepository) CountRoleAssignments(ctx context.Context, code string) (int, error) {
	count, err := r.queries.CountRoleAssignments(ctx, code)
	if err != nil {
		return 0, fmt.Errorf("count role assignments: %w", err)
	}
	return int(count), nil
}

func roleFromRow(row authzsqlc.Role) Role {
	return Role{
		Code:        row.Code,
		Name:        row.Name,
		Description: row.Description,
		Rank:        int(row.Rank),
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func permissions(codes []string) []Permission {
	values := make([]Permission, 0, len(codes))
	for _, code := range codes {
		values = append(values, Permission(code))
	}
	return values
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

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}
