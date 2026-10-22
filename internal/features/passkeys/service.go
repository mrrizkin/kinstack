package passkeys

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"

	"github.com/go-webauthn/webauthn/webauthn"
)

// User is the account data needed by the WebAuthn user boundary.
type User struct {
	PublicID    string
	Email       string
	Username    string
	DisplayName string
}

// PasskeyRecord is the credential data used by the WebAuthn business flow.
type PasskeyRecord struct {
	ID         []byte
	Name       string
	Credential webauthn.Credential
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type ServiceDependency struct {
	Repository Repository
}

type Service struct {
	repository Repository
}

func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "passkeys repository is required")

	return &Service{repository: dependency.Repository}
}

// FindByPublicID resolves an active account for a protected passkey flow.
func (s *Service) FindByPublicID(ctx context.Context, id string) (User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return User{}, ErrUserNotFound
	}
	result, err := s.repository.FindByPublicID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return userFromResult(result), nil
}

// ValidatePublicID validates an active account identifier.
func (s *Service) ValidatePublicID(ctx context.Context, id string) error {
	_, err := s.FindByPublicID(ctx, id)
	return err
}

// ListPasskeys returns the safe public representation for the management page.
func (s *Service) ListPasskeys(ctx context.Context, id string) ([]PasskeySummary, error) {
	records, err := s.passkeyRecords(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]PasskeySummary, 0, len(records))
	for _, record := range records {
		items = append(items, passkeySummary(record))
	}
	return items, nil
}

// PasskeyRecords returns credentials needed to construct a WebAuthn user.
func (s *Service) PasskeyRecords(ctx context.Context, id string) ([]PasskeyRecord, error) {
	return s.passkeyRecords(ctx, id)
}

func (s *Service) passkeyRecords(ctx context.Context, id string) ([]PasskeyRecord, error) {
	results, err := s.repository.ListPasskeys(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	records := make([]PasskeyRecord, 0, len(results))
	for _, result := range results {
		record, err := recordFromResult(result)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// SavePasskey validates and persists a newly registered credential.
func (s *Service) SavePasskey(ctx context.Context, id string, record PasskeyRecord) error {
	record.Name = strings.TrimSpace(record.Name)
	if len(record.ID) == 0 || !bytes.Equal(record.ID, record.Credential.ID) ||
		record.Name == "" || len(record.Name) > maxPasskeyNameLength {
		return ErrPasskeyInvalid
	}
	return s.repository.SavePasskey(ctx, strings.TrimSpace(id), record)
}

// UpdatePasskey persists authenticator state after a successful assertion.
func (s *Service) UpdatePasskey(ctx context.Context, id string, record PasskeyRecord) error {
	return s.repository.UpdatePasskey(ctx, strings.TrimSpace(id), record)
}

// DeletePasskey revokes one credential belonging to the active account.
func (s *Service) DeletePasskey(ctx context.Context, id string, credentialID []byte) error {
	if len(credentialID) == 0 {
		return ErrPasskeyInvalid
	}
	return s.repository.DeletePasskey(ctx, strings.TrimSpace(id), credentialID)
}

// FindPasskeyUser resolves the account and credentials selected by discoverable login.
func (s *Service) FindPasskeyUser(
	ctx context.Context,
	credentialID, userHandle []byte,
) (User, []PasskeyRecord, error) {
	userResult, results, err := s.repository.FindPasskeyUser(ctx, credentialID, userHandle)
	if err != nil {
		return User{}, nil, err
	}
	records := make([]PasskeyRecord, 0, len(results))
	for _, result := range results {
		record, err := recordFromResult(result)
		if err != nil {
			return User{}, nil, err
		}
		records = append(records, record)
	}
	return userFromResult(userResult), records, nil
}

func userFromResult(result FindPasskeyUserResult) User {
	return User{
		PublicID:    result.PublicID,
		Email:       result.Email,
		Username:    result.Username,
		DisplayName: result.DisplayName,
	}
}

func recordFromResult(result ListPasskeysResult) (PasskeyRecord, error) {
	var credential webauthn.Credential
	if err := json.Unmarshal(result.Credential, &credential); err != nil {
		return PasskeyRecord{}, fmt.Errorf("decode passkey credential: %w", err)
	}
	return PasskeyRecord{
		ID:         append([]byte(nil), result.ID...),
		Name:       result.Name,
		Credential: credential,
		CreatedAt:  result.CreatedAt,
		LastUsedAt: result.LastUsedAt,
	}, nil
}

func passkeySummary(record PasskeyRecord) PasskeySummary {
	summary := PasskeySummary{
		ID:        base64.RawURLEncoding.EncodeToString(record.ID),
		Name:      record.Name,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
	}
	if record.LastUsedAt != nil {
		summary.LastUsedAt = record.LastUsedAt.UTC().Format(time.RFC3339)
	}
	return summary
}

type passkeyUser struct {
	user        User
	credentials []webauthn.Credential
}

var _ webauthn.User = (*passkeyUser)(nil)

func passkeyUserHandle(id string) []byte {
	digest := sha256.Sum256([]byte("kinstack-webauthn:" + id))
	return digest[:]
}

func (u *passkeyUser) WebAuthnID() []byte { return passkeyUserHandle(u.user.PublicID) }

func (u *passkeyUser) WebAuthnName() string { return u.user.Username }

func (u *passkeyUser) WebAuthnDisplayName() string { return u.user.DisplayName }

func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return append([]webauthn.Credential(nil), u.credentials...)
}
