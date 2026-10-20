package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestMigrationLatestRollbackRoundTrip(t *testing.T) {
	dsn := os.Getenv("KINSTACK_MIGRATION_ROLLBACK_DATABASE_URL")
	if dsn == "" {
		t.Skip("KINSTACK_MIGRATION_ROLLBACK_DATABASE_URL is not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL is unavailable: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database := &DB{DB: db}
	if err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	before, err := database.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before == 0 {
		t.Fatal("migration version was not recorded")
	}
	if err := database.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := database.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("migration version after rollback round trip=%d, want %d", after, before)
	}
}
