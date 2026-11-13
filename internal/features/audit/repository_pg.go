package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	auditsqlc "kinstack/internal/features/audit/sqlc"
	"kinstack/internal/pkg/assert"
)

// SQLRepository reads audit_events directly. The projection is append-only and
// written by platform/audit, so this repository is read-only.
type SQLRepository struct {
	queries *auditsqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "audit database is required")

	queries, err := auditsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare audit queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) List(ctx context.Context, query Query) (Page, error) {
	filter := auditFilter(query)

	total, err := r.queries.CountAuditEvents(ctx, auditsqlc.CountAuditEventsParams{
		ActorKey: filter.actorKey,
		Action:   filter.action,
		Outcome:  filter.outcome,
		Search:   filter.search,
		FromTime: filter.from,
		ToTime:   filter.to,
	})
	if err != nil {
		return Page{}, fmt.Errorf("count audit events: %w", err)
	}

	rows, err := r.queries.ListAuditEvents(ctx, auditsqlc.ListAuditEventsParams{
		ActorKey:  filter.actorKey,
		Action:    filter.action,
		Outcome:   filter.outcome,
		Search:    filter.search,
		FromTime:  filter.from,
		ToTime:    filter.to,
		RowLimit:  int32(query.Limit()),
		RowOffset: int32(query.Offset()),
	})
	if err != nil {
		return Page{}, fmt.Errorf("list audit events: %w", err)
	}

	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		metadata, err := decodeMetadata(row.Metadata)
		if err != nil {
			return Page{}, fmt.Errorf("decode audit metadata: %w", err)
		}
		entries = append(entries, Entry{
			ID:         row.ID,
			OccurredAt: row.OccurredAt,
			RequestID:  row.RequestID,
			ActorKey:   row.ActorKey,
			Action:     row.Action,
			Resource:   row.Resource,
			Method:     row.Method,
			Status:     int(row.Status),
			Outcome:    row.Outcome,
			Metadata:   metadata,
		})
	}
	return Page{Entries: entries, Total: int(total)}, nil
}

type auditFilterParams struct {
	actorKey sql.NullString
	action   sql.NullString
	outcome  sql.NullString
	search   sql.NullString
	from     sql.NullTime
	to       sql.NullTime
}

func auditFilter(query Query) auditFilterParams {
	return auditFilterParams{
		actorKey: optionalString(query.ActorKey),
		action:   optionalString(query.Action),
		outcome:  optionalString(query.Outcome),
		search:   optionalSearch(query.Search),
		from:     optionalTime(query.From),
		to:       optionalTime(query.To),
	}
}

func optionalString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func optionalSearch(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: "%" + value + "%", Valid: true}
}

func optionalTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
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
