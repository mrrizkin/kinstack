package twofactor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	twofactorsqlc "kinstack/internal/platform/twofactor/sqlc"
)

type SQLRepository struct {
	queries *twofactorsqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "two-factor database is required")

	queries, err := twofactorsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare two-factor queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) FindStatus(ctx context.Context, userPublicID string) (StatusResult, error) {
	row, err := r.queries.FindTwoFactorStatus(ctx, userPublicID)
	if errors.Is(err, sql.ErrNoRows) {
		return StatusResult{}, ErrNotFound
	}
	if err != nil {
		return StatusResult{}, fmt.Errorf("find two-factor status: %w", err)
	}
	return StatusResult{
		Enabled:    row.TwoFactorEnabled,
		Secret:     row.TwoFactorSecret,
		Sealed:     row.TwoFactorSecretEnc,
		KeyVersion: int(row.TwoFactorKeyVersion),
		LastStep:   row.TwoFactorLastStep,
	}, nil
}

func (r *SQLRepository) SetSecret(ctx context.Context, userPublicID string, secret Secret) error {
	sealed := secret.Sealed
	if sealed == nil {
		sealed = []byte{}
	}
	affected, err := r.queries.SetTwoFactorSecret(ctx, twofactorsqlc.SetTwoFactorSecretParams{
		PublicID:            userPublicID,
		TwoFactorSecret:     secret.Plain,
		TwoFactorSecretEnc:  sealed,
		TwoFactorKeyVersion: int32(secret.KeyVersion),
	})
	return affectingOne(affected, err, "set two-factor secret")
}

func (r *SQLRepository) SealSecret(
	ctx context.Context,
	userPublicID string,
	sealed []byte,
	version int,
) error {
	if sealed == nil {
		sealed = []byte{}
	}
	affected, err := r.queries.SealTwoFactorSecret(ctx, twofactorsqlc.SealTwoFactorSecretParams{
		PublicID:            userPublicID,
		TwoFactorSecretEnc:  sealed,
		TwoFactorKeyVersion: int32(version),
	})
	return affectingOne(affected, err, "seal two-factor secret")
}

func (r *SQLRepository) Enable(ctx context.Context, userPublicID string, lastStep int64) error {
	affected, err := r.queries.EnableTwoFactor(ctx, twofactorsqlc.EnableTwoFactorParams{
		PublicID:          userPublicID,
		TwoFactorLastStep: lastStep,
	})
	return affectingOne(affected, err, "enable two-factor")
}

func (r *SQLRepository) Disable(ctx context.Context, userPublicID string) error {
	affected, err := r.queries.DisableTwoFactor(ctx, userPublicID)
	return affectingOne(affected, err, "disable two-factor")
}

// ConsumeStep records an accepted TOTP step; zero rows means the code was
// already used for that step (or an earlier one).
func (r *SQLRepository) ConsumeStep(ctx context.Context, userPublicID string, step int64) error {
	affected, err := r.queries.ConsumeTwoFactorStep(ctx, twofactorsqlc.ConsumeTwoFactorStepParams{
		PublicID:          userPublicID,
		TwoFactorLastStep: step,
	})
	return affectingOne(affected, err, "consume two-factor step")
}

func (r *SQLRepository) ReplaceRecoveryCodes(
	ctx context.Context,
	userPublicID string,
	hashes []string,
) error {
	if err := r.queries.DeleteRecoveryCodes(ctx, userPublicID); err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}
	if len(hashes) == 0 {
		return nil
	}
	if err := r.queries.InsertRecoveryCodes(ctx, twofactorsqlc.InsertRecoveryCodesParams{
		PublicID:   userPublicID,
		CodeHashes: hashes,
	}); err != nil {
		return fmt.Errorf("insert recovery codes: %w", err)
	}
	return nil
}

func (r *SQLRepository) UnusedRecoveryCodes(
	ctx context.Context,
	userPublicID string,
) ([]RecoveryCode, error) {
	rows, err := r.queries.ListUnusedRecoveryCodes(ctx, userPublicID)
	if err != nil {
		return nil, fmt.Errorf("list recovery codes: %w", err)
	}
	codes := make([]RecoveryCode, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, RecoveryCode{ID: row.ID, Hash: row.CodeHash})
	}
	return codes, nil
}

func (r *SQLRepository) MarkRecoveryCodeUsed(ctx context.Context, id int64) error {
	affected, err := r.queries.MarkRecoveryCodeUsed(ctx, id)
	return affectingOne(affected, err, "mark recovery code used")
}

func (r *SQLRepository) ReplaceRecoveryCodeHash(ctx context.Context, id int64, hash string) error {
	affected, err := r.queries.ReplaceRecoveryCodeHash(
		ctx,
		twofactorsqlc.ReplaceRecoveryCodeHashParams{
			ID:       id,
			CodeHash: hash,
		},
	)
	return affectingOne(affected, err, "replace recovery code hash")
}

func (r *SQLRepository) LegacySecrets(ctx context.Context, limit int) ([]LegacySecret, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.ListLegacyTwoFactorSecrets(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("list legacy two-factor secrets: %w", err)
	}
	secrets := make([]LegacySecret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, LegacySecret{PublicID: row.PublicID, Secret: row.TwoFactorSecret})
	}
	return secrets, nil
}

func affectingOne(affected int64, err error, action string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
