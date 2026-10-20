package auth

// LoginRequest is the credential payload accepted by the sign-in endpoint.
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

// LoginPageProps is the public contract for the sign-in page.
type LoginPageProps struct {
	AppName string            `json:"app_name"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// TwoFactorRequest is the second-factor code submitted after a valid password.
type TwoFactorRequest struct {
	Code string `json:"code"`
}

// MagicLinkRequest is the email payload for a passwordless sign-in request.
type MagicLinkRequest struct {
	Email string `json:"email"`
}

// ForgotPasswordRequest is the email payload for a self-service reset request.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the new password submitted with a reset token.
type ResetPasswordRequest struct {
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}
