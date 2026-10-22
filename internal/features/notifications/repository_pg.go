package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const notificationProjection = `
id,
public_id,
recipient_public_id,
kind,
title,
body,
status,
created_at,
read_at`

const (
	listNotificationsQuery = `
SELECT ` + notificationProjection + `
FROM notifications
WHERE recipient_public_id = $1
  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3))
ORDER BY created_at DESC, id DESC
LIMIT $4`

	unreadCountQuery = `
SELECT count(*)
FROM notifications
WHERE recipient_public_id = $1
  AND status = 'unread'`

	createNotificationQuery = `
INSERT INTO notifications (public_id, recipient_public_id, kind, title, body)
VALUES ($1, $2, $3, $4, $5)
RETURNING ` + notificationProjection

	markNotificationReadQuery = `
UPDATE notifications
SET status = 'read',
    read_at = COALESCE(read_at, now())
WHERE recipient_public_id = $1
  AND public_id = $2
RETURNING ` + notificationProjection

	markAllNotificationsReadQuery = `
UPDATE notifications
SET status = 'read',
    read_at = COALESCE(read_at, now())
WHERE recipient_public_id = $1
  AND status = 'unread'`
)

type SQLRepository struct {
	owner        *platformdatabase.StatementOwner
	listStmt     *sqlx.Stmt
	unreadStmt   *sqlx.Stmt
	createStmt   *sqlx.Stmt
	markReadStmt *sqlx.Stmt
	markAllStmt  *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sqlx.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "notification database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.listStmt, "notification_list", listNotificationsQuery},
		{&repository.unreadStmt, "notification_unread", unreadCountQuery},
		{&repository.createStmt, "notification_create", createNotificationQuery},
		{&repository.markReadStmt, "notification_mark_read", markNotificationReadQuery},
		{&repository.markAllStmt, "notification_mark_all", markAllNotificationsReadQuery},
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

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

// List returns one keyset window of the recipient's notifications, newest first.
// The tuple comparison keeps the window stable while new notifications arrive.
func (r *SQLRepository) List(
	ctx context.Context,
	recipientPublicID string,
	query ListQuery,
) ([]Notification, error) {
	var before *time.Time
	var beforeID int64
	if query.Before != nil {
		before = &query.Before.CreatedAt
		beforeID = query.Before.ID
	}
	items := make([]Notification, 0)
	if err := r.listStmt.SelectContext(
		ctx,
		&items,
		recipientPublicID,
		before,
		beforeID,
		query.Limit,
	); err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) UnreadCount(ctx context.Context, recipientPublicID string) (int, error) {
	var count int
	if err := r.unreadStmt.GetContext(ctx, &count, recipientPublicID); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *SQLRepository) Create(ctx context.Context, input CreateInput) (Notification, error) {
	var created Notification
	if err := r.createStmt.GetContext(
		ctx,
		&created,
		input.PublicID,
		input.RecipientPublicID,
		input.Kind,
		input.Title,
		input.Body,
	); err != nil {
		return Notification{}, fmt.Errorf("create notification: %w", err)
	}
	return created, nil
}

func (r *SQLRepository) MarkRead(
	ctx context.Context,
	recipientPublicID, publicID string,
) (Notification, error) {
	var updated Notification
	if err := r.markReadStmt.GetContext(ctx, &updated, recipientPublicID, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, fmt.Errorf("mark notification read: %w", err)
	}
	return updated, nil
}

func (r *SQLRepository) MarkAllRead(ctx context.Context, recipientPublicID string) (int, error) {
	result, err := r.markAllStmt.ExecContext(ctx, recipientPublicID)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark all notifications rows: %w", err)
	}
	return int(affected), nil
}
