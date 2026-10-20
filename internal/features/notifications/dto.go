package notifications

// NotificationDTO is the public contract for one in-app notification.
type NotificationDTO struct {
	PublicID  string `json:"public_id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
	ReadAt    string `json:"read_at,omitempty"`
}

// ListPageProps is the public contract for the notification center page.
type ListPageProps struct {
	Notifications []NotificationDTO `json:"notifications"`
	Unread        int               `json:"unread"`
}
