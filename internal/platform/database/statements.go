package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// StatementOwner owns prepared statements for a repository. It deliberately
// keeps an ordered list rather than a query cache: each statement is prepared
// explicitly by the repository constructor and closed with the repository.
type StatementOwner struct {
	db         *sqlx.DB
	statements []ownedStatement
	closed     bool
}

type ownedStatement struct {
	name string
	stmt *sqlx.Stmt
}

// NewStatementOwner creates an owner for statements prepared from db.
func NewStatementOwner(db *sqlx.DB) *StatementOwner {
	return &StatementOwner{db: db}
}

// Prepare prepares and takes ownership of a named statement. If preparation
// fails, statements already owned by the owner are closed before the error is
// returned.
func (o *StatementOwner) Prepare(name, query string) (*sqlx.Stmt, error) {
	return o.PrepareContext(context.Background(), name, query)
}

func (o *StatementOwner) PrepareContext(
	ctx context.Context,
	name, query string,
) (*sqlx.Stmt, error) {
	if o == nil {
		return nil, fmt.Errorf("prepare statement %q: statement owner is nil", name)
	}
	if o.db == nil {
		return nil, fmt.Errorf("prepare statement %q: database is required", name)
	}
	if o.closed {
		return nil, fmt.Errorf("prepare statement %q: statement owner is closed", name)
	}

	stmt, err := o.db.PreparexContext(ctx, query)
	if err != nil {
		prepareErr := fmt.Errorf("prepare statement %q: %w", name, err)
		return nil, errors.Join(prepareErr, o.Close())
	}
	o.statements = append(o.statements, ownedStatement{name: name, stmt: stmt})
	return stmt, nil
}

// Close releases every statement owned by o. It attempts all closes and joins
// any errors so one failing statement does not leak the remaining statements.
func (o *StatementOwner) Close() error {
	if o == nil || o.closed {
		return nil
	}
	o.closed = true

	errs := make([]error, 0)
	for _, owned := range o.statements {
		if owned.stmt == nil {
			continue
		}
		if err := owned.stmt.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close statement %q: %w", owned.name, err))
		}
	}
	o.statements = nil
	return errors.Join(errs...)
}
