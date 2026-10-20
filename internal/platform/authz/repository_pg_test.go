package authz

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func authzTestRepository(t *testing.T) (*SQLRepository, *sqlx.DB) {
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
	if err := db.GetContext(ctx, &table, `SELECT to_regclass('roles')`); err != nil || table == "" {
		t.Skip("authorization migration is not applied")
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

func TestSQLRolePermissionsMany(t *testing.T) {
	repository, _ := authzTestRepository(t)
	ctx := context.Background()

	grouped, err := repository.RolePermissionsMany(ctx, []string{"administrator", "missing"})
	if err != nil {
		t.Fatalf("batch role permissions: %v", err)
	}
	if len(grouped["administrator"]) == 0 {
		t.Fatal("administrator permissions must be loaded")
	}
	if len(grouped["missing"]) != 0 {
		t.Fatalf("unknown role must be empty, got %+v", grouped["missing"])
	}
}

func TestSQLUserOverridesMany(t *testing.T) {
	repository, db := authzTestRepository(t)
	ctx := context.Background()

	userA := fmt.Sprintf("batch-a-%d", time.Now().UnixNano())
	userB := userA + "-b"
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM user_permissions WHERE user_public_id IN ($1, $2)`, userA, userB)
	})

	if _, err := repository.SetUserOverride(ctx, SetOverrideInput{
		UserPublicID: userA,
		Permission:   PermissionUsersView,
		Effect:       EffectAllow,
		Reason:       "batch test",
	}); err != nil {
		t.Fatalf("seed override: %v", err)
	}

	grouped, err := repository.UserOverridesMany(ctx, []string{userA, userB})
	if err != nil {
		t.Fatalf("batch overrides: %v", err)
	}
	if got := grouped[userA]; len(got) != 1 || got[0].Permission != PermissionUsersView {
		t.Fatalf("unexpected overrides for %s: %+v", userA, got)
	}
	if got := grouped[userB]; len(got) != 0 {
		t.Fatalf("user without overrides must not carry rows, got %#v", got)
	}
}
