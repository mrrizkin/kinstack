package authz

import (
	"context"
	"errors"
	"time"

	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/i18n"
)

type Effect string

const (
	// EffectAllow grants a permission to a single user beyond their role.
	EffectAllow Effect = "allow"
	// EffectDeny revokes a permission from a single user despite their role.
	EffectDeny Effect = "deny"
)

func (e Effect) Valid() bool {
	return e == EffectAllow || e == EffectDeny
}

type Role struct {
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Rank        int       `db:"rank"`
	IsSystem    bool      `db:"is_system"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Override struct {
	Permission Permission `db:"permission_code"`
	Effect     Effect     `db:"effect"`
	Reason     string     `db:"reason"`
	ExpiresAt  *time.Time `db:"expires_at"`
}

func (o Override) Expired(now time.Time) bool {
	return o.ExpiresAt != nil && !o.ExpiresAt.After(now)
}

type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
	Rank        int
}

type UpdateRoleInput struct {
	Code        string
	Name        string
	Description string
	Rank        int
}

type SetOverrideInput struct {
	UserPublicID string
	Permission   Permission
	Effect       Effect
	Reason       string
	ExpiresAt    *time.Time
}

type Repository interface {
	ListRoles(context.Context, pagination.Query) ([]Role, error)
	CountRoles(context.Context) (int, error)
	FindRole(context.Context, string) (Role, error)
	CreateRole(context.Context, CreateRoleInput) (Role, error)
	UpdateRole(context.Context, UpdateRoleInput) (Role, error)
	DeleteRole(context.Context, string) error

	RolePermissions(context.Context, string) ([]Permission, error)
	// RolePermissionsMany loads the permission sets of several roles in one
	// query so list views do not issue a query per role.
	RolePermissionsMany(context.Context, []string) (map[string][]Permission, error)
	ReplaceRolePermissions(context.Context, string, []Permission) error
	CountRoleAssignments(context.Context, string) (int, error)

	UserOverrides(context.Context, string) ([]Override, error)
	// UserOverridesMany loads overrides for several users in one query so list
	// views do not issue a query per row.
	UserOverridesMany(context.Context, []string) (map[string][]Override, error)
	SetUserOverride(context.Context, SetOverrideInput) (Override, error)
	DeleteUserOverride(context.Context, string, Permission) error

	AssignUserRole(context.Context, string, string) error
	UserRole(context.Context, string) (string, error)
}

var (
	ErrRoleNotFound      = errors.New(i18n.T("authz.reason.role_not_found"))
	ErrRoleInUse         = errors.New(i18n.T("authz.reason.role_in_use"))
	ErrSystemRole        = errors.New(i18n.T("authz.reason.system_role"))
	ErrRoleCodeTaken     = errors.New(i18n.T("authz.reason.role_code_taken"))
	ErrInvalidRole       = errors.New(i18n.T("authz.reason.invalid_role"))
	ErrUnknownPermission = errors.New(i18n.T("authz.reason.unknown_permission_code"))
	ErrInvalidOverride   = errors.New(i18n.T("authz.reason.invalid_override"))
	ErrUserNotFound      = errors.New(i18n.T("authz.reason.user_not_found"))
)
