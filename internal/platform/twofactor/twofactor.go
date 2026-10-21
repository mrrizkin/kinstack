// Package twofactor owns per-user two-factor authentication: TOTP enrollment,
// login verification, and one-time recovery codes.
package twofactor

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/envelope"
	"kinstack/internal/platform/password"
	"kinstack/internal/platform/totp"
)

const (
	// RecoveryCodeCount is how many one-time codes are issued at a time.
	RecoveryCodeCount = 10
	recoveryCodeBytes = 8
	// recoveryKeyInfo scopes the envelope key to TOTP secrets.
	recoveryKeyInfo = "kinstack/two-factor/v1"
	// legacyHashLength is the hex length of codes hashed before Argon2id.
	legacyHashLength = sha256.Size * 2
)

const recoveryAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Dependency contains the persistence boundary and display metadata.
type Dependency struct {
	Repository Repository
	Issuer     string
	Now        func() time.Time
	// Box seals TOTP secrets at rest. When nil secrets stay in plaintext, which
	// is only acceptable for local wiring and tests.
	Box *envelope.Box
}

// Service owns two-factor enrollment and verification.
type Service struct {
	repository Repository
	issuer     string
	now        func() time.Time
	box        *envelope.Box
}

// New constructs the two-factor service.
func New(d Dependency) *Service {
	assert.Assert(d.Repository != nil, "two-factor repository is required")

	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		repository: d.Repository,
		issuer:     strings.TrimSpace(d.Issuer),
		now:        now,
		box:        d.Box,
	}
}

// Enrollment is a freshly generated secret and its provisioning URI.
type Enrollment struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// IsEnabled reports whether the account requires a second factor at login.
func (s *Service) IsEnabled(ctx context.Context, userPublicID string) (bool, error) {
	status, err := s.repository.FindStatus(ctx, userPublicID)
	if err != nil {
		return false, err
	}
	return status.Enabled, nil
}

// Status returns the enrollment state with the secret decrypted for the
// enrollment screen. The secret is never rendered once two-factor is enabled.
func (s *Service) Status(ctx context.Context, userPublicID string) (StatusResult, error) {
	status, err := s.repository.FindStatus(ctx, userPublicID)
	if err != nil {
		return StatusResult{}, err
	}
	secret, err := s.open(status)
	if err != nil {
		return StatusResult{}, err
	}
	return StatusResult{Enabled: status.Enabled, Secret: secret, LastStep: status.LastStep}, nil
}

// ProvisioningURI rebuilds the otpauth URI for a stored secret.
func (s *Service) ProvisioningURI(secret, account string) string {
	return totp.ProvisioningURI(secret, account, s.issuer)
}

// BeginEnrollment stores a new pending secret. It is not active until the user
// confirms one valid code.
func (s *Service) BeginEnrollment(
	ctx context.Context,
	userPublicID string,
	account string,
) (Enrollment, error) {
	status, err := s.repository.FindStatus(ctx, userPublicID)
	if err != nil {
		return Enrollment{}, err
	}
	if status.Enabled {
		return Enrollment{}, ErrAlreadyEnabled
	}
	secret, err := totp.GenerateSecret()
	if err != nil {
		return Enrollment{}, err
	}
	stored, err := s.seal(secret)
	if err != nil {
		return Enrollment{}, err
	}
	if err := s.repository.SetSecret(ctx, userPublicID, stored); err != nil {
		return Enrollment{}, err
	}
	return Enrollment{
		Secret: secret,
		URI:    totp.ProvisioningURI(secret, account, s.issuer),
	}, nil
}

