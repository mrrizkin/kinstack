package apitoken

import "kinstack/internal/pkg/pagination"

// The plaintext token is returned once at creation and never stored here.
type APITokenDTO struct {
	PublicID   string   `json:"public_id"`
	Name       string   `json:"name"`
	Prefix     string   `json:"prefix"`
	Scopes     []string `json:"scopes"`
	ExpiresAt  string   `json:"expires_at,omitempty"`
	LastUsedAt string   `json:"last_used_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

type TokenPage struct {
	Tokens []APITokenDTO   `json:"tokens"`
	Page   pagination.Page `json:"page"`
}

type CreateAPITokenRequest struct {
	Name          string   `json:"name"`
	ExpiresInDays int      `json:"expires_in_days"`
	Scopes        []string `json:"scopes"`
}
