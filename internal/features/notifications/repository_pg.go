package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const notificationProjection = `
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
ORDER BY created_at DESC, id DESC
LIMIT $2`

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

// SQLRepository is the PostgreSQL notification repository.
type SQLRepository struct {
	owner        *platformdatabase.StatementOwner
	listStmt     *sqlx.Stmt
	unreadStmt   *sqlx.Stmt
	createStmt   *sqlx.Stmt
	markReadStmt *sqlx.Stmt
	markAllStmt  *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to build the repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL notification repository.
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

// Close releases the prepared statements owned by the repository.
func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

// List returns the newest notifications of the recipient.
func (r *SQLRepository) List(
	ctx context.Context,
	recipientPublicID string,
	limit int,
) ([]Notification, error) {
	items := make([]Notification, 0)
	if err := r.listStmt.SelectContext(ctx, &items, recipientPublicID, limit); err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

// UnreadCount returns how many notifications are still unread.
func (r *SQLRepository) UnreadCount(ctx context.Context, recipientPublicID string) (int, error) {
	var count int
	if err := r.unreadStmt.GetContext(ctx, &count, recipientPublicID); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// Create persists one notification.
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

// MarkRead marks one owned notification as read.
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

// MarkAllRead marks every unread notification of the recipient as read.
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
