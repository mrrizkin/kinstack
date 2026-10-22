package system

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	page     PageResult
	notFound NotFoundResult
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		page: PageResult{
			Topics: []string{
				"Getting started",
				"Account and access",
				"Audit and notifications",
				"Privacy and safety",
			},
			Shortcuts: []PageShortcutResult{
				{Key: "Ctrl K", Action: "Open menu search"},
				{Key: "N", Action: "Open notifications"},
				{Key: "?", Action: "Open help"},
			},
		},
		notFound: NotFoundResult{
			Status:  404,
			Message: "The page you requested does not exist or is no longer available.",
		},
	}
}

func (r *MemoryRepository) Page(surface string) PageResult {
	result := r.page
	result.Surface = surface
	result.Topics = append([]string(nil), r.page.Topics...)
	result.Shortcuts = append([]PageShortcutResult(nil), r.page.Shortcuts...)
	return result
}

func (r *MemoryRepository) NotFound() NotFoundResult {
	return r.notFound
}
