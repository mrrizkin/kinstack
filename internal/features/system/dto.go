package system

// Shortcut is a keyboard shortcut displayed on a system page.
type Shortcut struct {
	Key    string `json:"key"`
	Action string `json:"action"`
}

// PageProps contains the public contract for system state and help pages.
type PageProps struct {
	Surface   string     `json:"surface"`
	Topics    []string   `json:"topics"`
	Shortcuts []Shortcut `json:"shortcuts"`
}

// NotFoundPageProps contains the public not-found page contract.
type NotFoundPageProps struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}
