package notifications

import (
	"context"
	"strconv"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/publicid"
)

const (
	// feedLimit is the dashboard summary size; the center uses the account
	// page size instead.
	feedLimit      = 10
	listLimitMax   = 100
	kindLengthMax  = 60
	titleLengthMax = 200
	bodyLengthMax  = 2000
)

type ServiceDependency struct {
	Repository Repository
}

type Service struct {
	repository Repository
}

func NewService(d ServiceDependency) *Service {
	assert.Assert(d.Repository != nil, "notification repository is required")
	return &Service{repository: d.Repository}
}

// requireRecipient trims and validates the recipient once. The rule lives here
// so four methods cannot drift into four slightly different checks.
func requireRecipient(recipientPublicID string) (string, error) {
	trimmed := strings.TrimSpace(recipientPublicID)
	if trimmed == "" {
		return "", ErrInvalidInput
	}
	return trimmed, nil
}

// Notify creates one notification for a recipient. Other slices use this to
// emit user-facing events without coupling to the HTTP boundary.
func (s *Service) Notify(
	ctx context.Context,
	recipientPublicID, kind, title, body string,
) (NotificationDTO, error) {
	recipientPublicID, err := requireRecipient(recipientPublicID)
	if err != nil {
		return NotificationDTO{}, err
	}
	kind = strings.TrimSpace(kind)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if kind == "" || title == "" || body == "" ||
		len(kind) > kindLengthMax || len(title) > titleLengthMax || len(body) > bodyLengthMax {
		return NotificationDTO{}, ErrInvalidInput
	}
	created, err := s.repository.Create(ctx, CreateInput{
		PublicID:          publicid.MustNew(),
		RecipientPublicID: recipientPublicID,
		Kind:              kind,
		Title:             title,
		Body:              body,
	})
	if err != nil {
		return NotificationDTO{}, err
	}
	return toDTO(created), nil
}

// List returns one chunk of the recipient's notification center. A zero limit
// uses the dashboard feed size; an empty before starts at the newest item.
func (s *Service) List(
	ctx context.Context,
	recipientPublicID, before string,
	limit int,
) (ListPageProps, error) {
	recipientPublicID, err := requireRecipient(recipientPublicID)
	if err != nil {
		return ListPageProps{}, err
	}
	if limit <= 0 {
		limit = feedLimit
	}
	if limit > listLimitMax {
		limit = listLimitMax
	}
	cursor, err := decodeCursor(before)
	if err != nil {
		return ListPageProps{}, err
	}

	// One extra row tells us whether another chunk exists without a count query.
	items, err := s.repository.List(
		ctx,
		recipientPublicID,
		ListQuery{Limit: limit + 1, Before: cursor},
	)
	if err != nil {
		return ListPageProps{}, err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		if len(items) > 0 {
			next = encodeCursor(items[len(items)-1])
		}
	}
	unread, err := s.repository.UnreadCount(ctx, recipientPublicID)
	if err != nil {
		return ListPageProps{}, err
	}
	notifications := make([]NotificationDTO, 0, len(items))
	for _, item := range items {
		notifications = append(notifications, toDTO(item))
	}
	return ListPageProps{Notifications: notifications, Unread: unread, NextCursor: next}, nil
}

// encodeCursor keeps microseconds, the Postgres timestamp precision, so a page
// never repeats or skips a row.
func encodeCursor(notification Notification) string {
	return strconv.FormatInt(notification.CreatedAt.UnixMicro(), 10) + ":" +
		strconv.FormatInt(notification.ID, 10)
}

func decodeCursor(value string) (*Cursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	rawTime, rawID, ok := strings.Cut(value, ":")
	if !ok {
		return nil, ErrInvalidInput
	}
	micros, err := strconv.ParseInt(rawTime, 10, 64)
	if err != nil {
		return nil, ErrInvalidInput
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return nil, ErrInvalidInput
	}
	return &Cursor{CreatedAt: time.UnixMicro(micros).UTC(), ID: id}, nil
}

// MarkRead marks one owned notification as read.
func (s *Service) MarkRead(ctx context.Context, recipientPublicID, publicID string) error {
	recipientPublicID, err := requireRecipient(recipientPublicID)
	if err != nil {
		return err
	}
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return ErrInvalidInput
	}
	if _, err := s.repository.MarkRead(ctx, recipientPublicID, publicID); err != nil {
		return err
	}
	return nil
}

// MarkAllRead marks every unread notification of the recipient as read.
func (s *Service) MarkAllRead(ctx context.Context, recipientPublicID string) error {
	recipientPublicID, err := requireRecipient(recipientPublicID)
	if err != nil {
		return err
	}
	if _, err := s.repository.MarkAllRead(ctx, recipientPublicID); err != nil {
		return err
	}
	return nil
}

func toDTO(notification Notification) NotificationDTO {
	readAt := ""
	if notification.ReadAt != nil {
		readAt = notification.ReadAt.UTC().Format(time.RFC3339)
	}
	return NotificationDTO{
		PublicID:  notification.PublicID,
		Kind:      notification.Kind,
		Title:     notification.Title,
		Body:      notification.Body,
		Read:      notification.Status == StatusRead,
		CreatedAt: notification.CreatedAt.UTC().Format(time.RFC3339),
		ReadAt:    readAt,
	}
}
