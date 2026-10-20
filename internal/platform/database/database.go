// Package database owns the PostgreSQL connection and Goose migrations.
package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"kinstack/db/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

const (
	maxOpenConnections = 10
	maxIdleConnections = 5
	maxConnLifetime    = 30 * time.Minute
	maxConnIdleTime    = 5 * time.Minute
)

type DB struct {
	*sqlx.DB
}

// Options tunes the shared pool. Zero values keep the defaults.
type Options struct {
	StatementTimeout time.Duration
}

// Open connects with the default pool settings. Migration commands use this
// path so long-running schema changes are not cut off by a query timeout.
func Open(dsn string) (*DB, error) {
	return OpenWithOptions(dsn, Options{})
}

// OpenWithOptions connects with pool and per-connection runtime parameters.
func OpenWithOptions(dsn string, opts Options) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN is required")
	}
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}
	if opts.StatementTimeout > 0 {
		connConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(
			opts.StatementTimeout.Milliseconds(),
			10,
		)
	}

	db := sqlx.NewDb(stdlib.OpenDB(*connConfig), "pgx")
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(maxConnLifetime)
	db.SetConnMaxIdleTime(maxConnIdleTime)
	return &DB{DB: db}, nil
}

func (db *DB) Close() error {
	if db == nil || db.DB == nil {
		return nil
	}
	return db.DB.Close()
}

func (db *DB) Migrate(ctx context.Context) error {
	return db.withGoose(func() error { return goose.UpContext(ctx, db.DB.DB, ".") })
}

func (db *DB) Down(ctx context.Context) error {
	return db.withGoose(func() error { return goose.DownContext(ctx, db.DB.DB, ".") })
}

func (db *DB) Status(ctx context.Context) error {
	return db.withGoose(func() error { return goose.StatusContext(ctx, db.DB.DB, ".") })
}

func (db *DB) Version(ctx context.Context) (int64, error) {
	var version int64
	err := db.withGoose(func() error {
		var err error
		version, err = goose.GetDBVersionContext(ctx, db.DB.DB)
		return err
	})
	return version, err
}

func (db *DB) withGoose(run func() error) error {
	if err := migrations.Validate(); err != nil {
		return fmt.Errorf("validate migrations: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	goose.SetBaseFS(migrations.Files)
	return run()
}
