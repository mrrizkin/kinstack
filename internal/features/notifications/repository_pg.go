package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	notificationssqlc "kinstack/internal/features/notifications/sqlc"
	"kinstack/internal/pkg/assert"
)

type SQLRepository struct {
	queries *notificationssqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "notification database is required")

	queries, err := notificationssqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare notification queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

// List returns one keyset window of the recipient's notifications, newest first.
// The tuple comparison keeps the window stable while new notifications arrive.
func (r *SQLRepository) List(
	ctx context.Context,
	recipientPublicID string,
	query ListQuery,
) ([]Notification, error) {
	if query.Before == nil {
		rows, err := r.queries.ListNotifications(ctx, notificationssqlc.ListNotificationsParams{
			RecipientPublicID: recipientPublicID,
			RowLimit:          int32(query.Limit),
		})
		if err != nil {
			return nil, fmt.Errorf("list notifications: %w", err)
		}
		items := make([]Notification, 0, len(rows))
		for _, row := range rows {
			items = append(items, notificationFromListRow(row))
		}
		return items, nil
	}
	rows, err := r.queries.ListNotificationsBefore(
		ctx,
		notificationssqlc.ListNotificationsBeforeParams{
			RecipientPublicID: recipientPublicID,
			Before:            query.Before.CreatedAt,
			BeforeID:          query.Before.ID,
			RowLimit:          int32(query.Limit),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	items := make([]Notification, 0, len(rows))
	for _, row := range rows {
		items = append(items, notificationFromListBeforeRow(row))
	}
	return items, nil
}

func (r *SQLRepository) UnreadCount(ctx context.Context, recipientPublicID string) (int, error) {
	count, err := r.queries.CountUnreadNotifications(ctx, recipientPublicID)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return int(count), nil
}

func (r *SQLRepository) Create(ctx context.Context, input CreateInput) (Notification, error) {
	row, err := r.queries.CreateNotification(ctx, notificationssqlc.CreateNotificationParams{
		PublicID:          input.PublicID,
		RecipientPublicID: input.RecipientPublicID,
		Kind:              input.Kind,
		Title:             input.Title,
		Body:              input.Body,
	})
	if err != nil {
		return Notification{}, fmt.Errorf("create notification: %w", err)
	}
	return notificationFromCreateRow(row), nil
}

func (r *SQLRepository) MarkRead(
	ctx context.Context,
	recipientPublicID, publicID string,
) (Notification, error) {
	row, err := r.queries.MarkNotificationRead(ctx, notificationssqlc.MarkNotificationReadParams{
		RecipientPublicID: recipientPublicID,
		PublicID:          publicID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	if err != nil {
		return Notification{}, fmt.Errorf("mark notification read: %w", err)
	}
	return notificationFromMarkReadRow(row), nil
}

func (r *SQLRepository) MarkAllRead(ctx context.Context, recipientPublicID string) (int, error) {
	affected, err := r.queries.MarkAllNotificationsRead(ctx, recipientPublicID)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return int(affected), nil
}

func notificationFromListRow(row notificationssqlc.ListNotificationsRow) Notification {
	return newNotification(
		row.ID,
		row.PublicID,
		row.RecipientPublicID,
		row.Kind,
		row.Title,
		row.Body,
		row.Status,
		row.CreatedAt,
		nullableTime(row.ReadAt),
	)
}

func notificationFromListBeforeRow(
	row notificationssqlc.ListNotificationsBeforeRow,
) Notification {
	return newNotification(
		row.ID,
		row.PublicID,
		row.RecipientPublicID,
		row.Kind,
		row.Title,
		row.Body,
		row.Status,
		row.CreatedAt,
		nullableTime(row.ReadAt),
	)
}

func notificationFromCreateRow(row notificationssqlc.CreateNotificationRow) Notification {
	return newNotification(
		row.ID,
		row.PublicID,
		row.RecipientPublicID,
		row.Kind,
		row.Title,
		row.Body,
		row.Status,
		row.CreatedAt,
		nullableTime(row.ReadAt),
	)
}

func notificationFromMarkReadRow(row notificationssqlc.MarkNotificationReadRow) Notification {
	return newNotification(
		row.ID,
		row.PublicID,
		row.RecipientPublicID,
		row.Kind,
		row.Title,
		row.Body,
		row.Status,
		row.CreatedAt,
		nullableTime(row.ReadAt),
	)
}

func newNotification(
	id int64,
	publicID string,
	recipientPublicID string,
	kind string,
	title string,
	body string,
	status string,
	createdAt time.Time,
	readAt *time.Time,
) Notification {
	return Notification{
		ID:                id,
		PublicID:          publicID,
		RecipientPublicID: recipientPublicID,
		Kind:              kind,
		Title:             title,
		Body:              body,
		Status:            Status(status),
		CreatedAt:         createdAt,
		ReadAt:            readAt,
	}
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
