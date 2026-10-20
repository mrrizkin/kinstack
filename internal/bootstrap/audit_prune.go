package bootstrap

import (
	"context"
	"fmt"
	"os"
	"time"

	"kinstack/config"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/database"
)

// PruneAudit deletes audit records older than retention. Audit rows are never
// removed by the request path, so a scheduled run of this command is what
// actually enforces the retention target.
func PruneAudit(retention time.Duration) error {
	if retention <= 0 {
		return fmt.Errorf("audit retention must be positive")
	}

	cfg := config.Load()

	db, err := database.OpenWithOptions(cfg.DB.URL, database.Options{
		StatementTimeout: cfg.DB.StatementTimeout,
		MaxOpenConns:     cfg.DB.MaxOpenConns,
		MaxIdleConns:     cfg.DB.MaxIdleConns,
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	repository, err := audit.NewSQLRepository(audit.RepositoryDependency{DB: db.DB})
	if err != nil {
		return fmt.Errorf("configure audit repository: %w", err)
	}
	defer func() { _ = repository.Close() }()

	cutoff := time.Now().UTC().Add(-retention)
	deleted, err := audit.New(repository).Prune(context.Background(), cutoff)
	if err != nil {
		return fmt.Errorf("prune audit events: %w", err)
	}
	fmt.Fprintf(os.Stdout, "pruned %d audit records older than %s\n", deleted, retention)
	return nil
}
