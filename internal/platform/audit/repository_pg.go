package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

// -----------------------------------------------------------------------------
// PostgreSQL repository
// -----------------------------------------------------------------------------

type sqlRepository struct {
	owner *platformdatabase.StatementOwner

	insertStmt            *sqlx.Stmt
	listStmt              *sqlx.Stmt
	listByActorStmt       *sqlx.Stmt
	listByActionStmt      *sqlx.Stmt
	listByActorActionStmt *sqlx.Stmt
}

var _ Repository = (*sqlRepository)(nil)

const (
	insertAuditQuery = `
		INSERT INTO audit_events (
			occurred_at, request_id, actor_key, action, resource, method,
			status, outcome, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
		RETURNING id`

	listAuditQuery = `
		SELECT id, occurred_at, request_id, actor_key, action, resource,
			method, status, outcome, metadata::text
		FROM audit_events
		ORDER BY occurred_at DESC, id DESC
		LIMIT $1`

	listAuditByActorQuery = `
		SELECT id, occurred_at, request_id, actor_key, action, resource,
			method, status, outcome, metadata::text
		FROM audit_events
		WHERE actor_key = $1
		ORDER BY occurred_at DESC, id DESC
		LIMIT $2`

	listAuditByActionQuery = `
		SELECT id, occurred_at, request_id, actor_key, action, resource,
			method, status, outcome, metadata::text
		FROM audit_events
		WHERE action = $1
		ORDER BY occurred_at DESC, id DESC
		LIMIT $2`

	listAuditByActorActionQuery = `
		SELECT id, occurred_at, request_id, actor_key, action, resource,
			method, status, outcome, metadata::text
		FROM audit_events
		WHERE actor_key = $1 AND action = $2
		ORDER BY occurred_at DESC, id DESC
		LIMIT $3`
)

// NewSQLRepository prepares the audit queries for the repository lifetime.
// RepositoryDependency contains the dependencies required to construct the SQL repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*sqlRepository, error) {
	assert.Assert(dependency.DB != nil, "audit database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &sqlRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.insertStmt, "audit_insert", insertAuditQuery},
		{&repository.listStmt, "audit_list", listAuditQuery},
		{&repository.listByActorStmt, "audit_list_by_actor", listAuditByActorQuery},
		{&repository.listByActionStmt, "audit_list_by_action", listAuditByActionQuery},
		{
			&repository.listByActorActionStmt,
			"audit_list_by_actor_action",
			listAuditByActorActionQuery,
		},
	}
	for _, statement := range statements {
		stmt, err := owner.Prepare(statement.name, statement.query)
		if err != nil {
			return nil, err
		}
		*statement.target = stmt
	}
	return repository, nil
}

// Close releases the prepared statements owned by the repository.
func (r *sqlRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

// Insert appends a record to audit_events.
func (r *sqlRepository) Insert(ctx context.Context, record AuditRecord) (AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return AuditRecord{}, err
	}
	metadata := encodeMetadata(record.Metadata)
	if err := r.insertStmt.GetContext(ctx, &record.ID,
		record.OccurredAt, record.RequestID, record.ActorKey, string(record.Action),
		record.Resource, record.Method, record.Status, string(record.Outcome), metadata,
	); err != nil {
		return AuditRecord{}, fmt.Errorf("record audit event: %w", err)
	}
	return recordWithClonedMetadata(record), nil
}

// List returns matching records ordered from newest to oldest.
func (r *sqlRepository) List(ctx context.Context, query RepositoryQuery) ([]AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateLimit(query.Limit); err != nil {
		return nil, err
	}
	limit := query.Limit
	if limit == 0 {
		limit = 100
	}

	stmt := r.listStmt
	args := []any{limit}
	switch {
	case query.ActorKey != "" && query.Action != "":
		stmt = r.listByActorActionStmt
		args = []any{query.ActorKey, string(query.Action), limit}
	case query.ActorKey != "":
		stmt = r.listByActorStmt
		args = []any{query.ActorKey, limit}
	case query.Action != "":
		stmt = r.listByActionStmt
		args = []any{string(query.Action), limit}
	}

	rows, err := stmt.QueryxContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	records := make([]AuditRecord, 0)
	for rows.Next() {
		var record AuditRecord
		var action string
		var outcome string
		var metadata string
		if err := rows.Scan(
			&record.ID,
			&record.OccurredAt,
			&record.RequestID,
			&record.ActorKey,
			&action,
			&record.Resource,
			&record.Method,
			&record.Status,
			&outcome,
			&metadata,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		record.Action = Action(action)
		record.Outcome = Outcome(outcome)
		record.Metadata, err = decodeMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("decode audit event metadata: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return records, nil
}

// -----------------------------------------------------------------------------
// Metadata
// -----------------------------------------------------------------------------

func encodeMetadata(metadata map[string]string) string {
	encoded, err := json.Marshal(cloneMetadata(metadata))
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func decodeMetadata(value string) (map[string]string, error) {
	if value == "" {
		return map[string]string{}, nil
	}
	metadata := make(map[string]string)
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}
