package notifications

type NotificationDTO struct {
	PublicID  string `json:"public_id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
	ReadAt    string `json:"read_at,omitempty"`
}

// ListPageProps is one chunk of the notification center; NextCursor is empty on
// the last chunk, which hides the "Load more" control.
type ListPageProps struct {
	Notifications []NotificationDTO `json:"notifications"`
	Unread        int               `json:"unread"`
	NextCursor    string            `json:"next_cursor"`
}
