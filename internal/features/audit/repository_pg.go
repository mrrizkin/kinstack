package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"kinstack/internal/pkg/assert"

	"github.com/jmoiron/sqlx"
)

// SQLRepository reads audit_events directly. The projection is append-only and
// written by platform/audit, so this repository is read-only.
type SQLRepository struct {
	db *sqlx.DB
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to build the repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL audit viewer repository.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "audit database is required")
	return &SQLRepository{db: dependency.DB}, nil
}

// Close releases resources owned by the repository.
func (r *SQLRepository) Close() error {
	return nil
}

const auditEntryProjection = `
id, occurred_at, request_id, actor_key, action, resource, method, status,
outcome, metadata::text`

// List returns matching audit entries plus the total number of matches.
func (r *SQLRepository) List(ctx context.Context, query Query) (Page, error) {
	where, args := buildAuditFilter(query)

	var total int
	if err := r.db.GetContext(
		ctx,
		&total,
		"SELECT count(*) FROM audit_events"+where,
		args...); err != nil {
		return Page{}, fmt.Errorf("count audit events: %w", err)
	}

	limitIndex := len(args) + 1
	offsetIndex := len(args) + 2
	listQuery := "SELECT " + auditEntryProjection + " FROM audit_events" + where +
		" ORDER BY occurred_at DESC, id DESC LIMIT $" + strconv.Itoa(limitIndex) +
		" OFFSET $" + strconv.Itoa(offsetIndex)
	listArgs := append(append([]any{}, args...), query.Limit, query.Offset)

	rows, err := r.db.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return Page{}, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		var entry Entry
		var metadata string
		if err := rows.Scan(
			&entry.ID,
			&entry.OccurredAt,
			&entry.RequestID,
			&entry.ActorKey,
			&entry.Action,
			&entry.Resource,
			&entry.Method,
			&entry.Status,
			&entry.Outcome,
			&metadata,
		); err != nil {
			return Page{}, fmt.Errorf("scan audit event: %w", err)
		}
		entry.Metadata, err = decodeMetadata(metadata)
		if err != nil {
			return Page{}, fmt.Errorf("decode audit metadata: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate audit events: %w", err)
	}
	return Page{Entries: entries, Total: total}, nil
}

func buildAuditFilter(query Query) (string, []any) {
	conditions := make([]string, 0, 6)
	args := make([]any, 0, 6)

	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(
			conditions,
			strings.Replace(condition, "?", "$"+strconv.Itoa(len(args)), 1),
		)
	}

	if query.ActorKey != "" {
		add("actor_key = ?", query.ActorKey)
	}
	if query.Action != "" {
		add("action = ?", query.Action)
	}
	if query.Outcome != "" {
		add("outcome = ?", query.Outcome)
	}
	if query.Search != "" {
		args = append(args, "%"+query.Search+"%")
		index := "$" + strconv.Itoa(len(args))
		conditions = append(conditions, "(resource ILIKE "+index+" OR request_id ILIKE "+index+
			" OR actor_key ILIKE "+index+" OR action ILIKE "+index+")")
	}
	if query.From != nil {
		add("occurred_at >= ?", *query.From)
	}
	if query.To != nil {
		add("occurred_at < ?", *query.To)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
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
