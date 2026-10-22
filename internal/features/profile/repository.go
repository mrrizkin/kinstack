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
	PageSize        int        `db:"page_size"`
	LastLoginAt     *time.Time `db:"last_login_at"`
	CreatedAt       time.Time  `db:"created_at"`
}

// PreferencesResult is the persistence projection of personalization settings.
type PreferencesResult struct {
	ThemePreference string `db:"theme_preference"`
	SoundEnabled    bool   `db:"sound_enabled"`
	PageSize        int    `db:"page_size"`
}

// PreferencesUpdate is the full personalization state written in one statement,
// so a form submit cannot drop a preference it does not render.
type PreferencesUpdate struct {
	ThemePreference string
	SoundEnabled    bool
	PageSize        int
}

// Repository is the persistence contract owned by the profile slice.
type Repository interface {
	FindProfile(context.Context, string) (ProfileResult, error)
	FindPasswordHash(context.Context, string) (string, error)
	FindPageSize(context.Context, string) (int, error)
	UpdateDisplayName(context.Context, string, string) (ProfileResult, error)
	UpdatePreferences(context.Context, string, PreferencesUpdate) (PreferencesResult, error)
	MarkPasswordSet(context.Context, string, string) error
}
