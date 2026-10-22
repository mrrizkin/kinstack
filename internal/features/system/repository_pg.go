package system

import "kinstack/internal/platform/i18n"

// This feature intentionally has no PostgreSQL adapter. System state and help
// pages are application-owned static content, so production uses this
// explicit no-database repository instead of treating the test memory
// repository as a runtime fallback.
//
// The file retains the canonical repository_pg.go slot so the feature shape is
// uniform while the exception remains visible at the production boundary.
type StaticRepository struct{}

var _ Repository = (*StaticRepository)(nil)

func NewStaticRepository() *StaticRepository {
	return &StaticRepository{}
}

func (r *StaticRepository) Page(surface string) PageResult {
	return systemPageResult(surface)
}

func (r *StaticRepository) NotFound() NotFoundResult {
	return systemNotFoundResult()
}

func systemPageResult(surface string) PageResult {
	return PageResult{
		Surface: surface,
		Topics: []string{
			i18n.T("help.manage_users"),
			i18n.T("help.manage_access"),
			i18n.T("help.trace_audit"),
			i18n.T("help.manage_media"),
			i18n.T("help.account_security"),
			i18n.T("help.api_tokens"),
		},
		Shortcuts: []PageShortcutResult{
			{Key: "Ctrl K", Action: i18n.T("help.search_shortcut")},
			{Key: "N", Action: i18n.T("help.notifications_shortcut")},
			{Key: "Esc", Action: i18n.T("help.close_shortcut")},
		},
	}
}

func systemNotFoundResult() NotFoundResult {
	return NotFoundResult{
		Status:  404,
		Message: "The page you requested does not exist or is no longer available.",
	}
}