// ConfirmEnrollment activates two-factor auth and returns the one-time recovery
// codes in plaintext. They are never retrievable again.
func (s *Service) ConfirmEnrollment(
	ctx context.Context,
	userPublicID string,
	code string,
) ([]string, error) {
	status, err := s.repository.FindStatus(ctx, userPublicID)
	if err != nil {
		return nil, err
	}
	if status.Enabled {
		return nil, ErrAlreadyEnabled
	}
	secret, err := s.open(status)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(secret) == "" {
		return nil, ErrNotConfigured
	}
	step, ok := totp.ValidateStep(secret, code, s.now())
	if !ok {
		return nil, ErrInvalidCode
	}
	if err := s.repository.Enable(ctx, userPublicID, step); err != nil {
		return nil, err
	}
	s.sealLegacy(ctx, userPublicID, status)
	return s.issueRecoveryCodes(ctx, userPublicID)
}

// Disable turns off two-factor auth and removes the secret and recovery codes.
func (s *Service) Disable(ctx context.Context, userPublicID string) error {
	if err := s.repository.Disable(ctx, userPublicID); err != nil {
		return err
	}
	return s.repository.ReplaceRecoveryCodes(ctx, userPublicID, nil)
}

// RegenerateRecoveryCodes replaces every stored code with a fresh set.
func (s *Service) RegenerateRecoveryCodes(
	ctx context.Context,
	userPublicID string,
) ([]string, error) {
	status, err := s.repository.FindStatus(ctx, userPublicID)
	if err != nil {
		return nil, err
	}
	if !status.Enabled {
		return nil, ErrNotEnabled
	}
	return s.issueRecoveryCodes(ctx, userPublicID)
}

// Verify accepts either a current TOTP code or an unused recovery code. A used
// code is consumed so it cannot be replayed, and a TOTP step is recorded so a
// captured code cannot be reused inside its validity window.
func (s *Service) Verify(ctx context.Context, userPublicID, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	status, err := s.repository.FindStatus(ctx, userPublicID)
	if err != nil {
		return false, err
	}
	if !status.Enabled {
		return false, ErrNotEnabled
	}
	secret, err := s.open(status)
	if err != nil {
		return false, err
	}
	if step, ok := totp.ValidateStep(secret, code, s.now()); ok {
		switch err := s.repository.ConsumeStep(ctx, userPublicID, step); {
		case errors.Is(err, ErrNotFound):
			// The step was already used: treat a replayed code as invalid.
			return false, nil
		case err != nil:
			return false, err
		}
		s.sealLegacy(ctx, userPublicID, status)
		return true, nil
	}
	return s.verifyRecoveryCode(ctx, userPublicID, code)
}

