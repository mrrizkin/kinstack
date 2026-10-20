package twofactor

import (
	"context"
	"errors"
	"testing"
	"time"

	"kinstack/internal/platform/totp"
)

var testNow = time.Unix(1_700_000_000, 0).UTC()

func newService(t *testing.T) (*Service, *MemoryRepository, string) {
	t.Helper()
	repository := NewMemoryRepository("user-1")
	service := New(Dependency{
		Repository: repository,
		Issuer:     "Aplikasi Uji",
		Now:        func() time.Time { return testNow },
	})
	return service, repository, "user-1"
}

func TestEnrollmentConfirmAndVerify(t *testing.T) {
	service, _, publicID := newService(t)
	ctx := context.Background()

	enrollment, err := service.BeginEnrollment(ctx, publicID, "pemilik")
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	if enrollment.Secret == "" || enrollment.URI == "" {
		t.Fatalf("unexpected enrollment: %#v", enrollment)
	}
	enabled, err := service.IsEnabled(ctx, publicID)
	if err != nil {
		t.Fatalf("is enabled: %v", err)
	}
	if enabled {
		t.Fatal("expected two-factor to stay disabled before confirmation")
	}

	if _, err := service.ConfirmEnrollment(ctx, publicID, "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}

	code, err := totp.GenerateCode(enrollment.Secret, testNow)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	codes, err := service.ConfirmEnrollment(ctx, publicID, code)
	if err != nil {
		t.Fatalf("confirm enrollment: %v", err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("expected %d recovery codes, got %d", RecoveryCodeCount, len(codes))
	}

	enabled, err = service.IsEnabled(ctx, publicID)
	if err != nil {
		t.Fatalf("is enabled: %v", err)
	}
	if !enabled {
		t.Fatal("expected two-factor to be enabled after confirmation")
	}

	ok, err := service.Verify(ctx, publicID, code)
	if err != nil || ok {
		t.Fatalf("expected a replayed TOTP code to be rejected, ok=%v err=%v", ok, err)
	}

	// The next time step issues a fresh code that still verifies.
	service.now = func() time.Time { return testNow.Add(totp.Period * time.Second) }
	next, err := totp.GenerateCode(enrollment.Secret, service.now())
	if err != nil {
		t.Fatalf("generate next code: %v", err)
	}
	ok, err = service.Verify(ctx, publicID, next)
	if err != nil || !ok {
		t.Fatalf("expected the next TOTP code to verify, ok=%v err=%v", ok, err)
	}

	ok, err = service.Verify(ctx, publicID, "123456")
	if err != nil {
		t.Fatalf("verify wrong code: %v", err)
	}
	if ok {
		t.Fatal("expected wrong code to be rejected")
	}
}

func TestRecoveryCodeIsOneTime(t *testing.T) {
	service, _, publicID := newService(t)
	ctx := context.Background()

	enrollment, err := service.BeginEnrollment(ctx, publicID, "pemilik")
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	code, err := totp.GenerateCode(enrollment.Secret, testNow)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	codes, err := service.ConfirmEnrollment(ctx, publicID, code)
	if err != nil {
		t.Fatalf("confirm enrollment: %v", err)
	}

	ok, err := service.Verify(ctx, publicID, codes[0])
	if err != nil || !ok {
		t.Fatalf("expected recovery code to verify, ok=%v err=%v", ok, err)
	}
	ok, err = service.Verify(ctx, publicID, codes[0])
	if err != nil {
		t.Fatalf("re-verify recovery code: %v", err)
	}
	if ok {
		t.Fatal("expected a recovery code to be rejected on replay")
	}
}

func TestDisableAndRegenerate(t *testing.T) {
	service, _, publicID := newService(t)
	ctx := context.Background()

	enrollment, err := service.BeginEnrollment(ctx, publicID, "pemilik")
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	code, _ := totp.GenerateCode(enrollment.Secret, testNow)
	if _, err := service.ConfirmEnrollment(ctx, publicID, code); err != nil {
		t.Fatalf("confirm enrollment: %v", err)
	}

	rotated, err := service.RegenerateRecoveryCodes(ctx, publicID)
	if err != nil || len(rotated) != RecoveryCodeCount {
		t.Fatalf("regenerate recovery codes: %v (%d)", err, len(rotated))
	}

	if err := service.Disable(ctx, publicID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	enabled, err := service.IsEnabled(ctx, publicID)
	if err != nil {
		t.Fatalf("is enabled: %v", err)
	}
	if enabled {
		t.Fatal("expected two-factor to be disabled")
	}
	if _, err := service.Verify(ctx, publicID, code); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("expected ErrNotEnabled, got %v", err)
	}
	if _, err := service.RegenerateRecoveryCodes(ctx, publicID); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("expected ErrNotEnabled for regenerate, got %v", err)
	}
}

func TestConfirmRequiresEnrollment(t *testing.T) {
	service, _, publicID := newService(t)

	if _, err := service.ConfirmEnrollment(context.Background(), publicID, "123456"); !errors.Is(
		err,
		ErrNotConfigured,
	) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}
