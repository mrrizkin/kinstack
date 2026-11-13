package users

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"kinstack/internal/pkg/pagination"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// usersTestRepository skips unless PostgreSQL is available: the suite's substrate
// is repository_mem, so the generated read queries are checked where a real
// schema exists instead of being mocked.
func usersTestRepository(t *testing.T) (*SQLRepository, *sqlx.DB) {
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
	if err := db.GetContext(ctx, &table, `SELECT to_regclass('users')`); err != nil || table == "" {
		t.Skip("the users migration is not applied")
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

func TestSQLReadQueriesMatchTheSchema(t *testing.T) {
	repository, db := usersTestRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	seed := fmt.Sprintf("read-test-%d", time.Now().UnixNano())
	username := seed + "-user"
	created, err := repository.CreateUser(ctx, CreateUserInput{
		PublicID:     seed,
		Username:     username,
		Email:        username + "@example.com",
		DisplayName:  "Read Test",
		PasswordHash: "x",
		Invited:      true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE public_id = $1`, seed)
	})

	if created.Invited != true {
		t.Fatalf("created user should be invited")
	}

	found, err := repository.FindUserByPublicID(ctx, seed)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if found.Username != username || found.DisplayName != "Read Test" || !found.Invited {
		t.Fatalf("find returned %+v", found)
	}
	if found.LastLoginAt != nil {
		t.Fatalf("a fresh account has no last login, got %v", found.LastLoginAt)
	}

	if _, err := repository.FindUserByPublicID(ctx, seed+"-missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	listed, err := repository.ListUsers(ctx, pagination.New(1, 100))
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if !containsUser(listed, seed) {
		t.Fatalf("list did not include the seeded account")
	}

	counts, err := repository.CountUsers(ctx)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if counts.Total < 1 || counts.Active < 1 {
		t.Fatalf("counts look wrong: %+v", counts)
	}
}

func containsUser(results []UserResult, publicID string) bool {
	for _, result := range results {
		if result.PublicID == publicID {
			return true
		}
	}
	return false
}
