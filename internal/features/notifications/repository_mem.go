package notifications

import (
	"context"
	"sort"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu     sync.Mutex
	items  map[string]Notification
	order  []string
	nextID int64
	now    func() time.Time
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository(now func() time.Time) *MemoryRepository {
	assert.Assert(now != nil, "notifications memory repository clock is required")
	return &MemoryRepository{items: map[string]Notification{}, now: now}
}

// Seed inserts one notification into the memory store.
func (r *MemoryRepository) Seed(notification Notification) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if notification.Status == "" {
		notification.Status = StatusUnread
	}
	// Mirror the Postgres timestamp precision: the keyset cursor travels as
	// microseconds, so a finer value here would repeat one row per chunk.
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = r.now().UTC()
	}
	notification.CreatedAt = notification.CreatedAt.UTC().Truncate(time.Microsecond)
	notification.ID = r.nextID + 1
	r.nextID = notification.ID
	r.items[notification.PublicID] = notification
	r.order = append(r.order, notification.PublicID)
}

// List returns one keyset window of the recipient's notifications, newest first.
func (r *MemoryRepository) List(
	_ context.Context,
	recipientPublicID string,
	query ListQuery,
) ([]Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	matching := make([]Notification, 0)
	for _, notification := range r.items {
		if notification.RecipientPublicID != recipientPublicID {
			continue
		}
		if query.Before != nil && !beforeCursor(notification, *query.Before) {
			continue
		}
		matching = append(matching, notification)
	}
	sort.SliceStable(matching, func(i, j int) bool {
		if !matching[i].CreatedAt.Equal(matching[j].CreatedAt) {
			return matching[i].CreatedAt.After(matching[j].CreatedAt)
		}
		return matching[i].ID > matching[j].ID
	})
	return matching[:min(len(matching), query.Limit)], nil
}

// beforeCursor reports whether the notification sorts after the cursor, which is
// exactly the next chunk.
func beforeCursor(notification Notification, cursor Cursor) bool {
	if !notification.CreatedAt.Equal(cursor.CreatedAt) {
		return notification.CreatedAt.Before(cursor.CreatedAt)
	}
	return notification.ID < cursor.ID
}

func (r *MemoryRepository) UnreadCount(_ context.Context, recipientPublicID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, item := range r.items {
		if item.RecipientPublicID == recipientPublicID && item.Status == StatusUnread {
			count++
		}
	}
	return count, nil
}

func (r *MemoryRepository) Create(_ context.Context, input CreateInput) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	notification := Notification{
		ID:                r.nextID + 1,
		PublicID:          input.PublicID,
		RecipientPublicID: input.RecipientPublicID,
		Kind:              input.Kind,
		Title:             input.Title,
		Body:              input.Body,
		Status:            StatusUnread,
		CreatedAt:         r.now().UTC().Truncate(time.Microsecond),
	}
	r.nextID = notification.ID
	r.items[notification.PublicID] = notification
	r.order = append(r.order, notification.PublicID)
	return notification, nil
}

func (r *MemoryRepository) MarkRead(
	_ context.Context,
	recipientPublicID, publicID string,
) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[publicID]
	if !ok || item.RecipientPublicID != recipientPublicID {
		return Notification{}, ErrNotFound
	}
	next, err := transitionStatus(item.Status, EventMarkRead)
	if err != nil {
		return Notification{}, err
	}
	now := r.now().UTC()
	item.Status, item.ReadAt = next, &now
	r.items[publicID] = item
	return item, nil
}

func (r *MemoryRepository) MarkAllRead(_ context.Context, recipientPublicID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now().UTC()
	affected := 0
	for id, item := range r.items {
		if item.RecipientPublicID != recipientPublicID || item.Status != StatusUnread {
			continue
		}
		next, err := transitionStatus(item.Status, EventMarkAllRead)
		if err != nil {
			return 0, err
		}
		item.Status, item.ReadAt = next, &now
		r.items[id] = item
		affected++
	}
	return affected, nil
}

// SeedDefault inserts an unread notification owned by the given recipient.
func SeedDefault(repository *MemoryRepository, recipientPublicID, publicID, title string) {
	assert.Assert(repository != nil, "memory repository is required")

	repository.Seed(Notification{
		PublicID:          publicID,
		RecipientPublicID: recipientPublicID,
		Kind:              "system",
		Title:             title,
		Body:              "Pesan " + title,
		Status:            StatusUnread,
		CreatedAt:         repository.now().UTC(),
	})
}
