package profile

// ProfileDTO is the public contract for the signed-in user profile.
type ProfileDTO struct {
	PublicID    string `json:"public_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Invited     bool   `json:"invited"`
	LastLoginAt string `json:"last_login_at"`
	CreatedAt   string `json:"created_at"`
}

// PreferencesDTO is the public contract for per-user personalization.
type PreferencesDTO struct {
	Theme        string `json:"theme"`
	SoundEnabled bool   `json:"sound_enabled"`
}

// SessionDTO describes the current browser session.
type SessionDTO struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	LoginAt   string `json:"login_at"`
}

// ActiveSessionDTO describes one signed-in browser session of the account.
type ActiveSessionDTO struct {
	PublicID   string `json:"public_id"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	LoginAt    string `json:"login_at"`
	LastSeenAt string `json:"last_seen_at"`
	Current    bool   `json:"current"`
}

// TwoFactorDTO is the public contract for the two-factor security panel.
type TwoFactorDTO struct {
	Enabled       bool     `json:"enabled"`
	PendingSecret string   `json:"pending_secret"`
	URI           string   `json:"uri"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// ProfilePageProps is the public contract for the profile page.
type ProfilePageProps struct {
	Profile     ProfileDTO                `json:"profile"`
	Preferences PreferencesDTO            `json:"preferences"`
	Session     SessionDTO                `json:"session"`
}

// UpdateProfileRequest is the payload accepted when renaming the profile.
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

// UpdatePreferencesRequest is the payload accepted when saving preferences.
type UpdatePreferencesRequest struct {
	Theme        string `json:"theme"`
	SoundEnabled bool   `json:"sound_enabled"`
}

// ChangePasswordRequest is the payload accepted when changing the password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// ConfirmTwoFactorRequest is the TOTP code that activates enrollment.
type ConfirmTwoFactorRequest struct {
	Code string `json:"code"`
}

// PasswordProtectedRequest carries the password required for two-factor changes.
type PasswordProtectedRequest struct {
	Password string `json:"password"`
}
