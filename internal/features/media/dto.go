package media

// AssetDTO is the public contract for one media asset.
type AssetDTO struct {
	PublicID    string `json:"public_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	CreatedAt   string `json:"created_at"`
}
