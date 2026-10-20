package authz

import (
	"context"
	"errors"
	"kinstack/internal/platform/i18n"
	"time"
)

// Effect is the polarity of a per-user permission override.
type Effect string

const (
	// EffectAllow grants a permission to a single user beyond their role.
	EffectAllow Effect = "allow"
	// EffectDeny revokes a permission from a single user despite their role.
	EffectDeny Effect = "deny"
)

// Valid reports whether the effect is known.
func (e Effect) Valid() bool {
	return e == EffectAllow || e == EffectDeny
}

// Role is an authorization role owned by the access feature.
type Role struct {
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Rank        int       `db:"rank"`
	IsSystem    bool      `db:"is_system"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Override is a per-user permission exception.
type Override struct {
	Permission Permission `db:"permission_code"`
	Effect     Effect     `db:"effect"`
	Reason     string     `db:"reason"`
	ExpiresAt  *time.Time `db:"expires_at"`
}

// Expired reports whether the override is no longer active at now.
func (o Override) Expired(now time.Time) bool {
	return o.ExpiresAt != nil && !o.ExpiresAt.After(now)
}

// CreateRoleInput is the persistence input for a new role.
type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
	Rank        int
}

// UpdateRoleInput is the persistence input for an edited role.
type UpdateRoleInput struct {
	Code        string
	Name        string
	Description string
	Rank        int
}

// SetOverrideInput is the persistence input for an override upsert.
type SetOverrideInput struct {
	UserPublicID string
	Permission   Permission
	Effect       Effect
	Reason       string
	ExpiresAt    *time.Time
}

// Repository is the persistence contract owned by the authz platform package.
type Repository interface {
	ListRoles(context.Context) ([]Role, error)
	FindRole(context.Context, string) (Role, error)
	CreateRole(context.Context, CreateRoleInput) (Role, error)
	UpdateRole(context.Context, UpdateRoleInput) (Role, error)
	DeleteRole(context.Context, string) error

	RolePermissions(context.Context, string) ([]Permission, error)
	ReplaceRolePermissions(context.Context, string, []Permission) error
	CountRoleAssignments(context.Context, string) (int, error)

	UserOverrides(context.Context, string) ([]Override, error)
	SetUserOverride(context.Context, SetOverrideInput) (Override, error)
	DeleteUserOverride(context.Context, string, Permission) error

	AssignUserRole(context.Context, string, string) error
}

// Errors returned by the authz repository and service.
var (
	// ErrRoleNotFound identifies a missing role.
	ErrRoleNotFound = errors.New(i18n.T("authz.reason.role_not_found"))
	// ErrRoleInUse identifies a role still assigned to one or more users.
	ErrRoleInUse = errors.New(i18n.T("authz.reason.role_in_use"))
	// ErrSystemRole identifies a protected built-in role.
	ErrSystemRole = errors.New(i18n.T("authz.reason.system_role"))
	// ErrRoleCodeTaken identifies a duplicate role code.
	ErrRoleCodeTaken = errors.New(i18n.T("authz.reason.role_code_taken"))
	// ErrInvalidRole identifies a malformed role payload.
	ErrInvalidRole = errors.New(i18n.T("authz.reason.invalid_role"))
	// ErrUnknownPermission identifies a permission code outside the catalog.
	ErrUnknownPermission = errors.New(i18n.T("authz.reason.unknown_permission_code"))
	// ErrInvalidOverride identifies a malformed override payload.
	ErrInvalidOverride = errors.New(i18n.T("authz.reason.invalid_override"))
	// ErrUserNotFound identifies a missing user account.
	ErrUserNotFound = errors.New(i18n.T("authz.reason.user_not_found"))
)
