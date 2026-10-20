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
	mu    sync.Mutex
	items map[string]Notification
	order []string
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an isolated-test notification store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: map[string]Notification{}}
}

// Seed inserts one notification into the memory store.
func (r *MemoryRepository) Seed(notification Notification) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if notification.Status == "" {
		notification.Status = "unread"
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now().UTC()
	}
	r.items[notification.PublicID] = notification
	r.order = append(r.order, notification.PublicID)
}

// List returns the newest notifications of the recipient.
func (r *MemoryRepository) List(_ context.Context, recipientPublicID string, limit int) ([]Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]Notification, 0, limit)
	for index := len(r.order) - 1; index >= 0 && len(items) < limit; index-- {
		item := r.items[r.order[index]]
		if item.RecipientPublicID == recipientPublicID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

// UnreadCount returns how many notifications are still unread.
func (r *MemoryRepository) UnreadCount(_ context.Context, recipientPublicID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, item := range r.items {
		if item.RecipientPublicID == recipientPublicID && item.Status == "unread" {
			count++
		}
	}
	return count, nil
}

// Create persists one notification.
func (r *MemoryRepository) Create(_ context.Context, input CreateInput) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	notification := Notification{
		PublicID:          input.PublicID,
		RecipientPublicID: input.RecipientPublicID,
		Kind:              input.Kind,
		Title:             input.Title,
		Body:              input.Body,
		Status:            "unread",
		CreatedAt:         time.Now().UTC(),
	}
	r.items[notification.PublicID] = notification
	r.order = append(r.order, notification.PublicID)
	return notification, nil
}

// MarkRead marks one owned notification as read.
func (r *MemoryRepository) MarkRead(_ context.Context, recipientPublicID, publicID string) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[publicID]
	if !ok || item.RecipientPublicID != recipientPublicID {
		return Notification{}, ErrNotFound
	}
	now := time.Now().UTC()
	item.Status, item.ReadAt = "read", &now
	r.items[publicID] = item
	return item, nil
}

// MarkAllRead marks every unread notification of the recipient as read.
func (r *MemoryRepository) MarkAllRead(_ context.Context, recipientPublicID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	affected := 0
	for id, item := range r.items {
		if item.RecipientPublicID == recipientPublicID && item.Status == "unread" {
			item.Status, item.ReadAt = "read", &now
			r.items[id] = item
			affected++
		}
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
		Status:            "unread",
		CreatedAt:         time.Now().UTC(),
	})
}
