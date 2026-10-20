package profile

import (
	"context"
	"time"
)

// ProfileResult is the persistence projection of one user account.
type ProfileResult struct {
	PublicID        string     `db:"public_id"`
	Username        string     `db:"username"`
	DisplayName     string     `db:"display_name"`
	Role            string     `db:"role"`
	Invited         bool       `db:"invited"`
	ThemePreference string     `db:"theme_preference"`
	SoundEnabled    bool       `db:"sound_enabled"`
	LastLoginAt     *time.Time `db:"last_login_at"`
	CreatedAt       time.Time  `db:"created_at"`
}

// PreferencesResult is the persistence projection of personalization settings.
type PreferencesResult struct {
	ThemePreference string `db:"theme_preference"`
	SoundEnabled    bool   `db:"sound_enabled"`
}

// Repository is the persistence contract owned by the profile slice.
type Repository interface {
	FindProfile(context.Context, string) (ProfileResult, error)
	FindPasswordHash(context.Context, string) (string, error)
	UpdateDisplayName(context.Context, string, string) (ProfileResult, error)
	UpdatePreferences(context.Context, string, string, bool) (PreferencesResult, error)
	MarkPasswordSet(context.Context, string, string) error
}
