package passkeys

type PasskeyRegistrationRequest struct {
	Name string `json:"name"`
}

// PasskeySummary is the public credential representation returned by the feature.
type PasskeySummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

type PasskeysPageProps struct {
	Passkeys []PasskeySummary `json:"passkeys"`
}
