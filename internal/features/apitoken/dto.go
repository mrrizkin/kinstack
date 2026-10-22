package apitoken

import "kinstack/internal/pkg/pagination"

// APITokenDTO is the public contract for one personal API token. The plaintext
// token is never part of it: it is returned once at creation.
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

// CreateAPITokenRequest is the payload accepted when creating a token.
type CreateAPITokenRequest struct {
	Name          string   `json:"name"`
	ExpiresInDays int      `json:"expires_in_days"`
	Scopes        []string `json:"scopes"`
}
