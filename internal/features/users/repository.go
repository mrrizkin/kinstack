package users

import (
	"context"
	"time"

	"kinstack/internal/pkg/pagination"
)

// UserResult is the account projection shared by user queries.
type UserResult struct {
	PublicID    string     `db:"public_id"`
	Username    string     `db:"username"`
	Email       string     `db:"email"`
	DisplayName string     `db:"display_name"`
	Role        string     `db:"role"`
	IsActive    bool       `db:"is_active"`
	Invited     bool       `db:"invited"`
	LastLoginAt *time.Time `db:"last_login_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

// CreateUserInput is the persistence input for a new account.
type CreateUserInput struct {
	PublicID     string
	Username     string
	Email        string
	DisplayName  string
	PasswordHash string
	Invited      bool
}

// UpdateUserInput is the persistence input for an edited account.
type UpdateUserInput struct {
	PublicID    string
	Email       string
	DisplayName string
}

// UserCounts is the aggregate projection used by the dashboard and pagination.
type UserCounts struct {
	Total  int
	Active int
}

// Repository is the persistence contract owned by the users slice.
type Repository interface {
	ListUsers(context.Context, pagination.Query) ([]UserResult, error)
	ListAllUsers(context.Context) ([]UserResult, error)
	CountUsers(context.Context) (UserCounts, error)
	FindUserByPublicID(context.Context, string) (UserResult, error)
	CreateUser(context.Context, CreateUserInput) (UserResult, error)
	UpdateUser(context.Context, UpdateUserInput) (UserResult, error)
	UpdatePassword(context.Context, string, string) error
	SetActive(context.Context, string, bool) (UserResult, error)
}
