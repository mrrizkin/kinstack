package system

// Repository is the explicit content contract for system pages. The system
// feature owns application-provided service-state and help content rather than
// reading a database schema.
type Repository interface {
	Page(surface string) PageResult
	NotFound() NotFoundResult
}

// PageResult is the repository projection for a system page.
type PageResult struct {
	Surface   string               `db:"surface"`
	Topics    []string             `db:"topics"`
	Shortcuts []PageShortcutResult `db:"shortcuts"`
}

// PageShortcutResult is the repository projection for one page shortcut.
type PageShortcutResult struct {
	Key    string `db:"key"`
	Action string `db:"action"`
}

// NotFoundResult is the repository projection for the not-found page.
type NotFoundResult struct {
	Status  int    `db:"status"`
	Message string `db:"message"`
}
