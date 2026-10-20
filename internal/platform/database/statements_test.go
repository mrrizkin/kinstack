package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jmoiron/sqlx"
)

const statementOwnerTestDriverName = "kinstack_statement_owner_test"

var (
	statementOwnerTestTrackers sync.Map
	statementOwnerTestID       atomic.Uint64
	errStatementOwnerPrepare   = errors.New("prepare failed")
)

type statementOwnerTestDriver struct{}

func init() {
	sql.Register(statementOwnerTestDriverName, statementOwnerTestDriver{})
}

func (statementOwnerTestDriver) Open(name string) (driver.Conn, error) {
	value, ok := statementOwnerTestTrackers.Load(name)
	if !ok {
		return nil, fmt.Errorf("statement owner test tracker %q is not registered", name)
	}
	return &statementOwnerTestConn{tracker: value.(*statementOwnerTestTracker)}, nil
}

type statementOwnerTestConn struct {
	tracker       *statementOwnerTestTracker
	mu            sync.Mutex
	inTransaction bool
}

func (c *statementOwnerTestConn) Prepare(query string) (driver.Stmt, error) {
	c.tracker.mu.Lock()
	c.tracker.prepared = append(c.tracker.prepared, query)
	fail := query == c.tracker.failQuery
	closeErr := c.tracker.closeErrors[query]
	c.tracker.mu.Unlock()
	if fail {
		return nil, errStatementOwnerPrepare
	}
	return &statementOwnerTestStmt{conn: c, query: query, closeErr: closeErr}, nil
}

func (*statementOwnerTestConn) Close() error { return nil }

func (c *statementOwnerTestConn) Begin() (driver.Tx, error) {
	c.mu.Lock()
	c.inTransaction = true
	c.mu.Unlock()
	return statementOwnerTestTx{conn: c}, nil
}

type statementOwnerTestTx struct {
	conn *statementOwnerTestConn
}

func (tx statementOwnerTestTx) Commit() error {
	tx.conn.mu.Lock()
	tx.conn.inTransaction = false
	tx.conn.mu.Unlock()
	return nil
}

func (tx statementOwnerTestTx) Rollback() error {
	tx.conn.mu.Lock()
	tx.conn.inTransaction = false
	tx.conn.mu.Unlock()
	return nil
}

type statementOwnerTestStmt struct {
	conn     *statementOwnerTestConn
	query    string
	closeErr error
}

func (s *statementOwnerTestStmt) Close() error {
	s.conn.tracker.mu.Lock()
	s.conn.tracker.closed = append(s.conn.tracker.closed, s.query)
	s.conn.tracker.mu.Unlock()
	return s.closeErr
}

func (*statementOwnerTestStmt) NumInput() int { return -1 }

func (s *statementOwnerTestStmt) Exec([]driver.Value) (driver.Result, error) {
	s.conn.mu.Lock()
	inTransaction := s.conn.inTransaction
	s.conn.mu.Unlock()
	s.conn.tracker.mu.Lock()
	s.conn.tracker.execInTransaction = append(s.conn.tracker.execInTransaction, inTransaction)
	s.conn.tracker.mu.Unlock()
	return driver.RowsAffected(1), nil
}

func (*statementOwnerTestStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, errors.New("queries are not supported by the statement owner test driver")
}

type statementOwnerTestTracker struct {
	mu                sync.Mutex
	failQuery         string
	closeErrors       map[string]error
	prepared          []string
	closed            []string
	execInTransaction []bool
}

func newStatementOwnerTestDB(t *testing.T) (*sqlx.DB, *statementOwnerTestTracker) {
	t.Helper()
	name := fmt.Sprintf("test-%d", statementOwnerTestID.Add(1))
	tracker := &statementOwnerTestTracker{closeErrors: make(map[string]error)}
	statementOwnerTestTrackers.Store(name, tracker)
	t.Cleanup(func() { statementOwnerTestTrackers.Delete(name) })

	db, err := sql.Open(statementOwnerTestDriverName, name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, statementOwnerTestDriverName), tracker
}

func TestStatementOwnerPrepareAndClose(t *testing.T) {
	db, tracker := newStatementOwnerTestDB(t)
	owner := NewStatementOwner(db)

	if _, err := owner.Prepare("first", "SELECT 1"); err != nil {
		t.Fatalf("prepare first statement: %v", err)
	}
	if _, err := owner.Prepare("second", "SELECT 2"); err != nil {
		t.Fatalf("prepare second statement: %v", err)
	}

	if err := owner.Close(); err != nil {
		t.Fatalf("close statement owner: %v", err)
	}
	if err := owner.Close(); err != nil {
		t.Fatalf("close statement owner a second time: %v", err)
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if got, want := tracker.prepared, []string{"SELECT 1", "SELECT 2"}; !equalStrings(got, want) {
		t.Fatalf("prepared queries=%v, want %v", got, want)
	}
	if got, want := tracker.closed, []string{"SELECT 1", "SELECT 2"}; !equalStrings(got, want) {
		t.Fatalf("closed queries=%v, want %v", got, want)
	}
}

func TestStatementOwnerDoesNotCacheExplicitPreparations(t *testing.T) {
	db, tracker := newStatementOwnerTestDB(t)
	owner := NewStatementOwner(db)

	for range 2 {
		if _, err := owner.Prepare("same-query", "SELECT 1"); err != nil {
			t.Fatalf("prepare statement: %v", err)
		}
	}
	if err := owner.Close(); err != nil {
		t.Fatalf("close statement owner: %v", err)
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if got, want := tracker.prepared, []string{"SELECT 1", "SELECT 1"}; !equalStrings(got, want) {
		t.Fatalf("prepared queries=%v, want %v", got, want)
	}
	if got, want := tracker.closed, []string{"SELECT 1", "SELECT 1"}; !equalStrings(got, want) {
		t.Fatalf("closed queries=%v, want %v", got, want)
	}
}

func TestStatementOwnerStatementBindsToTransaction(t *testing.T) {
	db, tracker := newStatementOwnerTestDB(t)
	db.SetMaxOpenConns(1)
	owner := NewStatementOwner(db)
	stmt, err := owner.Prepare("transactional", "SELECT 1")
	if err != nil {
		t.Fatalf("prepare statement: %v", err)
	}
	defer owner.Close()

	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if _, err := tx.Stmtx(stmt).ExecContext(context.Background()); err != nil {
		t.Fatalf("execute transaction-bound statement: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if got, want := tracker.execInTransaction, []bool{true}; !equalBools(got, want) {
		t.Fatalf("statement execution transaction state=%v, want %v", got, want)
	}
}

func equalBools(got, want []bool) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestStatementOwnerPrepareFailureCleansUpPartialConstruction(t *testing.T) {
	db, tracker := newStatementOwnerTestDB(t)
	tracker.failQuery = "SELECT broken"
	owner := NewStatementOwner(db)

	if _, err := owner.Prepare("first", "SELECT 1"); err != nil {
		t.Fatalf("prepare first statement: %v", err)
	}
	if _, err := owner.Prepare("broken", "SELECT broken"); err == nil {
		t.Fatal("prepare broken statement succeeded")
	} else {
		if !errors.Is(err, errStatementOwnerPrepare) {
			t.Fatalf("prepare error=%v, want underlying prepare error", err)
		}
		if !strings.Contains(err.Error(), `prepare statement "broken"`) {
			t.Fatalf("prepare error=%v, want statement name context", err)
		}
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if got, want := tracker.closed, []string{"SELECT 1"}; !equalStrings(got, want) {
		t.Fatalf("closed queries=%v, want %v", got, want)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
