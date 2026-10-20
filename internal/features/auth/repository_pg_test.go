package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func identityTestRepository(t *testing.T) (*SQLRepository, *sqlx.DB) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL is unavailable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var table string
	if err := db.GetContext(ctx, &table, `SELECT to_regclass('user_identities')`); err != nil ||
		table == "" {
		t.Skip("user_identities migration is not applied")
	}
	repository, err := NewSQLRepository(RepositoryDependency{DB: db})
	if err != nil {
		t.Fatalf("build repository: %v", err)
	}
	t.Cleanup(func() {
		_ = repository.Close()
		_ = db.Close()
	})
	return repository, db
}

func TestSQLIdentityBindingRoundTrip(t *testing.T) {
	repository, db := identityTestRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	seed := fmt.Sprintf("identity-test-%d", time.Now().UnixNano())
	first := seed + "-a"
	second := seed + "-b"
	for _, publicID := range []string{first, second} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO users (public_id, username, password_hash) VALUES ($1, $1, 'x')`,
			publicID,
		); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM user_identities WHERE subject = $1`, seed)
		_, _ = db.Exec(`DELETE FROM users WHERE public_id IN ($1, $2)`, first, second)
	})

	subject := seed
	if err := repository.UpsertIdentity(ctx, first, "test-provider", subject, "first@example.com"); err != nil {
		t.Fatalf("bind identity: %v", err)
	}
	if err := repository.UpsertIdentity(ctx, first, "test-provider", subject, "renamed@example.com"); err != nil {
		t.Fatalf("refresh identity: %v", err)
	}
	resolved, err := repository.FindIdentityUser(ctx, "test-provider", subject)
	if err != nil {
		t.Fatalf("find identity: %v", err)
	}
	if resolved.PublicID != first {
		t.Fatalf("identity resolved to %q, want %q", resolved.PublicID, first)
	}

	if err := repository.UpsertIdentity(ctx, second, "test-provider", subject, "second@example.com"); !errors.Is(err, ErrIdentityConflict) {
		t.Fatalf("expected ErrIdentityConflict on rebinding, got %v", err)
	}
	if _, err := repository.FindIdentityUser(ctx, "test-provider", subject+"-missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
