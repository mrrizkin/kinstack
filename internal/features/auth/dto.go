package auth

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SessionResponse is the authenticated actor returned to the client.
type SessionResponse struct {
	PublicID    string `json:"public_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Superuser   bool   `json:"superuser,omitempty"`
}

type LoginPageProps struct {
	AppName string            `json:"app_name"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// TwoFactorRequest is the second-factor code submitted after a valid password.
type TwoFactorRequest struct {
	Code string `json:"code"`
}

type MagicLinkRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}
