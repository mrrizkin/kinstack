package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"kinstack/internal/platform/email"
	platformpassword "kinstack/internal/platform/password"
)

const (
	testAdminRole = "administrator"
	testStaffRole = "staff"
)

func seedUser(t *testing.T, repository *MemoryRepository, password string) {
	t.Helper()
	hash, err := platformpassword.Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repository.AddUser(FindUserByUsernameResult{
		PublicID:     "usr_1",
		Username:     "admin",
		DisplayName:  "Administrator",
		PasswordHash: hash,
		Role:         testAdminRole,
		IsActive:     true,
	})
}

func TestLoginAcceptsValidCredentials(t *testing.T) {
	repository := NewMemoryRepository()
	seedUser(t, repository, "rahasia123")
	service := NewService(ServiceDependency{Repository: repository})

	session, err := service.Login(context.Background(), "admin", "rahasia123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if session.PublicID != "usr_1" || session.Role != testAdminRole {
		t.Fatalf("unexpected session: %#v", session)
	}
	if len(repository.Touched()) != 1 {
		t.Fatalf("expected last login touch, got %#v", repository.Touched())
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	repository := NewMemoryRepository()
	seedUser(t, repository, "rahasia123")
	service := NewService(ServiceDependency{Repository: repository})

	_, err := service.Login(context.Background(), "admin", "salah")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestLoginRejectsUnknownUser(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	_, err := service.Login(context.Background(), "ghost", "apapun")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	repository := NewMemoryRepository()
	seedUser(t, repository, "rahasia123")
	repository.AddUser(FindUserByUsernameResult{
		PublicID:     "usr_2",
		Username:     "staf",
		PasswordHash: repository.users["admin"].PasswordHash,
		Role:         testStaffRole,
		IsActive:     false,
	})
	service := NewService(ServiceDependency{Repository: repository})

	_, err := service.Login(context.Background(), "staf", "rahasia123")
	if !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("expected inactive user, got %v", err)
	}
}

func TestLoginRejectsEmptyInput(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	if _, err := service.Login(context.Background(), "", "x"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestLoginAcceptsEnvironmentSuperAdmin(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(ServiceDependency{
		Repository: repository,
		AccessKey:  "superadmin",
		SecretKey:  "secret-super-admin-panjang",
	})

	session, err := service.Login(context.Background(), "superadmin", "secret-super-admin-panjang")
	if err != nil {
		t.Fatalf("super admin login: %v", err)
	}
	if !session.Superuser || session.PublicID == "" {
		t.Fatalf("expected ephemeral superuser session, got %#v", session)
	}
	if _, err := repository.FindUserByUsername(
		context.Background(),
		"superadmin",
	); !errors.Is(
		err,
		ErrNotFound,
	) {
		t.Fatalf("break-glass must not persist an account, got err=%v", err)
	}
}

func TestLoginRejectsWrongEnvironmentSecret(t *testing.T) {
	service := NewService(ServiceDependency{
		Repository: NewMemoryRepository(),
		AccessKey:  "superadmin",
		SecretKey:  "secret-super-admin-panjang",
	})
	_, err := service.Login(context.Background(), "superadmin", "salah")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestLoginWithoutEnvironmentCredentialsIgnoresSuperAdminPath(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	_, err := service.Login(context.Background(), "superadmin", "secret-super-admin-panjang")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestConsumeMagicLinkAuthenticatesAndBurnsToken(t *testing.T) {
	repository := NewMemoryRepository()
	repository.AddUser(FindUserByUsernameResult{
		PublicID:    "usr_magic",
		Username:    "budi",
		Email:       "budi@contoh.com",
		DisplayName: "Budi",
		Role:        testAdminRole,
		IsActive:    true,
	})
	service := NewService(ServiceDependency{Repository: repository})
	if err := repository.CreateMagicLinkToken(context.Background(), MagicLinkToken{
		UserPublicID: "usr_magic",
		TokenHash:    hashMagicToken("raw-token"),
		ExpiresAt:    time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("seed token: %v", err)
	}

	session, err := service.ConsumeMagicLink(context.Background(), "raw-token")
	if err != nil {
		t.Fatalf("consume magic link: %v", err)
	}
	if session.PublicID != "usr_magic" {
		t.Fatalf("unexpected session: %#v", session)
	}
	if _, err := service.ConsumeMagicLink(context.Background(), "raw-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected single-use token, got %v", err)
	}
}

func TestRequestMagicLinkSendsSingleUseEmail(t *testing.T) {
	repository := NewMemoryRepository()
	repository.AddUser(FindUserByUsernameResult{
		PublicID:    "usr_magic",
		Username:    "budi",
		Email:       "budi@contoh.com",
		DisplayName: "Budi",
		Role:        testAdminRole,
		IsActive:    true,
	})
	sender := email.NewMemorySender()
	service := NewService(ServiceDependency{
		Repository: repository,
		Email:      sender,
		EmailFrom:  "no-reply@kinstack.test",
		BaseURL:    "http://localhost:3000",
	})

	if err := service.RequestMagicLink(context.Background(), "budi@contoh.com"); err != nil {
		t.Fatalf("request magic link: %v", err)
	}
	messages := sender.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected one email, got %d", len(messages))
	}
	link := "http://localhost:3000/login/magic-link/"
	index := strings.Index(messages[0].TextBody, link)
	if index < 0 {
		t.Fatalf("magic link missing from email body: %q", messages[0].TextBody)
	}
	token := strings.TrimSpace(messages[0].TextBody[index+len(link):])
	if _, err := service.ConsumeMagicLink(context.Background(), token); err != nil {
		t.Fatalf("consume emailed token %q (body %q): %v", token, messages[0].TextBody, err)
	}
}

func TestRequestMagicLinkIgnoresUnknownEmail(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	if err := service.RequestMagicLink(context.Background(), "ghost@contoh.com"); err != nil {
		t.Fatalf("unknown email must stay silent, got %v", err)
	}
}

func TestSendInvitationIssuesUsableLink(t *testing.T) {
	repository := NewMemoryRepository()
	repository.AddUser(FindUserByUsernameResult{
		PublicID:    "usr_invited",
		Username:    "staf",
		Email:       "staf@contoh.com",
		DisplayName: "Staf Baru",
		Role:        testAdminRole,
		IsActive:    true,
	})
	sender := email.NewMemorySender()
	service := NewService(ServiceDependency{
		Repository: repository,
		Email:      sender,
		EmailFrom:  "no-reply@kinstack.test",
		BaseURL:    "http://localhost:3000",
	})

	if err := service.SendInvitation(context.Background(), "staf@contoh.com"); err != nil {
		t.Fatalf("send invitation: %v", err)
	}
	messages := sender.Messages()
	if len(messages) != 1 || messages[0].Subject == "" {
		t.Fatalf("expected one invitation email, got %#v", messages)
	}
	link := "http://localhost:3000/login/magic-link/"
	index := strings.Index(messages[0].TextBody, link)
	if index < 0 {
		t.Fatalf("invitation link missing from email body: %q", messages[0].TextBody)
	}
	token := strings.TrimSpace(messages[0].TextBody[index+len(link):])
	if _, err := service.ConsumeMagicLink(context.Background(), token); err != nil {
		t.Fatalf("consume invited token %q (body %q): %v", token, messages[0].TextBody, err)
	}
}

func TestSendInvitationFailsForUnknownEmail(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	if err := service.SendInvitation(context.Background(), "ghost@contoh.com"); !errors.Is(
		err,
		ErrNotFound,
	) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRequestPasswordResetEmailsUsableLink(t *testing.T) {
	repository := NewMemoryRepository()
	repository.AddUser(FindUserByUsernameResult{
		PublicID:     "usr_reset",
		Username:     "budi",
		Email:        "budi@contoh.com",
		DisplayName:  "Budi",
		PasswordHash: "stale",
		Role:         testAdminRole,
		IsActive:     true,
	})
	sender := email.NewMemorySender()
	service := NewService(ServiceDependency{
		Repository: repository,
		Email:      sender,
		EmailFrom:  "no-reply@kinstack.test",
		BaseURL:    "http://localhost:3000",
	})

	if err := service.RequestPasswordReset(context.Background(), "budi@contoh.com"); err != nil {
		t.Fatalf("request password reset: %v", err)
	}
	messages := sender.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected one email, got %d", len(messages))
	}
	link := "http://localhost:3000/login/reset/"
	index := strings.Index(messages[0].TextBody, link)
	if index < 0 {
		t.Fatalf("reset link missing from email body: %q", messages[0].TextBody)
	}
	token := strings.TrimSpace(messages[0].TextBody[index+len(link):])
	if err := service.ResetPassword(context.Background(), token, "passwordbaru123"); err != nil {
		t.Fatalf("reset password %q (body %q): %v", token, messages[0].TextBody, err)
	}
	// The token is single-use.
	if err := service.ResetPassword(context.Background(), token, "passwordlain123"); !errors.Is(
		err,
		ErrInvalidToken,
	) {
		t.Fatalf("expected burned token, got %v", err)
	}
	updated, err := repository.FindUserByPublicID(context.Background(), "usr_reset")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if err := platformpassword.Compare(updated.PasswordHash, "passwordbaru123"); err != nil {
		t.Fatalf("expected new password hash: %v", err)
	}
}

func TestRequestPasswordResetIgnoresUnknownEmail(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	if err := service.RequestPasswordReset(context.Background(), "ghost@contoh.com"); err != nil {
		t.Fatalf("unknown email must stay silent, got %v", err)
	}
}

func TestResetPasswordRejectsShortPassword(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	if err := service.ResetPassword(context.Background(), "token", "pendek"); !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
