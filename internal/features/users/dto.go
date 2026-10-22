package users

import "kinstack/internal/pkg/pagination"

// UserDTO is the public contract for one user account.
type UserDTO struct {
	PublicID    string `json:"public_id"`
	Username    string `json:"username"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsActive    bool   `json:"is_active"`
	Invited     bool   `json:"invited"`
	LastLoginAt string `json:"last_login_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// CreateUserRequest is the payload accepted when creating an account.
type CreateUserRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

// InviteUserRequest is the payload accepted when inviting a new account.
type InviteUserRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// UpdateUserRequest is the payload accepted when editing an account.
type UpdateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// ResetPasswordRequest is the payload accepted when resetting a password.
type ResetPasswordRequest struct {
	Password string `json:"password"`
}

// SetActiveRequest is the payload accepted when enabling or disabling an account.
type SetActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// UsersPageProps is the public contract for the user management page.
type UsersPageProps struct {
	Users []UserDTO `json:"users"`
	Roles []RoleDTO `json:"roles"`
}

// RoleDTO is the public contract for one authorization role.
type RoleDTO struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rank        int      `json:"rank"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
}

// PermissionDTO is one catalog permission shown in the access matrix.
type PermissionDTO struct {
	Code        string `json:"code"`
	Category    string `json:"category"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// OverrideDTO is a per-user permission exception.
type OverrideDTO struct {
	Permission string `json:"permission"`
	Label      string `json:"label"`
	Effect     string `json:"effect"`
	Reason     string `json:"reason"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Expired    bool   `json:"expired"`
}

// UserAccessDTO is a user account with its authorization state.
type UserAccessDTO struct {
	PublicID    string        `json:"public_id"`
	Username    string        `json:"username"`
	DisplayName string        `json:"display_name"`
	Role        string        `json:"role"`
	RoleName    string        `json:"role_name"`
	IsActive    bool          `json:"is_active"`
	Overrides   []OverrideDTO `json:"overrides"`
}

// AccessPageProps is the public contract for the access management page.
type AccessPageProps struct {
	Roles       []RoleDTO       `json:"roles"`
	RolesPage   pagination.Page `json:"roles_page"`
	Permissions []PermissionDTO `json:"permissions"`
	Categories  []string        `json:"categories"`
	Users       []UserAccessDTO `json:"users"`
	Page        pagination.Page `json:"page"`
}

// CreateRoleRequest is the payload accepted when creating a role.
type CreateRoleRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Rank        int    `json:"rank"`
}

// UpdateRoleRequest is the payload accepted when editing a role.
type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Rank        int    `json:"rank"`
}

// SetRolePermissionsRequest replaces a role's permission set.
type SetRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// AssignRoleRequest changes a user's role.
type AssignRoleRequest struct {
	Role string `json:"role"`
}

// SetOverrideRequest upserts a per-user permission override.
type SetOverrideRequest struct {
	Permission string `json:"permission"`
	Effect     string `json:"effect"`
	Reason     string `json:"reason"`
	ExpiresAt  string `json:"expires_at"`
}

// DeleteOverrideRequest removes a per-user permission override.
type DeleteOverrideRequest struct {
	Permission string `json:"permission"`
}
