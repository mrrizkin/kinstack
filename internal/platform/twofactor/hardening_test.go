package twofactor

import (
	"context"
	"strings"
	"testing"
	"time"

	"kinstack/internal/platform/envelope"
	"kinstack/internal/platform/totp"
)

func testBox(t *testing.T) *envelope.Box {
	t.Helper()
	box, err := envelope.New(strings.Repeat("k", 32), RecoveryKeyInfo, 1)
	if err != nil {
		t.Fatalf("build envelope box: %v", err)
	}
	return box
}

func enabledSecret(t *testing.T, service *Service, publicID string) string {
	t.Helper()
	ctx := context.Background()
	enrollment, err := service.BeginEnrollment(ctx, publicID, "pemilik")
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	code, err := totp.GenerateCode(enrollment.Secret, testNow)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if _, err := service.ConfirmEnrollment(ctx, publicID, code); err != nil {
		t.Fatalf("confirm enrollment: %v", err)
	}
	return enrollment.Secret
}

func TestSecretIsSealedAtRest(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository("user-1")
	service := New(Dependency{
		Repository: repository,
		Issuer:     "Aplikasi Uji",
		Now:        func() time.Time { return testNow },
		Box:        testBox(t),
	})

	enabledSecret(t, service, "user-1")

	status, err := repository.FindStatus(ctx, "user-1")
	if err != nil {
		t.Fatalf("find status: %v", err)
	}
	if status.Secret != "" {
		t.Fatal("expected the plaintext secret to be cleared")
	}
	if len(status.Sealed) == 0 {
		t.Fatal("expected a sealed secret")
	}
	if status.KeyVersion != 1 {
		t.Fatalf("expected key version 1, got %d", status.KeyVersion)
	}

	service.now = func() time.Time { return testNow.Add(totp.Period * time.Second) }
	code, _ := totp.GenerateCode(
		mustOpen(t, testBox(t), status.Sealed),
		service.now(),
	)
	ok, err := service.Verify(ctx, "user-1", code)
	if err != nil || !ok {
		t.Fatalf("expected sealed secret to verify, ok=%v err=%v", ok, err)
	}
}

func mustOpen(t *testing.T, box *envelope.Box, sealed []byte) string {
	t.Helper()
	plaintext, err := box.Open(sealed)
	if err != nil {
		t.Fatalf("open sealed secret: %v", err)
	}
	return string(plaintext)
}

func TestSealedSecretRejectsWrongKey(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository("user-1")
	service := New(Dependency{
		Repository: repository,
		Now:        func() time.Time { return testNow },
		Box:        testBox(t),
	})
	enabledSecret(t, service, "user-1")

	otherKey, err := envelope.New(strings.Repeat("o", 32), RecoveryKeyInfo, 1)
	if err != nil {
		t.Fatalf("build other box: %v", err)
	}
	swap := New(Dependency{
		Repository: repository,
		Now:        func() time.Time { return testNow },
		Box:        otherKey,
	})
	if _, err := swap.Verify(ctx, "user-1", "123456"); err == nil {
		t.Fatal("expected a decryption failure with the wrong key")
	}
}

type recordingRepository struct {
	*MemoryRepository
	upgraded []string
}

func (r *recordingRepository) ReplaceRecoveryCodeHash(
	ctx context.Context,
	id int64,
	hash string,
) error {
	r.upgraded = append(r.upgraded, hash)
	return r.MemoryRepository.ReplaceRecoveryCodeHash(ctx, id, hash)
}

func TestLegacyRecoveryCodeIsAcceptedAndUpgraded(t *testing.T) {
	ctx := context.Background()
	memory := NewMemoryRepository("user-1")
	repository := &recordingRepository{MemoryRepository: memory}
	service := New(Dependency{
		Repository: repository,
		Now:        func() time.Time { return testNow },
		Box:        testBox(t),
	})

	repository.ReplaceRecoveryCodes(ctx, "user-1", nil)
	enabledSecret(t, service, "user-1")

	legacyCode := "ABCD-EFGH"
	if err := memory.ReplaceRecoveryCodes(
		ctx,
		"user-1",
		[]string{legacyRecoveryDigest(normalizeRecoveryCode(legacyCode))},
	); err != nil {
		t.Fatalf("seed legacy recovery code: %v", err)
	}

	ok, err := service.Verify(ctx, "user-1", legacyCode)
	if err != nil || !ok {
		t.Fatalf("expected legacy recovery code to verify, ok=%v err=%v", ok, err)
	}
	if len(repository.upgraded) != 1 {
		t.Fatalf("expected one hash upgrade, got %d", len(repository.upgraded))
	}
	if !strings.HasPrefix(repository.upgraded[0], "$argon2id$") {
		t.Fatalf("expected an Argon2id hash, got %q", repository.upgraded[0])
	}

	// The code stays one-time after the upgrade.
	ok, err = service.Verify(ctx, "user-1", legacyCode)
	if err != nil {
		t.Fatalf("re-verify legacy code: %v", err)
	}
	if ok {
		t.Fatal("expected the legacy code to be consumed")
	}
}

func TestRecoveryCodesUseArgon2id(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository("user-1")
	service := New(Dependency{
		Repository: repository,
		Now:        func() time.Time { return testNow },
		Box:        testBox(t),
	})
	enabledSecret(t, service, "user-1")

	codes, err := repository.UnusedRecoveryCodes(ctx, "user-1")
	if err != nil {
		t.Fatalf("list recovery codes: %v", err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("expected %d codes, got %d", RecoveryCodeCount, len(codes))
	}
	for _, code := range codes {
		if !strings.HasPrefix(code.Hash, "$argon2id$") {
			t.Fatalf("expected Argon2id hash, got %q", code.Hash)
		}
	}
	hashA, err := hashRecoveryCode("ABCDEFGH")
	if err != nil {
		t.Fatalf("hash code: %v", err)
	}
	hashB, err := hashRecoveryCode("ABCDEFGH")
	if err != nil {
		t.Fatalf("hash code: %v", err)
	}
	if hashA == hashB {
		t.Fatal("expected per-hash salts")
	}
}

func TestBackfillSecretsSealsLegacyRows(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository("user-1", "user-2")
	plainService := New(Dependency{
		Repository: repository,
		Now:        func() time.Time { return testNow },
	})
	for _, publicID := range []string{"user-1", "user-2"} {
		enrollment, err := plainService.BeginEnrollment(ctx, publicID, publicID)
		if err != nil {
			t.Fatalf("begin enrollment: %v", err)
		}
		if enrollment.Secret == "" {
			t.Fatal("expected a plaintext secret")
		}
	}

	migrated, err := BackfillSecrets(ctx, repository, testBox(t), 1)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if migrated != 2 {
		t.Fatalf("expected 2 migrated rows, got %d", migrated)
	}
	legacy, err := repository.LegacySecrets(ctx, 10)
	if err != nil {
		t.Fatalf("list legacy secrets: %v", err)
	}
	if len(legacy) != 0 {
		t.Fatalf("expected no plaintext secrets left, got %d", len(legacy))
	}
	again, err := BackfillSecrets(ctx, repository, testBox(t), 10)
	if err != nil || again != 0 {
		t.Fatalf("expected an idempotent backfill, migrated=%d err=%v", again, err)
	}
}
