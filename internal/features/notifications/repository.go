package notifications

import (
	"context"
	"fmt"
	"time"
)

// Status is the two states a notification can be in. The schema enforces the
// same set with a CHECK constraint; Go mirrors it so a transition is written
// once instead of compared as a string literal.
type Status string

const (
	StatusUnread Status = "unread"
	StatusRead   Status = "read"
)

func (s Status) valid() bool { return s == StatusUnread || s == StatusRead }

// Event asks a notification to move forward.
type Event string

const (
	EventMarkRead    Event = "mark_read"
	EventMarkAllRead Event = "mark_all_read"
)

// transitionStatus is the only place a notification changes state. Read is
// terminal: marking a read notification is a no-op, never a regression.
func transitionStatus(from Status, event Event) (Status, error) {
	if !from.valid() {
		return "", fmt.Errorf("notification status %q is not a state", from)
	}
	if from == StatusRead {
		return StatusRead, nil
	}
	switch event {
	case EventMarkRead, EventMarkAllRead:
		return StatusRead, nil
	default:
		return "", fmt.Errorf("event %q has no transition from %q", event, from)
	}
}

type Notification struct {
	ID                int64      `db:"id"`
	PublicID          string     `db:"public_id"`
	RecipientPublicID string     `db:"recipient_public_id"`
	Kind              string     `db:"kind"`
	Title             string     `db:"title"`
	Body              string     `db:"body"`
	Status            Status     `db:"status"`
	CreatedAt         time.Time  `db:"created_at"`
	ReadAt            *time.Time `db:"read_at"`
}

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

type Repository interface {
	List(context.Context, string, ListQuery) ([]Notification, error)
	UnreadCount(context.Context, string) (int, error)
	Create(context.Context, CreateInput) (Notification, error)
	MarkRead(context.Context, string, string) (Notification, error)
	MarkAllRead(context.Context, string) (int, error)
}
