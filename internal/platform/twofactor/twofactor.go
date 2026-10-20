// Package twofactor owns per-user two-factor authentication: TOTP enrollment,
// login verification, and one-time recovery codes.
package twofactor

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/totp"
)

const (
	// RecoveryCodeCount is how many one-time codes are issued at a time.
	RecoveryCodeCount = 10
	recoveryCodeBytes = 8
)

const recoveryAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Dependency contains the persistence boundary and display metadata.
type Dependency struct {
	Repository Repository
	Issuer     string
	Now        func() time.Time
}

// Service owns two-factor enrollment and verification.
type Service struct {
	repository Repository
	issuer     string
	now        func() time.Time
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

// Status returns the raw enrollment state of the account.
func (s *Service) Status(ctx context.Context, userPublicID string) (StatusResult, error) {
	return s.repository.FindStatus(ctx, userPublicID)
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
	if err := s.repository.SetSecret(ctx, userPublicID, secret); err != nil {
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
	if strings.TrimSpace(status.Secret) == "" {
		return nil, ErrNotConfigured
	}
	if !totp.Validate(status.Secret, code, s.now()) {
		return nil, ErrInvalidCode
	}
	if err := s.repository.Enable(ctx, userPublicID); err != nil {
		return nil, err
	}
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
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, userPublicID string) ([]string, error) {
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
// recovery code is consumed so it cannot be replayed.
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
	if totp.Validate(status.Secret, code, s.now()) {
		return true, nil
	}
	recoveryID, err := s.repository.FindRecoveryCode(ctx, userPublicID, HashRecoveryCode(code))
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := s.repository.MarkRecoveryCodeUsed(ctx, recoveryID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) issueRecoveryCodes(ctx context.Context, userPublicID string) ([]string, error) {
	codes, err := GenerateRecoveryCodes(RecoveryCodeCount)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hashes = append(hashes, HashRecoveryCode(code))
	}
	if err := s.repository.ReplaceRecoveryCodes(ctx, userPublicID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
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

// HashRecoveryCode normalizes and hashes a code for storage and lookup.
func HashRecoveryCode(code string) string {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	normalized = strings.NewReplacer("-", "", " ", "").Replace(normalized)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
