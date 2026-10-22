package profile

type ProfileDTO struct {
	PublicID    string `json:"public_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Invited     bool   `json:"invited"`
	LastLoginAt string `json:"last_login_at"`
	CreatedAt   string `json:"created_at"`
}

type PreferencesDTO struct {
	Theme        string `json:"theme"`
	SoundEnabled bool   `json:"sound_enabled"`
	PageSize     int    `json:"page_size"`
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

type TwoFactorDTO struct {
	Enabled       bool     `json:"enabled"`
	PendingSecret string   `json:"pending_secret"`
	URI           string   `json:"uri"`
	RecoveryCodes []string `json:"recovery_codes"`
}

type ProfilePageProps struct {
	Profile     ProfileDTO     `json:"profile"`
	Preferences PreferencesDTO `json:"preferences"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

type UpdatePreferencesRequest struct {
	Theme        string `json:"theme"`
	SoundEnabled bool   `json:"sound_enabled"`
	PageSize     int    `json:"page_size"`
}

// UpdateThemeRequest is the payload accepted from the quick theme menu, which
// changes the theme without touching the notification-sound preference.
type UpdateThemeRequest struct {
	Theme string `json:"theme"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type ConfirmTwoFactorRequest struct {
	Code string `json:"code"`
}

// PasswordProtectedRequest carries the password required for two-factor changes.
type PasswordProtectedRequest struct {
	Password string `json:"password"`
}
