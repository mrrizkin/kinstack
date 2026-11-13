package bootstrap

import (
	"context"
	"fmt"
	"os"

	"kinstack/config"
	"kinstack/internal/platform/database"
	"kinstack/internal/platform/envelope"
	"kinstack/internal/platform/twofactor"
)

const twoFactorKeyVersion = 1

// twoFactorBox reuses the application secret under a separate HKDF info label when
// AUTH_2FA_KEY is unset, so the derived keys never collide. Local wiring without
// 32-byte material keeps secrets in plaintext, which production rejects.
func twoFactorBox(cfg config.Config) *envelope.Box {
	material := cfg.Auth.TwoFactorKey
	if material == "" {
		material = cfg.App.SecretKey
	}
	if material == "" {
		material = cfg.Auth.StateSecret
	}
	if len([]byte(material)) < envelope.KeySize {
		return nil
	}
	box, err := envelope.New(material, twofactor.RecoveryKeyInfo, twoFactorKeyVersion)
	if err != nil {
		return nil
	}
	return box
}

// BackfillTwoFactorSecrets seals every plaintext TOTP secret. It is idempotent
// and safe to run against a live database: rows are updated one at a time and a
// second run finds nothing to do.
func BackfillTwoFactorSecrets() error {
	cfg := config.Load()

	box := twoFactorBox(cfg)
	if box == nil {
		return fmt.Errorf(
			"no 32-byte key material available: set AUTH_2FA_KEY or SECRET_KEY",
		)
	}

	db, err := database.OpenWithOptions(cfg.DB.URL, database.Options{
		StatementTimeout: cfg.DB.StatementTimeout,
		OpenConnsMax:     cfg.DB.OpenConnsMax,
		IdleConnsMax:     cfg.DB.IdleConnsMax,
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	repository, err := twofactor.NewSQLRepository(twofactor.RepositoryDependency{DB: db.DB})
	if err != nil {
		return fmt.Errorf("configure two-factor repository: %w", err)
	}
	defer func() { _ = repository.Close() }()

	migrated, err := twofactor.BackfillSecrets(
		context.Background(),
		repository,
		box,
		100,
	)
	if err != nil {
		return fmt.Errorf("seal two-factor secrets: %w", err)
	}
	fmt.Fprintf(os.Stdout, "sealed %d two-factor secret(s)\n", migrated)
	return nil
}
