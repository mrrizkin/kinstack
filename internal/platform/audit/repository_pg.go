package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"kinstack/internal/pkg/assert"
	platformauditsqlc "kinstack/internal/platform/audit/sqlc"
)

type sqlRepository struct {
	queries *platformauditsqlc.Queries
}

var _ Repository = (*sqlRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*sqlRepository, error) {
	assert.Assert(dependency.DB != nil, "audit database is required")

	queries, err := platformauditsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare audit queries: %w", err)
	}
	return &sqlRepository{queries: queries}, nil
}

func (r *sqlRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

// pruneAuditBatchSize bounds one DELETE so pruning a large table does not hold
// locks for the whole operation.
const pruneAuditBatchSize = 5_000

func (r *sqlRepository) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		deleted, err := r.queries.DeleteAuditBatchBefore(
			ctx,
			platformauditsqlc.DeleteAuditBatchBeforeParams{
				OccurredAt: cutoff,
				RowLimit:   pruneAuditBatchSize,
			},
		)
		if err != nil {
			return total, fmt.Errorf("prune audit events: %w", err)
		}
		total += deleted
		if deleted < pruneAuditBatchSize {
			return total, nil
		}
	}
}

func (r *sqlRepository) Insert(ctx context.Context, record AuditRecord) (AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return AuditRecord{}, err
	}
	metadata := encodeMetadata(record.Metadata)
	id, err := r.queries.InsertAuditEvent(ctx, platformauditsqlc.InsertAuditEventParams{
		OccurredAt: record.OccurredAt,
		RequestID:  record.RequestID,
		ActorKey:   record.ActorKey,
		Action:     string(record.Action),
		Resource:   record.Resource,
		Method:     record.Method,
		Status:     int32(record.Status),
		Outcome:    string(record.Outcome),
		Metadata:   json.RawMessage(metadata),
	})
	if err != nil {
		return AuditRecord{}, fmt.Errorf("record audit event: %w", err)
	}
	record.ID = id
	return recordWithClonedMetadata(record), nil
}

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

	rows, err := r.queries.ListAuditEvents(ctx, platformauditsqlc.ListAuditEventsParams{
		ActorKey: optionalString(query.ActorKey),
		Action:   optionalAction(query.Action),
		RowLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}

	records := make([]AuditRecord, 0, len(rows))
	for _, row := range rows {
		metadata, err := decodeMetadata(row.Metadata)
		if err != nil {
			return nil, fmt.Errorf("decode audit event metadata: %w", err)
		}
		records = append(records, AuditRecord{
			ID:         row.ID,
			OccurredAt: row.OccurredAt,
			RequestID:  row.RequestID,
			ActorKey:   row.ActorKey,
			Action:     Action(row.Action),
			Resource:   row.Resource,
			Method:     row.Method,
			Status:     int(row.Status),
			Outcome:    Outcome(row.Outcome),
			Metadata:   metadata,
		})
	}
	return records, nil
}

func optionalString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func optionalAction(value Action) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(value), Valid: true}
}

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
