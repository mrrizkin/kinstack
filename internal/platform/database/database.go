// Package database owns the PostgreSQL connection and Goose migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"kinstack/db/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	openConnectionsMax = 10
	idleConnectionsMax = 5
	connLifetimeMax    = 30 * time.Minute
	connIdleTimeMax    = 5 * time.Minute
)

type DB struct {
	*sql.DB
}

// Options tunes the shared pool. Zero values keep the defaults.
type Options struct {
	StatementTimeout time.Duration
	OpenConnsMax     int
	IdleConnsMax     int
}

// Open connects with the default pool settings. Migration commands use this
// path so long-running schema changes are not cut off by a query timeout.
func Open(dsn string) (*DB, error) {
	return OpenWithOptions(dsn, Options{})
}

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

	db := stdlib.OpenDB(*connConfig)
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	db.SetMaxOpenConns(poolSize(opts.OpenConnsMax, openConnectionsMax))
	db.SetMaxIdleConns(poolSize(opts.IdleConnsMax, idleConnectionsMax))
	db.SetConnMaxLifetime(connLifetimeMax)
	db.SetConnMaxIdleTime(connIdleTimeMax)
	return &DB{DB: db}, nil
}

// poolSize applies an explicit pool setting, falling back to the default when
// the caller did not configure one.
func poolSize(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func (db *DB) Close() error {
	if db == nil || db.DB == nil {
		return nil
	}
	return db.DB.Close()
}

func (db *DB) Migrate(ctx context.Context) error {
	return db.withGoose(func() error { return goose.UpContext(ctx, db.DB, ".") })
}

func (db *DB) Down(ctx context.Context) error {
	return db.withGoose(func() error { return goose.DownContext(ctx, db.DB, ".") })
}

func (db *DB) Status(ctx context.Context) error {
	return db.withGoose(func() error { return goose.StatusContext(ctx, db.DB, ".") })
}

func (db *DB) Version(ctx context.Context) (int64, error) {
	var version int64
	err := db.withGoose(func() error {
		var err error
		version, err = goose.GetDBVersionContext(ctx, db.DB)
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