// verifyRecoveryCode checks a one-time code against the unused hashes. Argon2id
// hashes carry their own salt, so every stored hash is compared in turn; a
// handler-level lockout bounds the cost of repeated guesses.
func (s *Service) verifyRecoveryCode(ctx context.Context, userPublicID, code string) (bool, error) {
	if !recoveryCodeShape.MatchString(code) {
		return false, nil
	}
	normalized := normalizeRecoveryCode(code)
	codes, err := s.repository.UnusedRecoveryCodes(ctx, userPublicID)
	if err != nil {
		return false, err
	}
	legacyDigest := legacyRecoveryDigest(normalized)
	for _, stored := range codes {
		matches := false
		legacy := isLegacyRecoveryHash(stored.Hash)
		if legacy {
			matches = subtle.ConstantTimeCompare(
				[]byte(legacyDigest),
				[]byte(strings.ToLower(stored.Hash)),
			) == 1
		} else if password.Compare(stored.Hash, normalized) == nil {
			matches = true
		}
		if !matches {
			continue
		}
		if legacy {
			if upgraded, err := hashRecoveryCode(normalized); err == nil {
				_ = s.repository.ReplaceRecoveryCodeHash(ctx, stored.ID, upgraded)
			}
		}
		switch err := s.repository.MarkRecoveryCodeUsed(ctx, stored.ID); {
		case errors.Is(err, ErrNotFound):
			// Another request consumed it first; keep looking.
			continue
		case err != nil:
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) issueRecoveryCodes(ctx context.Context, userPublicID string) ([]string, error) {
	codes, err := GenerateRecoveryCodes(RecoveryCodeCount)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := hashRecoveryCode(normalizeRecoveryCode(code))
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}
	if err := s.repository.ReplaceRecoveryCodes(ctx, userPublicID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

// seal converts a plaintext secret into the persisted representation. A sealing
// failure is returned instead of falling back to plaintext: storing the secret
// unsealed would silently weaken the at-rest guarantee.
func (s *Service) seal(secret string) (Secret, error) {
	if s.box == nil || strings.TrimSpace(secret) == "" {
		return Secret{Plain: secret}, nil
	}
	sealed, err := s.box.Seal([]byte(secret))
	if err != nil {
		return Secret{}, err
	}
	return Secret{Sealed: sealed, KeyVersion: s.box.Version()}, nil
}

// open returns the plaintext secret for a stored row.
func (s *Service) open(status StatusResult) (string, error) {
	if len(status.Sealed) > 0 {
		if s.box == nil {
			return "", ErrNotConfigured
		}
		plaintext, err := s.box.Open(status.Sealed)
		if err != nil {
			return "", fmt.Errorf("%w: open two-factor secret", ErrNotConfigured)
		}
		return string(plaintext), nil
	}
	return status.Secret, nil
}

// sealLegacy writes a sealed copy for a row that still stores plaintext and
// clears the plaintext column. Failures are not fatal for the caller: the row
// stays usable and the backfill command can retry later.
func (s *Service) sealLegacy(ctx context.Context, userPublicID string, status StatusResult) {
	if s.box == nil || strings.TrimSpace(status.Secret) == "" || len(status.Sealed) > 0 {
		return
	}
	sealed, err := s.box.Seal([]byte(status.Secret))
	if err != nil {
		return
	}
	_ = s.repository.SealSecret(ctx, userPublicID, sealed, s.box.Version())
}

// GenerateRecoveryCodes returns n human-friendly one-time codes shaped
// "XXXX-XXXX" using an alphabet without ambiguous characters.
func GenerateRecoveryCodes(n int) ([]string, error) {
	if n <= 0 {
		return []string{}, nil
	}
	codes := make([]string, 0, n)
	for range n {
		buffer := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(buffer); err != nil {
			return nil, err
		}
		var builder strings.Builder
		for index, value := range buffer {
			if index == recoveryCodeBytes/2 {
				builder.WriteByte('-')
			}
			builder.WriteByte(recoveryAlphabet[int(value)%len(recoveryAlphabet)])
		}
		codes = append(codes, builder.String())
	}
	return codes, nil
}

// HashRecoveryCode normalizes a code and returns its Argon2id hash for storage.
func HashRecoveryCode(code string) (string, error) {
	return hashRecoveryCode(normalizeRecoveryCode(code))
}

func hashRecoveryCode(normalized string) (string, error) {
	return password.HashWithCost(normalized, password.OneTimeCost)
}

// BackfillSecrets seals plaintext secrets in batches and reports how many rows
// were migrated. It is idempotent and safe to run while the service is live.
func BackfillSecrets(
	ctx context.Context,
	repository Repository,
	box *envelope.Box,
	batch int,
) (int, error) {
	if box == nil {
		return 0, fmt.Errorf("%w: envelope box is required", ErrNotConfigured)
	}
	if batch <= 0 {
		batch = 100
	}
	migrated := 0
	for {
		legacy, err := repository.LegacySecrets(ctx, batch)
		if err != nil {
			return migrated, err
		}
		if len(legacy) == 0 {
			return migrated, nil
		}
		for _, row := range legacy {
			sealed, err := box.Seal([]byte(row.Secret))
			if err != nil {
				return migrated, err
			}
			if err := repository.SealSecret(ctx, row.PublicID, sealed, box.Version()); err != nil {
				return migrated, err
			}
			migrated++
		}
		if len(legacy) < batch {
			return migrated, nil
		}
	}
}

// RecoveryKeyInfo scopes the envelope key derivation for TOTP secrets.
const RecoveryKeyInfo = recoveryKeyInfo
