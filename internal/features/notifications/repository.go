package notifications

import (
	"context"
	"time"
)

// Notification is the persistence projection of one in-app notification.
type Notification struct {
	ID                int64      `db:"id"`
	PublicID          string     `db:"public_id"`
	RecipientPublicID string     `db:"recipient_public_id"`
	Kind              string     `db:"kind"`
	Title             string     `db:"title"`
	Body              string     `db:"body"`
	Status            string     `db:"status"`
	CreatedAt         time.Time  `db:"created_at"`
	ReadAt            *time.Time `db:"read_at"`
}

// CreateInput is the data required to persist one notification.
type CreateInput struct {
	PublicID          string
	RecipientPublicID string
	Kind              string
	Title             string
	Body              string
}

// Cursor is the keyset position in the newest-first order, so a chunk resumes
// after the last row even while new notifications arrive.
type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

// ListQuery windows the feed; a nil Before starts at the newest notification.
type ListQuery struct {
	Limit  int
	Before *Cursor
}

// Repository is the persistence contract owned by the notifications slice.
type Repository interface {
	List(context.Context, string, ListQuery) ([]Notification, error)
	UnreadCount(context.Context, string) (int, error)
	Create(context.Context, CreateInput) (Notification, error)
	MarkRead(context.Context, string, string) (Notification, error)
	MarkAllRead(context.Context, string) (int, error)
}
