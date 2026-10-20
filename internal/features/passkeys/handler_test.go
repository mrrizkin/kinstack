package passkeys

import (
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func TestPasskeyCeremoniesRequireUserVerification(t *testing.T) {
	webAuthn, err := newWebAuthn("localhost", "http://localhost")
	if err != nil {
		t.Fatalf("build webauthn: %v", err)
	}

	login, _, err := passkeyLoginOptions(webAuthn)
	if err != nil {
		t.Fatalf("login options: %v", err)
	}
	if login.Response.UserVerification != protocol.VerificationRequired {
		t.Fatalf("login must require user verification, got %q", login.Response.UserVerification)
	}

	user := &passkeyUser{
		user: User{
			PublicID:    "usr_1",
			Username:    "admin",
			DisplayName: "Administrator",
		},
	}
	registration, _, err := passkeyRegistrationOptions(webAuthn, user, nil)
	if err != nil {
		t.Fatalf("registration options: %v", err)
	}
	if registration.Response.AuthenticatorSelection.UserVerification != protocol.VerificationRequired {
		t.Fatalf(
			"registration must require user verification, got %q",
			registration.Response.AuthenticatorSelection.UserVerification,
		)
	}
}

// Ensure the helpers keep the webauthn types they are written against.
var _ webauthn.User = (*passkeyUser)(nil)
